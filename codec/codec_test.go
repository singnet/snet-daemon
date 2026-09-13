package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestBytesCodec_MarshalUnmarshal_GrpcFrame(t *testing.T) {
	c := BytesCodec("test", nil)
	frame := &GrpcFrame{Data: []byte("hello")}

	data, err := c.Marshal(frame)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), data)

	var frame2 GrpcFrame
	err = c.Unmarshal(data, &frame2)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), frame2.Data)
}

func TestBytesCodec_MarshalUnmarshal_InvalidType(t *testing.T) {
	c := BytesCodec("json", nil)
	_, err := c.Marshal("string")
	require.Error(t, err)

	err = c.Unmarshal([]byte("data"), "string")
	require.Error(t, err)
}

func TestBytesCodec_Name(t *testing.T) {
	c := BytesCodec("json", nil)
	require.Equal(t, "json", c.Name())
}

func TestBytesCodec_MarshalUnmarshal_FallbackProto(t *testing.T) {
	protoMsg := &anypb.Any{Value: []byte("test")}
	c := BytesCodec("proto", &protoCodec{})

	data, err := c.Marshal(protoMsg)
	require.NoError(t, err)

	var msg anypb.Any
	err = c.Unmarshal(data, &msg)
	require.NoError(t, err)
	require.Equal(t, protoMsg.Value, msg.Value)
}

func TestProtoCodec_MarshalUnmarshal(t *testing.T) {
	c := &protoCodec{}
	msg := &anypb.Any{Value: []byte("data")}

	data, err := c.Marshal(msg)
	require.NoError(t, err)

	var msg2 anypb.Any
	err = c.Unmarshal(data, &msg2)
	require.NoError(t, err)
	require.Equal(t, msg.Value, msg2.Value)
}

func TestProtoCodec_MarshalUnmarshal_Invalid(t *testing.T) {
	c := &protoCodec{}
	_, err := c.Marshal("not a proto")
	require.Error(t, err)

	err = c.Unmarshal([]byte("data"), "not a proto")
	require.Error(t, err)
}
