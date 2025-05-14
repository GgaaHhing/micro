package rpc

import (
	"context"
	"errors"
	"net"
	"reflect"
	"web/micro/rpc/message"
	"web/micro/rpc/serialize"
	"web/micro/rpc/serialize/json"
)

type Server struct {
	services map[string]reflectionStub
	// 不考虑做成这样，因为客户端的可能是确定的，但是一个服务端可能会有多个客户端
	// 所以服务端可能会有多个serialize序列化协议
	// serializer serialize.Serialize
	serializers map[uint8]serialize.Serialize
}

func NewServer() *Server {
	res := &Server{
		services:    make(map[string]reflectionStub, 16),
		serializers: make(map[uint8]serialize.Serialize, 4),
	}
	res.RegisterSerializer(&json.Serializer{})
	return res
}

func (s *Server) RegisterSerializer(sl serialize.Serialize) {
	s.serializers[sl.Code()] = sl
}

func (s *Server) RegisterService(service Service) {
	s.services[service.Name()] = reflectionStub{
		s:           service,
		serializers: s.serializers,
		value:       reflect.ValueOf(service),
	}
}

func (s *Server) Start(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			if er := s.handleConn(conn); er != nil {
				_ = conn.Close()
			}
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	for {
		// 读取响应
		resBs, err := ReadMsg(conn)
		if err != nil {
			return err
		}

		req := message.DecodeReq(resBs)
		if err != nil {
			return err
		}

		resp, err := s.Invoke(context.Background(), req)
		if err != nil {
			resp.Error = []byte(err.Error())
			// 不return，只要连接还正常就继续通信
			//return nil
		}
		resp.CalculateHeadLength()
		resp.CalculateBodyLength()
		data := message.EncodeResp(resp)
		// 回写请求
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		if n != len(resBs) {
			return errors.New("micro: 没写完数据")
		}
	}
}

func (s *Server) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	// 调用指定方法
	service, ok := s.services[req.ServiceName]
	resp := &message.Response{
		RequestId:  req.RequestId,
		Version:    req.Version,
		Compresser: req.Compresser,
		Serializer: req.Serializer,
	}

	if !ok {
		return resp, errors.New("rpc: 要调用的方法不存在")
	}

	respData, err := service.invoke(ctx, req)
	if err != nil {
		return resp, err
	}
	resp.Data = respData
	return resp, nil
}

// reflectionStub 的出现是为了防止，如果以后可能会需要使用到unsafe
type reflectionStub struct {
	s Service
	// 我们也不知道reflection要用哪个，所以全部传下来
	//serializer serialize.Serialize
	serializers map[uint8]serialize.Serialize
	value       reflect.Value
}

func (s *reflectionStub) invoke(ctx context.Context, req *message.Request) ([]byte, error) {
	method := s.value.MethodByName(req.MethodName)
	in := make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(context.Background())

	inReq := reflect.New(method.Type().In(1).Elem())
	serializer, ok := s.serializers[req.Serializer]
	if !ok {
		return nil, errors.New("micro: 不支持的序列化协议")
	}
	err := serializer.Decode(req.Data, inReq.Interface())
	if err != nil {
		return nil, err
	}
	in[1] = inReq
	results := method.Call(in)

	if results[1].Interface() != nil {
		err = results[1].Interface().(error)
	}
	var res []byte
	if results[0].IsNil() {
		return nil, err
	} else {
		var er error
		res, er = serializer.Encode(results[0].Interface())
		if er != nil {
			return nil, er
		}
	}
	return res, nil
}
