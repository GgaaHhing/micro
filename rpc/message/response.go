package message

import (
	"encoding/binary"
)

type Response struct {
	HeadLength uint32 //4字节
	BodyLength uint32
	RequestId  uint32
	Version    uint8
	// 压缩算法
	Compresser uint8
	// 序列化协议
	Serializer uint8
	Error      []byte
	Data       []byte
}

func (resp *Response) CalculateHeadLength() {
	header := 15 + len(resp.Error)
	resp.HeadLength = uint32(header)
}

func (resp *Response) CalculateBodyLength() {
	resp.BodyLength = uint32(len(resp.Data))
}

func EncodeResp(resp *Response) []byte {
	bs := make([]byte, resp.HeadLength+resp.BodyLength)
	// 写入头部长度
	binary.BigEndian.PutUint32(bs, resp.HeadLength)
	// 写入body长度
	binary.BigEndian.PutUint32(bs[4:8], resp.BodyLength)
	//
	binary.BigEndian.PutUint32(bs[8:12], resp.RequestId)
	// 1字节，直接写入就行
	bs[12] = resp.Version
	bs[13] = resp.Compresser
	bs[14] = resp.Serializer
	// 对于不定长的，我们使用copy
	cur := bs[15:]

	copy(cur, resp.Error)

	cur = cur[len(resp.Error):]

	//Data
	copy(cur, resp.Data)
	return bs
}

func DecodeResp(data []byte) *Response {
	resp := new(Response)
	resp.HeadLength = binary.BigEndian.Uint32(data[:4])
	resp.BodyLength = binary.BigEndian.Uint32(data[4:8])
	resp.RequestId = binary.BigEndian.Uint32(data[8:12])
	resp.Version = data[12]
	resp.Compresser = data[13]
	resp.Serializer = data[14]

	if resp.HeadLength > 15 {
		resp.Error = data[15:resp.HeadLength]
	}

	if resp.BodyLength != 0 {
		resp.Data = data[resp.HeadLength:]
	}
	return resp
}
