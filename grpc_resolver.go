package micro

import (
	"context"
	"google.golang.org/grpc/resolver"
	"time"
	"web/micro/registry"
)

type grpcResolverBuilder struct {
	r registry.Registry
}

func NewRegistryBuilder(r registry.Registry) (resolver.Builder, error) {
	return &grpcResolverBuilder{
		r: r,
	}, nil

}

func (b *grpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &grpcResolver{
		cc:     cc,
		r:      b.r,
		target: target,
	}
	r.resolve()
	go r.watch()
	return r, nil
}

func (b *grpcResolverBuilder) Scheme() string {
	return "registry"
}

type grpcResolver struct {
	r       registry.Registry
	cc      resolver.ClientConn
	target  resolver.Target
	timeout time.Duration
	close   chan struct{}
}

func (g *grpcResolver) ResolveNow(options resolver.ResolveNowOptions) {
	g.resolve()
}

// watch 接收注册中心传来的通知
func (g *grpcResolver) watch() {
	events, err := g.r.Subscribe(g.target.Endpoint())
	if err != nil {
		g.cc.ReportError(err)
		return
	}
	select {
	case <-events:
		g.resolve()
	case <-g.close:
		return
	}
}

func (g *grpcResolver) resolve() {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	instances, err := g.r.ListServices(ctx, g.target.Endpoint())
	if err != nil {
		g.cc.ReportError(err)
		return
	}
	address := make([]resolver.Address, 0, len(instances))
	for _, instance := range instances {
		address = append(address, resolver.Address{
			Addr:       instance.Address,
			ServerName: instance.Name,
		})
	}
	// 更改可用的节点
	err = g.cc.UpdateState(resolver.State{
		Addresses: address,
	})
	if err != nil {
		g.cc.ReportError(err)
		return
	}
}

func (g *grpcResolver) Close() {
	// close 发送一个关闭的信号给某个通道
	close(g.close)
}
