package codec

import (
	"fmt"
	"google.golang.org/grpc/encoding"
)

func init() {
	encoding.RegisterCodec(BytesCodec("proto"))
	encoding.RegisterCodec(BytesCodec("json"))
}

func BytesCodec(name string) encoding.Codec {
	return bytesCodec{name: name}
}

type bytesCodec struct {
	name string
}

type GrpcFrame struct {
	Data []byte
}

func (s bytesCodec) Marshal(v interface{}) ([]byte, error) {
	if m, ok := v.(*GrpcFrame); ok {
		return m.Data, nil
	}
	return nil, fmt.Errorf("object %+v not of type codec.GrpcFrame", v)
}

func (s bytesCodec) Unmarshal(data []byte, v interface{}) error {
	if m, ok := v.(*GrpcFrame); ok {
		m.Data = data
		return nil
	}
	return fmt.Errorf("object %+v not of type codec.GrpcFrame", v)
}

func (s bytesCodec) Name() string {
	return s.name
}
