package codec

import (
	"fmt"
	"google.golang.org/grpc/encoding"
	_ "google.golang.org/protobuf/encoding/protojson" // ensure default "proto" codec is registered first
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

func (s bytesCodec) Marshal(v interface{}) ([]byte, error) {
	if m, ok := v.(*GrpcFrame); ok {
		return m.Data, nil
	}

	if s.fallback == nil {
		return nil, fmt.Errorf("object %+v not of type codec.GrpcFrame", v)
	}

	return s.fallback.Marshal(v)
}

func (s bytesCodec) Unmarshal(data []byte, v interface{}) error {
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

/*
Copied from https://github.com/mwitkow/grpc-proxy/blob/67591eb23c48346a480470e462289835d96f70da/proxy/codec.go#L57
Original Copyright 2017 Michal Witkowski. All Rights Reserved. See LICENSE-GRPC-PROXY for licensing terms.
Modifications Copyright 2018 SingularityNET Foundation. All Rights Reserved. See LICENSE for licensing terms.
*/
// protoCodec is a Codec implementation with protobuf. It is the default rawCodec for gRPC.
type protoCodec struct{}

func (protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(protoadapt.MessageV2Of(v.(protoadapt.MessageV1)))
}

func (protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, protoadapt.MessageV2Of(v.(protoadapt.MessageV1)))
}

func (protoCodec) Name() string {
	return "proto"
}
