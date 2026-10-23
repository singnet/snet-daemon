package freecall

import (
	"context"
	_ "embed"
	"net"
	"testing"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// This is the original schema from dev before the freecall package was extracted.
// Keep its historical comments and definitions unchanged as a compatibility
// fixture; current protocol documentation belongs in state_service.proto.
//
//go:embed testdata/legacy_state_service.proto
var legacyStateProto string

func legacyStateDescriptor(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	compiler := protocompile.Compiler{Resolver: &protocompile.SourceResolver{
		Accessor: protocompile.SourceAccessorFromMap(map[string]string{"state_service.proto": legacyStateProto}),
	}}
	files, err := compiler.Compile(context.Background(), "state_service.proto")
	require.NoError(t, err)
	return files[0]
}

func TestLegacyProtobufContract(t *testing.T) {
	legacy := legacyStateDescriptor(t)
	current := File_freecall_state_service_proto
	require.Equal(t, legacy.Package(), current.Package())
	for i := 0; i < current.Messages().Len(); i++ {
		message := current.Messages().Get(i)
		previous := legacy.Messages().ByName(message.Name())
		require.NotNil(t, previous)
		require.True(t, proto.Equal(protodesc.ToDescriptorProto(previous), protodesc.ToDescriptorProto(message)),
			"message changed: %s", message.FullName())
	}
	require.True(t, proto.Equal(
		protodesc.ToServiceDescriptorProto(legacy.Services().ByName("FreeCallStateService")),
		protodesc.ToServiceDescriptorProto(current.Services().ByName("FreeCallStateService"))))
}

func (suite *FreeCallStateServiceSuite) TestLegacyRPCClient() {
	t := suite.T()
	legacy := legacyStateDescriptor(t)
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	RegisterFreeCallStateServiceServer(server, suite.stateService)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	client, err := grpc.NewClient("passthrough:///legacy-freecall",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	suite.newTokenRequest = &GetFreeCallTokenRequest{Address: suite.userAddress.Hex(), CurrentBlock: 99}
	suite.SetSignatureForNewToken()
	request := dynamicpb.NewMessage(legacy.Messages().ByName("GetFreeCallTokenRequest"))
	fields := request.Descriptor().Fields()
	request.Set(fields.ByName("address"), protoreflect.ValueOfString(suite.newTokenRequest.Address))
	request.Set(fields.ByName("signature"), protoreflect.ValueOfBytes(suite.newTokenRequest.Signature))
	request.Set(fields.ByName("current_block"), protoreflect.ValueOfUint64(99))
	response := dynamicpb.NewMessage(legacy.Messages().ByName("FreeCallToken"))
	require.NoError(t, client.Invoke(ctx, "/escrow.FreeCallStateService/GetFreeCallToken", request, response))
	token := response.Get(response.Descriptor().Fields().ByName("token")).Bytes()
	require.NotEmpty(t, token)

	suite.useFreeCallRequest = &FreeCallStateRequest{
		Address: suite.userAddress.Hex(), CurrentBlock: 99, FreeCallToken: token,
	}
	suite.SetSignatureWithToken()
	request = dynamicpb.NewMessage(legacy.Messages().ByName("FreeCallStateRequest"))
	fields = request.Descriptor().Fields()
	request.Set(fields.ByName("address"), protoreflect.ValueOfString(suite.useFreeCallRequest.Address))
	request.Set(fields.ByName("signature"), protoreflect.ValueOfBytes(suite.useFreeCallRequest.Signature))
	request.Set(fields.ByName("current_block"), protoreflect.ValueOfUint64(99))
	request.Set(fields.ByName("free_call_token"), protoreflect.ValueOfBytes(token))
	response = dynamicpb.NewMessage(legacy.Messages().ByName("FreeCallStateReply"))
	require.NoError(t, client.Invoke(ctx, "/escrow.FreeCallStateService/GetFreeCallsAvailable", request, response))
	require.Equal(t, uint64(10), response.Get(response.Descriptor().Fields().ByName("free_calls_available")).Uint())
}
