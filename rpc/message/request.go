package message

import (
	"bytes"
	"encoding/binary"
)

// Request RPC协议请求头定义
/*
	我们把req分为协议头和协议体，协议头用来存放req除Data外的数据
	协议体用来存放Data，且都是不定长
*/
type Request struct {
	HeadLength uint32 //4字节
	BodyLength uint32
	RequestId  uint32
	Version    uint8
	// 压缩算法
	Compresser uint8
	// 序列化协议：用来标识客户端和服务端用的是什么序列化方法
	Serializer  uint8
	ServiceName string
	MethodName  string
	// 拓展字段，用于传输自定义数据，比如traceId等等
	// any在编解码的过程很难搞，所以不选择
	Meta map[string]string
	Data []byte
}

func (req *Request) CalculateHeadLength() {
	headLength := 15 + len(req.ServiceName) + 1 + len(req.MethodName) + 1
	for key, value := range req.Meta {
		headLength += len(key)
		headLength++
		headLength += len(value)
		headLength++
	}
	req.HeadLength = uint32(headLength)
}

func (req *Request) CalculateBodyLength() {
	req.BodyLength = uint32(len(req.Data))
}

func EncodeReq(req *Request) []byte {
	bs := make([]byte, req.HeadLength+req.BodyLength)
	// 写入头部长度
	binary.BigEndian.PutUint32(bs, req.HeadLength)
	// 写入body长度
	binary.BigEndian.PutUint32(bs[4:8], req.BodyLength)
	//
	binary.BigEndian.PutUint32(bs[8:12], req.RequestId)
	// 1字节，直接写入就行
	bs[12] = req.Version
	bs[13] = req.Compresser
	bs[14] = req.Serializer
	// 对于不定长的，我们使用copy
	cur := bs[15:]
	copy(cur, req.ServiceName)

	cur = cur[len(req.ServiceName):]
	cur[0] = '\n'

	cur = cur[1:]
	copy(cur[:len(req.MethodName)], req.MethodName)

	cur = cur[len(req.MethodName):]
	cur[0] = '\n'

	// Meta
	cur = cur[1:]
	for key, val := range req.Meta {
		copy(cur, key)
		cur = cur[len(key):]
		cur[0] = '\r'
		cur = cur[1:]
		copy(cur, val)

		cur = cur[len(val):]
		cur[0] = '\n'

		cur = cur[1:]
	}

	//Data
	copy(cur, req.Data)
	return bs
}

func DecodeReq(data []byte) *Request {
	req := new(Request)
	req.HeadLength = binary.BigEndian.Uint32(data[:4])
	req.BodyLength = binary.BigEndian.Uint32(data[4:8])
	req.RequestId = binary.BigEndian.Uint32(data[8:12])
	req.Version = data[12]
	req.Compresser = data[13]
	req.Serializer = data[14]

	// 为了解决不定长内容，所以我们需要引入分隔符
	//req.ServiceName = string(data[15:])
	// 将header和data分隔开
	header := data[15:req.HeadLength]
	// MethodName的前面
	index := bytes.IndexByte(header, '\n')
	req.ServiceName = string(header[:index])

	header = header[index+1:]
	// Meta的数据的前面
	index = bytes.IndexByte(header, '\n')
	req.MethodName = string(header[:index])

	header = header[index+1:]
	// Meta的第一个数据后面
	index = bytes.IndexByte(header, '\n')
	if index != -1 {
		meta := make(map[string]string, 4)
		// xxServiceName\nMethodName\nMeta:xxx\rXxx\n
		for index != -1 {
			pair := header[:index]
			pairIndex := bytes.IndexByte(pair, '\r')
			key := string(pair[:pairIndex])
			value := string(pair[pairIndex+1:])
			meta[key] = value

			header = header[index+1:]
			index = bytes.IndexByte(header, '\n')
		}
		req.Meta = meta
	}

	if req.BodyLength != 0 {
		req.Data = data[req.HeadLength:]
	}
	return req
}
