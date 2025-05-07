package net

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Pool TODO:
type Pool struct {
	// 空闲连接队列
	idlesConns chan *idleConn
	// 请求等待队列
	reqQueue []connReq
	// 最大连接数
	maxCnt int
	// 当前连接数
	cnt int
	// 最大空闲时间
	maxIdleTime time.Duration
	//初始化连接数量
	initCnt int
	// 初始化连接
	factory func() (net.Conn, error)
	// 锁
	lock sync.Mutex
}

func NewPool(initCnt int, maxIdleCnt int, maxCnt int, maxIdleTime time.Duration,
	factory func() (net.Conn, error)) (*Pool, error) {
	idlesConns := make(chan *idleConn, maxIdleCnt)
	if initCnt > maxIdleCnt {
		return nil, fmt.Errorf("初始化连接数量 %d 不能大于 最大空闲连接数量 %d", initCnt, maxIdleCnt)
	}
	// factory要提前建起来
	for i := 0; i < initCnt; i++ {
		conn, err := factory()
		if err != nil {
			return nil, err
		}
		idlesConns <- &idleConn{
			c:              conn,
			lastActiveTime: time.Now(),
		}
	}

	res := &Pool{
		idlesConns:  idlesConns,
		maxCnt:      maxCnt,
		maxIdleTime: maxIdleTime,
		cnt:         0,
		initCnt:     initCnt,
		factory:     factory,
	}
	return res, nil
}

func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 为什么用for循环，是因为如果连接因为某些原因关闭了，我们取下一个空闲连接
	for {
		select {
		// 拿到了空闲连接
		case ic := <-p.idlesConns:
			// 上一次使用时间+最大空闲连接比现在的时间少，说明空闲了很久
			if ic.lastActiveTime.Add(p.maxIdleTime).Before(time.Now()) {
				_ = ic.c.Close()
				continue
			}
			return ic.c, nil
		//没有空闲连接
		default:
			//
			p.lock.Lock()
			// 超过上限
			if p.cnt >= p.maxCnt {
				// 使用的连接大于最大连接数，我们需要将后续请求加入等待队列
				req := connReq{connChan: make(chan net.Conn, 1)}
				p.reqQueue = append(p.reqQueue, req)
				// 进入select之前解锁，防止select中阻塞造成死锁
				p.lock.Unlock()
				select {
				case <-ctx.Done():
					go func() {
						c := <-req.connChan
						_ = p.Put(context.Background(), c)
					}()
				// 等别人归还
				case c := <-req.connChan:
					return c, nil
				}
			}
			// 又没有超出上限，也没有空闲连接，说明我们该创建一个连接了
			c, err := p.factory()
			if err != nil {
				p.lock.Unlock()
				return nil, err
			}
			p.cnt++
			p.lock.Unlock()
			return c, nil
		}
	}
}

func (p *Pool) Put(ctx context.Context, c net.Conn) error {
	p.lock.Lock()
	// 如果队列>0 说明有等待的，我们之间把接下来的请求放入队列中
	if len(p.reqQueue) > 0 {
		req := p.reqQueue[0]
		p.reqQueue = p.reqQueue[1:]
		p.lock.Unlock()
		req.connChan <- c
		return nil
	}
	p.lock.Unlock()
	// 没有阻塞的请求, 我们就创建一个连接，然后给他
	ic := &idleConn{
		c:              c,
		lastActiveTime: time.Now(),
	}
	select {
	case p.idlesConns <- ic:
	default:
		// 空闲队列满了
		_ = c.Close()
		//p.lock.Lock()
		p.cnt--
		//p.lock.Unlock()
	}
	return nil
}

type idleConn struct {
	c net.Conn
	// 上一次使用的时间
	lastActiveTime time.Time
}

type connReq struct {
	connChan chan net.Conn
}
