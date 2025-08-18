package codec

import (
	"fmt"

	"google.golang.org/grpc/encoding"
	_ "google.golang.org/grpc/encoding/gzip"
	_ "google.golang.org/grpc/encoding/proto" // ensure the default "proto" codec is registered first
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

func init() {
	encoding.RegisterCodec(BytesCodec("proto", &protoCodec{}))
	encoding.RegisterCodec(BytesCodec("json", nil))
}

func BytesCodec(name string, fallback encoding.Codec) encoding.Codec {
	return bytesCodec{name: name, fallback: fallback}
}

type bytesCodec struct {
	name     string
	fallback encoding.Codec
}

type GrpcFrame struct {
	Data []byte
}

func (s bytesCodec) Marshal(v any) ([]byte, error) {
	if m, ok := v.(*GrpcFrame); ok {
		return m.Data, nil
	}

	if s.fallback == nil {
		return nil, fmt.Errorf("object %+v not of type codec.GrpcFrame", v)
	}

	return s.fallback.Marshal(v)
}

func (s bytesCodec) Unmarshal(data []byte, v any) error {
	if m, ok := v.(*GrpcFrame); ok {
		m.Data = data
		return nil
	}

	if s.fallback == nil {
		return fmt.Errorf("object %+v not of type codec.GrpcFrame", v)
	}

	return s.fallback.Unmarshal(data, v)
}

func (s bytesCodec) Name() string {
	return s.name
}

// protoCodec is a Codec implementation with protobuf. It is the default rawCodec for gRPC.
type protoCodec struct{}

func (protoCodec) Name() string {
	return "proto"
}

func (protoCodec) Marshal(v any) ([]byte, error) {
	vv := messageV2Of(v)
	if vv == nil {
		return nil, fmt.Errorf("failed to marshal, message is %T, want proto.Message", v)
	}

	return proto.Marshal(vv)
}

func (protoCodec) Unmarshal(data []byte, v any) error {
	vv := messageV2Of(v)
	if vv == nil {
		return fmt.Errorf("failed to unmarshal, message is %T, want proto.Message", v)
	}

	return proto.Unmarshal(data, vv)
}

func messageV2Of(v any) proto.Message {
	switch v := v.(type) {
	case protoadapt.MessageV1:
		return protoadapt.MessageV2Of(v)
	case protoadapt.MessageV2:
		return v
	}

	return nil
}
