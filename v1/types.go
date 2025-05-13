package rpc

import "context"

// Service 让用户实现Service达到用户自定义名称的目的
type Service interface {
	Name() string
}

type Proxy interface {
	Invoke(ctx context.Context, req *Request) (*Response, error)
}

type Request struct {
	ServiceName string
	MethodName  string
	Arg         []byte
}

type Response struct {
	Data []byte
}
