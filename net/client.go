package net

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func Connect(network, addr string) error {
	conn, err := net.DialTimeout(network, addr, time.Second*3)
	defer conn.Close()
	if err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		_, err := conn.Write([]byte("hello"))
		if err != nil {
			return err
		}

		res := make([]byte, 128)
		_, err = conn.Read(res)
		if err != nil {
			return err
		}
		fmt.Println(res)
	}
	return nil
}

type Client struct {
	network string
	addr    string
}

func (c *Client) Send(data string) (string, error) {
	conn, err := net.DialTimeout(c.network, c.addr, time.Second*3)
	defer conn.Close()
	if err != nil {
		return "", err
	}
	reqLen := len(data)
	// 构造响应长度
	req := make([]byte, reqLen+numOfLengthBytes)

	// 第一步：把长度写入到前8个字节里
	binary.BigEndian.PutUint64(req[:numOfLengthBytes], uint64(reqLen))
	// 第二步：写入数据
	copy(req[numOfLengthBytes:], data)

	// 发送请求
	_, err = conn.Write(req)
	if err != nil {
		return "", err
	}

	// 读取响应的数据
	lenBs := make([]byte, numOfLengthBytes)
	_, err = conn.Read(lenBs)
	if err != nil {
		return "", err
	}

	// 读取响应的数据大小
	length := binary.BigEndian.Uint64(lenBs)
	respBs := make([]byte, length)
	_, err = conn.Read(respBs)
	if err != nil {
		return "", err
	}

	return string(respBs), nil
}
