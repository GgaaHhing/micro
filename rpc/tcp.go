package rpc

import (
	"encoding/binary"
	"net"
)

// ReadMsg 重构：从Server和Client中抽取而得
func ReadMsg(conn net.Conn) ([]byte, error) {
	lenBs := make([]byte, numOfLengthBytes)
	// 先读8字节，读取长度，获取字节大小
	_, err := conn.Read(lenBs)
	if err != nil {
		return nil, err
	}

	// 获取头部长度
	headerLength := binary.BigEndian.Uint32(lenBs[:4])
	// 获取协议体长度
	bodyLength := binary.BigEndian.Uint32(lenBs[4:])
	// 总长度
	length := headerLength + bodyLength

	// 读取响应的数据大小
	data := make([]byte, length)
	copy(data[:8], lenBs)
	_, err = conn.Read(data[8:])
	return data, err
}

//// EncodeMsg 编码
//func EncodeMsg(data []byte) []byte {
//	reqLen := len(data)
//	// 构造响应长度
//	res := make([]byte, reqLen+numOfLengthBytes)
//
//	// 第一步：把长度写入到前8个字节里
//	binary.BigEndian.PutUint64(res[:numOfLengthBytes], uint64(reqLen))
//	// 第二步：写入数据
//	copy(res[numOfLengthBytes:], data)
//
//	return res
//}
