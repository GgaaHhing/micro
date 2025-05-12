package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"time"
)

const numOfLengthBytes = 8

// InitClientProxy 要为函数类型的字段赋值
// type service struct{ GetById func() }
func InitClientProxy(addr string, service Service) error {
	client, err := NewClient(addr)
	if err != nil {
		return err
	}
	return setFuncField(service, client)
}

func setFuncField(service Service, p Proxy) error {
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
				reqData, err := json.Marshal(args[1])
				if err != nil {
					return []reflect.Value{retVal, reflect.ValueOf(err)}
				}
				req := &Request{
					ServiceName: service.Name(),
					MethodName:  fieldTyp.Name,
					Arg:         reqData,
				}

				// 关键就是这里，这里才是发起rpc调用的方法
				resp, err := p.Invoke(ctx, req)
				if err != nil {
					// Out 返回函数类型的第 i 个输出参数的类型。
					// err 在reflect的零值
					return []reflect.Value{retVal, reflect.ValueOf(err)}
				}
				// TODO 处理响应
				err = json.Unmarshal(resp.Data, retVal.Interface())
				if err != nil {
					return nil
				}
				return []reflect.Value{retVal, reflect.Zero(reflect.TypeOf(new(error)).Elem())}
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
	pool pool.Pool
}

func NewClient(addr string) (*Client, error) {
	p, err := pool.NewChannelPool(&pool.Config{
		InitialCap: 1,
		MaxCap:     30,
		MaxIdle:    10,
		Factory: func() (interface{}, error) {
			return net.DialTimeout("tcp", addr, time.Second*3), nil
		},
		Close: func(obj interface{}) error {
			return obj.(net.Conn).Close()
		},
		IdleTimeout: time.Second * 60,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		pool: p,
	}, nil
}

// Invoke 发送请求给服务端并调用方法，最终获取返回值
// 把一段二进制编码的调用信息发送给服务端
func (c *Client) Invoke(ctx context.Context, req *Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	// 接下来发送给服务端
	// 服务端需要提供一个连接
	res, err := c.Send(data)
	if err != nil {
		return nil, err
	}
	return &Response{
		Data: res,
	}, nil
}

func (c *Client) Send(data []byte) ([]byte, error) {
	val, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	conn := val.(net.Conn)
	// 编码数据
	req := EncodeMsg(data)

	// 发送请求
	_, err = conn.Write(req)
	if err != nil {
		return nil, err
	}
	// 读取响应的数据
	return ReadMsg(conn)
}
