package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	clientV3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"sync"
	"web/micro/registry"
)

type Registry struct {
	c       *clientV3.Client
	sess    *concurrency.Session
	cancels []func()
	mutex   sync.Mutex
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
	// clientV3.WithPrefix()：按照前缀匹配
	getResp, err := r.c.Get(ctx, r.serviceKey(serviceName), clientV3.WithPrefix())
	if err != nil {
		return nil, err
	}

	res := make([]registry.ServiceInstance, 0, len(getResp.Kvs))
	for _, kv := range getResp.Kvs {
		var si registry.ServiceInstance
		// registry的时候用json编码放进，现在要解码
		err = json.Unmarshal(kv.Value, &si)
		if err != nil {
			return nil, err
		}
		res = append(res, si)
	}
	return res, nil
}

// Subscribe 说是订阅，其实类似心跳一样的机制
func (r *Registry) Subscribe(serviceName string) (<-chan registry.Event, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r.mutex.Lock()
	r.cancels = append(r.cancels, cancel)
	r.mutex.Unlock()
	// WithRequireLeader：只有在集群中有leader的时候才去拿数据
	ctx = clientV3.WithRequireLeader(ctx)
	watchResp := r.c.Watch(ctx, r.serviceKey(serviceName), clientV3.WithPrefix())
	res := make(chan registry.Event)
	go func() {
		for {
			select {
			case resp := <-watchResp:
				if resp.Canceled {
					return
				}
				if resp.Err() != nil {
					return
				}
				for range resp.Events {
					res <- registry.Event{}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return res, nil
}

func (r *Registry) Close() error {
	r.mutex.Lock()
	cancels := r.cancels
	r.cancels = nil
	r.mutex.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	return r.sess.Close()
}

func (r *Registry) instanceKey(si registry.ServiceInstance) string {
	return fmt.Sprintf("/micro/%s/%s", si.Name, si.Address)
}

func (r *Registry) serviceKey(serviceName string) string {
	return fmt.Sprintf("/micro/%s", serviceName)
}
