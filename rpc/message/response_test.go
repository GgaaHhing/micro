package message

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRespEnDecode(t *testing.T) {
	testCases := []struct {
		name string
		resp *Response
	}{
		{
			name: "normal",
			resp: &Response{
				RequestId:  123,
				Version:    12,
				Compresser: 13,
				Serializer: 14,
				Data:       []byte("hello world"),
			},
		},
		{
			name: "error",
			resp: &Response{
				RequestId:  123,
				Version:    12,
				Compresser: 13,
				Serializer: 14,
				Error:      []byte("Error"),
				Data:       []byte("hello world"),
			},
		},
		{
			name: "no data",
			resp: &Response{
				RequestId:  123,
				Version:    12,
				Compresser: 13,
				Serializer: 14,
				Error:      []byte("Error"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.resp.CalculateBodyLength()
			tc.resp.CalculateHeadLength()
			// 对称过程，可以这样进行测试
			data := EncodeResp(tc.resp)
			resp := DecodeResp(data)
			assert.Equal(t, tc.resp, resp)
		})
	}
}
