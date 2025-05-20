package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	clientV3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"web/micro/registry"
)

type Registry struct {
	c    *clientV3.Client
	sess *concurrency.Session
}

func NewRegistry(c *clientV3.Client) (*Registry, error) {
	sess, err := concurrency.NewSession(c)
	if err != nil {
		return nil, err
	}
	return &Registry{
		c:    c,
		sess: sess,
	}, nil
}

func (r *Registry) Register(ctx context.Context, si registry.ServiceInstance) error {
	val, err := json.Marshal(si)
	if err != nil {
		return err
	}
	// option的地方传入租约
	_, err = r.c.Put(ctx, r.instanceKey(si), string(val), clientV3.WithLease(r.sess.Lease()))
	return err
}

func (r *Registry) UnRegister(ctx context.Context, si registry.ServiceInstance) error {
	_, err := r.c.Delete(ctx, r.instanceKey(si))
	return err
}

func (r *Registry) ListServices(ctx context.Context, serviceName string) ([]registry.ServiceInstance, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) Subscribe(serviceName string) (<-chan registry.Event, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) Close() error {
	return r.sess.Close()
}

func (r *Registry) instanceKey(si registry.ServiceInstance) string {
	return fmt.Sprintf("/micro/%s/%s", si.Name, si.Address)
}

func (r *Registry) serviceKey(si registry.ServiceInstance) string {
	return fmt.Sprintf("/micro/%s", si.Name)
}
