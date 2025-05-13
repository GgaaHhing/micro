package message

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnDecode(t *testing.T) {
	testCases := []struct {
		name string
		req  *Request
	}{
		{
			name: "normal",
			req: &Request{
				RequestId:   123,
				Version:     12,
				Compresser:  13,
				Serializer:  14,
				ServiceName: "user-service",
				MethodName:  "GetById",
				Meta: map[string]string{
					"trace-id": "123456",
					"a/b":      "a",
				},
				Data: []byte("hello world"),
			},
		}, {
			name: "data with \n",
			req: &Request{
				RequestId:   123,
				Version:     12,
				Compresser:  13,
				Serializer:  14,
				ServiceName: "user-service",
				MethodName:  "GetById",
				Meta: map[string]string{
					"trace-id": "123456",
					"a/b":      "a",
				},
				Data: []byte("hello \n world"),
			},
		},
		{
			name: "no meta",
			req: &Request{
				RequestId:   123,
				Version:     12,
				Compresser:  13,
				Serializer:  14,
				ServiceName: "user-service",
				MethodName:  "GetById",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.calculateBodyLength()
			tc.req.calculateHeadLength()
			// 对称过程，可以这样进行测试
			data := EncodeReq(tc.req)
			req := DecodeReq(data)
			assert.Equal(t, tc.req, req)
		})
	}
}

func (req *Request) calculateHeadLength() {
	headLength := 15 + len(req.ServiceName) + 1 + len(req.MethodName) + 1
	for key, value := range req.Meta {
		headLength += len(key)
		headLength++
		headLength += len(value)
		headLength++
	}
	req.HeadLength = uint32(headLength)
}

func (req *Request) calculateBodyLength() {
	req.BodyLength = uint32(len(req.Data))
}
