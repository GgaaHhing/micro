package message

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
