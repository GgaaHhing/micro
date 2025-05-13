package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
)

type Server struct {
	services map[string]reflectionStub
}

func NewServer() *Server {
	return &Server{
		services: make(map[string]reflectionStub, 16),
	}
}

func (s *Server) RegisterService(service Service) {
	s.services[service.Name()] = reflectionStub{
		s:     service,
		value: reflect.ValueOf(service),
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

		req := &Request{}
		err = json.Unmarshal(resBs, req)
		if err != nil {
			return err
		}

		resp, err := s.Invoke(context.Background(), req)
		if err != nil {
			return err
		}

		// 构造响应
		res := EncodeMsg(resp.Data)

		// 回写请求
		n, err := conn.Write(res)
		if err != nil {
			return err
		}
		if n != len(resBs) {
			return errors.New("micro: 没写完数据")
		}
	}
}

func (s *Server) Invoke(ctx context.Context, req *Request) (*Response, error) {
	// 调用指定方法
	service, ok := s.services[req.ServiceName]
	if !ok {
		return nil, errors.New("rpc: 要调用的方法不存在")
	}

	resp, err := service.invoke(ctx, req.MethodName, req.Arg)
	if err != nil {
		return nil, err
	}

	return &Response{
		Data: resp,
	}, nil
}

// reflectionStub 的出现是为了防止，如果以后可能会需要使用到unsafe
type reflectionStub struct {
	s     Service
	value reflect.Value
}

func (s *reflectionStub) invoke(ctx context.Context, methodName string, data []byte) ([]byte, error) {
	method := s.value.MethodByName(methodName)
	in := make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(context.Background())

	inReq := reflect.New(method.Type().In(1).Elem())
	err := json.Unmarshal(data, inReq.Interface())
	if err != nil {
		return nil, err
	}
	in[1] = inReq

	res := method.Call(in)
	if res[1].Interface() != nil {
		return nil, res[1].Interface().(error)
	}

	return json.Marshal(res[0].Interface())
}
