package rpc

import (
	"context"
	"errors"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"strconv"
	"time"
	"web/micro/rpc/message"
	"web/micro/rpc/serialize"
	"web/micro/rpc/serialize/json"
)

const numOfLengthBytes = 8

// InitService 要为函数类型的字段赋值
// type service struct{ GetById func() }
func (c *Client) InitService(service Service) error {
	return setFuncField(service, c, c.serializer)
}

func setFuncField(service Service, p Proxy, s serialize.Serialize) error {
	if service == nil {
		return errors.New("rpc: 不支持 nil")
	}
	val := reflect.ValueOf(service)
	typ := val.Type()
	// 只支持指向结构体的一级指针
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return errors.New("rpc: 只支持指向结构体的一级指针")
	}
	/*
		- ValueOf 获取的 val：
		- 指针时：指向结构体的指针的反射值
		- Elem() 后：结构体本身的反射值

		- Type 获取的 typ：
		- 指针时：指向结构体的指针的类型信息
		- Elem() 后：结构体本身的类型信息

		- Field 获取的内容：
		- typ.Field ：字段的类型信息（名称、类型等）
		- val.Field ：字段的值信息（可读写）
	*/
	val = val.Elem()
	typ = typ.Elem()

	//返回结构体类型的字段数量
	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fieldTyp := typ.Field(i)
		fieldVal := val.Field(i)
		// 要调用canSet，看是否可以修改
		if fieldVal.CanSet() {
			// 这个地方才是真正的发起RPC调用的地方
			fn := func(args []reflect.Value) []reflect.Value {

				ctx := args[0].Interface().(context.Context)
				retVal := reflect.New(fieldTyp.Type.Out(0).Elem())
				reqData, err := s.Encode(args[1].Interface())
				if err != nil {
					return []reflect.Value{retVal, reflect.ValueOf(err)}
				}
				meta := make(map[string]string, 2)
				if deadline, ok := ctx.Deadline(); ok {
					// 毫秒数，十进制
					meta["deadline"] = strconv.FormatInt(deadline.UnixMilli(), 10)
				}

				if isOneway(ctx) {
					meta = map[string]string{"one-way": "true"}
				}
				req := &message.Request{
					Serializer:  s.Code(),
					ServiceName: service.Name(),
					MethodName:  fieldTyp.Name,
					Meta:        meta,
					Data:        reqData,
				}

				req.CalculateHeadLength()
				req.CalculateBodyLength()

				// 关键就是这里，这里才是发起rpc调用的方法
				resp, err := p.Invoke(ctx, req)
				if err != nil {
					// Out 返回函数类型的第 i 个输出参数的类型。
					// err 在reflect的零值
					return []reflect.Value{retVal, reflect.ValueOf(err)}
				}

				var retErr error
				if len(resp.Error) > 0 {
					// 服务端出现error 可以考虑返回，也可以考虑继续执行
					retErr = errors.New(string(resp.Error))
				}

				// TODO 处理响应
				if len(resp.Data) > 0 {
					err = s.Decode(resp.Data, retVal.Interface())
					if err != nil {
						return nil
					}
				}

				var retErrVal reflect.Value
				if retErr == nil {
					retErrVal = reflect.Zero(reflect.TypeOf(new(error)).Elem())
				} else {
					retErrVal = reflect.ValueOf(retErr)
				}

				return []reflect.Value{retVal, retErrVal}
			}
			// 设置值给 GetById
			fnVal := reflect.MakeFunc(fieldTyp.Type, fn)
			// 这个Set就是篡改成对RPC发起调用的方法
			fieldVal.Set(fnVal)
		}
	}
	return nil
}

type Client struct {
	// 重构了，使用连接池
	//addr string
	// 也可以考虑使用连接池
	pool       pool.Pool
	serializer serialize.Serialize
}

type ClientOption func(*Client)

func NewClient(addr string, opts ...ClientOption) (*Client, error) {
	p, err := pool.NewChannelPool(&pool.Config{
		InitialCap: 1,
		MaxCap:     30,
		MaxIdle:    10,
		Factory: func() (interface{}, error) {
			conn, err := net.DialTimeout("tcp", addr, time.Second*3)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
		Close: func(obj interface{}) error {
			return obj.(net.Conn).Close()
		},
		IdleTimeout: time.Second * 60,
	})
	if err != nil {
		return nil, err
	}
	res := &Client{
		pool:       p,
		serializer: &json.Serializer{},
	}

	for _, opt := range opts {
		opt(res)
	}
	return res, nil
}

func ClientWithSerializer(s serialize.Serialize) ClientOption {
	return func(c *Client) {
		c.serializer = s
	}
}

// Invoke 发送请求给服务端并调用方法，最终获取返回值
// 把一段二进制编码的调用信息发送给服务端
func (c *Client) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	if ctx.Done() != nil {
		return nil, ctx.Err()
	}

	ch := make(chan struct{})
	defer close(ch)
	var (
		resp *message.Response
		err  error
	)
	go func() {
		resp, err = c.doInvoke(ctx, req)
		ch <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		return resp, err
	}
}

func (c *Client) doInvoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	data := message.EncodeReq(req)
	// 接下来发送给服务端
	// 服务端需要提供一个连接
	res, err := c.send(ctx, data)
	if err != nil {
		return nil, err
	}
	return message.DecodeResp(res), nil
}

func (c *Client) send(ctx context.Context, data []byte) ([]byte, error) {
	val, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	conn := val.(net.Conn)

	// 发送请求
	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}
	if isOneway(ctx) {
		return nil, errors.New("micro: 这是一个oneway调用，不应检测结果")
	}
	// 读取响应的数据
	return ReadMsg(conn)
}
