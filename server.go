package micro

import (
	"context"
	"google.golang.org/grpc"
	"net"
	"time"
	"web/micro/registry"
)

type Server struct {
	name            string
	registry        registry.Registry
	registerTimeout time.Duration
	*grpc.Server
	// 在close方法里，我们会关闭，所以要在这里维持
	listener net.Listener
}

type ServerOption func(*Server)

func NewServer(name string, opts ...ServerOption) (*Server, error) {
	res := &Server{
		name:            name,
		Server:          grpc.NewServer(),
		registerTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(res)
	}
	return res, nil
}

func ServerWithRegistry(reg registry.Registry) ServerOption {
	return func(s *Server) {
		s.registry = reg
	}
}

func ServerWithRegisterTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.registerTimeout = timeout
	}
}

// Start 当用户调用Start的时候，就意味着服务已经准备完成，开始注册
func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = listener

	if s.registry != nil {
		ctx, cancel := context.WithTimeout(context.Background(), s.registerTimeout)
		defer cancel()
		r := registry.ServiceInstance{
			Name:    s.name,
			Address: listener.Addr().String(),
		}
		err = s.registry.Register(ctx, r)
		if err != nil {
			return err
		}

		defer func() {
			_ = s.registry.Close()
		}()
	}

	err = s.Serve(listener)
	return err
}

func (s *Server) Close() error {
	if s.registry != nil {
		err := s.registry.Close()
		if err != nil {
			return err
		}
	}
	s.GracefulStop()
	return nil
}
