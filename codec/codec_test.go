package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

// Expose only ProtoReflect, without the legacy Reset, String and ProtoMessage methods.
type reflectionOnlyMessage struct{ proto.Message }

func TestCodecsSupportProtobufInterfaces(t *testing.T) {
	for _, tc := range []struct {
		name    string
		codec   encoding.Codec
		encoded []byte
	}{
		{"proto", protoCodec{}, []byte("\x0a\x05hello")},
		{"json", jsonCodec{}, []byte(`"hello"`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.name, tc.codec.Name())
			for _, reflectionOnly := range []bool{false, true} {
				name := "generated"
				if reflectionOnly {
					name = "reflection_only"
				}
				t.Run(name, func(t *testing.T) {
					var source any = wrapperspb.String("hello")
					decoded := &wrapperspb.StringValue{}
					var destination any = decoded
					if reflectionOnly {
						source = reflectionOnlyMessage{source.(proto.Message)}
						destination = reflectionOnlyMessage{decoded}
					}
					data, err := tc.codec.Marshal(source)
					require.NoError(t, err)
					require.Equal(t, tc.encoded, data)
					require.NoError(t, tc.codec.Unmarshal(data, destination))
					require.Equal(t, "hello", decoded.Value)
				})
			}
		})
	}
}

func TestCodecsRejectInvalidData(t *testing.T) {
	for _, c := range []encoding.Codec{protoCodec{}, jsonCodec{}} {
		t.Run(c.Name(), func(t *testing.T) {
			for _, value := range []any{nil, "not a message", struct{}{}} {
				data, err := c.Marshal(value)
				require.ErrorContains(t, err, "want proto.Message")
				require.Nil(t, data)
				require.ErrorContains(t, c.Unmarshal(nil, value), "want proto.Message")
			}
			// Both encodings must propagate protobuf validation errors.
			_, err := c.Marshal(wrapperspb.String(string([]byte{0xff})))
			require.Error(t, err)
			require.Error(t, c.Unmarshal([]byte{0xff}, &wrapperspb.StringValue{}))
		})
	}
}

func TestRegisteredCodecsHandleFramesAndMessages(t *testing.T) {
	for _, name := range []string{"proto", "json"} {
		t.Run(name, func(t *testing.T) {
			c := encoding.GetCodec(name)
			require.NotNil(t, c)
			require.Equal(t, name, c.Name())
			// Raw frames bypass parsing even for the JSON codec.
			frame := &GrpcFrame{Data: []byte{0xff, 0x00, 0x80}}
			data, err := c.Marshal(frame)
			require.NoError(t, err)
			require.Equal(t, frame.Data, data)
			var decodedFrame GrpcFrame
			require.NoError(t, c.Unmarshal(data, &decodedFrame))
			require.Equal(t, frame.Data, decodedFrame.Data)

			data, err = c.Marshal(wrapperspb.String("hello"))
			require.NoError(t, err)
			var decoded wrapperspb.StringValue
			require.NoError(t, c.Unmarshal(data, &decoded))
			require.Equal(t, "hello", decoded.Value)
			_, err = c.Marshal("not a message")
			require.ErrorContains(t, err, "want proto.Message")
			require.Error(t, c.Unmarshal([]byte{0xff}, &decoded))
		})
	}
}
