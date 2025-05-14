package serialize

type Serialize interface {
	// Code 用一个字节来表示序列化协议
	Code() uint8
	Encode(val any) ([]byte, error)
	// Decode val: 应该是一个结构体指针
	Decode(data []byte, val any) error
}
