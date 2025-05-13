package rpc

import (
	"context"
	"web/micro/rpc/message"
)

// Service 让用户实现Service达到用户自定义名称的目的
type Service interface {
	Name() string
}

type Proxy interface {
	Invoke(ctx context.Context, req *message.Request) (*message.Response, error)
}
