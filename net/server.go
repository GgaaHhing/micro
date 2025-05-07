package net

import (
	"encoding/binary"
	"errors"
	"net"
)

const numOfLengthBytes = 8

func Serve(network, addr string) error {
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
			if er := handleConn(conn); er != nil {
				_ = conn.Close()
			}
		}()
	}
}

func handleConn(conn net.Conn) error {
	for {
		bs := make([]byte, numOfLengthBytes)
		_, err := conn.Read(bs)
		if err != nil {
			return err
		}

		res := handleMsg(bs)
		n, err := conn.Write(res)
		if err != nil {
			return err
		}
		if n != len(res) {
			return errors.New("micro: 没写完数据")
		}
	}
}

func handleMsg(req []byte) []byte {
	res := make([]byte, len(req)*2)
	copy(res[:len(req)], req)
	copy(res[len(req):], req)
	return res
}

type Server struct {
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
		lenBs := make([]byte, numOfLengthBytes)
		_, err := conn.Read(lenBs)
		if err != nil {
			return err
		}

		// 通过解码获取数据长度，这就是简单的协议，双方都规定使用这个编解码
		length := binary.BigEndian.Uint64(lenBs)
		resBs := make([]byte, length)
		_, err = conn.Read(resBs)
		if err != nil {
			return err
		}
		// 处理数据
		respData := handleMsg(lenBs)
		// 获取数据大小
		respLen := len(respData)
		// 构造响应长度
		res := make([]byte, respLen+numOfLengthBytes)

		// 第一步：把长度写入到前8个字节里
		binary.BigEndian.PutUint64(res[:numOfLengthBytes], uint64(respLen))
		// 第二步：写入数据
		copy(res[numOfLengthBytes:], respData)

		// 回写请求
		n, err := conn.Write(resBs)
		if err != nil {
			return err
		}
		if n != len(resBs) {
			return errors.New("micro: 没写完数据")
		}
	}
}
