package rpc

import (
	"encoding/binary"
	"net"
)

// ReadMsg 重构：从Server和Client中抽取而得
func ReadMsg(conn net.Conn) ([]byte, error) {
	lenBs := make([]byte, numOfLengthBytes)
	// 读取长度，获取字节大小
	_, err := conn.Read(lenBs)
	if err != nil {
		return nil, err
	}

	// 读取响应的数据大小
	// 用大顶堆来解码读取到的数据
	length := binary.BigEndian.Uint64(lenBs)
	data := make([]byte, length)
	_, err = conn.Read(data)
	return data, err
}

func EncodeMsg(data []byte) []byte {
	reqLen := len(data)
	// 构造响应长度
	res := make([]byte, reqLen+numOfLengthBytes)

	// 第一步：把长度写入到前8个字节里
	binary.BigEndian.PutUint64(res[:numOfLengthBytes], uint64(reqLen))
	// 第二步：写入数据
	copy(res[numOfLengthBytes:], data)

	return res
}
