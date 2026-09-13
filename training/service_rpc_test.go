package training

import (
	"context"
	"errors"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/ctxkeys"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Use memory storage and a fixed block; do not start the constructor's background workers.
func trainingFixture(t *testing.T) *DaemonService {
	t.Helper()
	old := config.Vip()
	t.Cleanup(func() { config.SetVip(old) })
	v := viper.New()
	v.Set(config.OrganizationId, "YOUR_ORG_ID")
	v.Set(config.ServiceId, "YOUR_SERVICE_ID")
	v.Set(config.DaemonGroupName, "default_group")
	v.Set(config.MaxMessageSizeInMB, 4)
	config.SetVip(v)
	org, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	require.NoError(t, err)
	mem := storage.NewMemStorage()
	ds := &DaemonService{
		organizationMetaData: org, blockchain: &coverageBlockchain{}, allowBlockDifference: 5,
		storage: NewModelStorage(mem, org), userStorage: NewUserModelStorage(mem, org),
		pendingStorage: NewPendingModelStorage(mem, org), publicStorage: NewPublicModelStorage(mem, org),
	}
	for _, id := range []string{"owned", "foreign"} {
		owner := testUserAddress
		if id == "foreign" {
			owner = "other-owner"
		}
		require.NoError(t, ds.storage.Put(ds.storage.buildModelKey(id), &ModelData{
			ModelId: id, ModelName: "original", Description: "original description", Status: Status_CREATED,
			CreatedByAddress: owner, AuthorizedAddresses: []string{owner},
			GRPCServiceName: "Example", GRPCMethodName: "Predict", ValidatePrice: 10, TrainPrice: 20,
		}))
	}
	require.NoError(t, ds.userStorage.Put(ds.userStorage.buildModelUserKey(testUserAddress), &ModelUserData{ModelIds: []string{"owned"}}))
	return ds
}

type coverageBlockchain struct {
	blockchain.Processor
	check func(*big.Int, uint64) error
}

func (b *coverageBlockchain) CompareWithLatestBlockNumber(block *big.Int, allowance uint64) error {
	if b.check != nil {
		return b.check(block, allowance)
	}
	return nil
}

func trainingContext(t *testing.T, method string) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return context.WithValue(ctx, ctxkeys.MethodKey, "/training.Daemon/"+method)
}

func trainingAuth(method string) *AuthorizationDetails {
	return createTestAuthDetails(big.NewInt(1000), method)
}

type coverageProvider struct {
	UnimplementedModelServer
	uploads chan []*UploadInput
}

func (p *coverageProvider) UploadAndValidate(stream Model_UploadAndValidateServer) error {
	var chunks []*UploadInput
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		chunks = append(chunks, chunk)
	}
	p.uploads <- chunks
	return stream.SendAndClose(&StatusResponse{Status: Status_VALIDATING})
}

func (*coverageProvider) ValidateModelPrice(_ context.Context, req *ValidateRequest) (*PriceInBaseUnit, error) {
	if req.ModelId != "owned" || req.TrainingDataLink != "dataset" {
		return nil, status.Error(codes.InvalidArgument, "unexpected validation input")
	}
	return &PriceInBaseUnit{Price: 31}, nil
}

func (*coverageProvider) TrainModelPrice(_ context.Context, req *ModelID) (*PriceInBaseUnit, error) {
	if req.ModelId != "owned" {
		return nil, status.Error(codes.InvalidArgument, "unexpected model")
	}
	return &PriceInBaseUnit{Price: 47}, nil
}

func (*coverageProvider) ValidateModel(_ context.Context, req *ValidateRequest) (*StatusResponse, error) {
	if req.ModelId != "owned" || req.TrainingDataLink != "dataset" {
		return nil, status.Error(codes.InvalidArgument, "unexpected validation input")
	}
	return &StatusResponse{Status: Status_VALIDATING}, nil
}

func (*coverageProvider) TrainModel(_ context.Context, req *ModelID) (*StatusResponse, error) {
	if req.ModelId != "owned" {
		return nil, status.Error(codes.InvalidArgument, "unexpected model")
	}
	return &StatusResponse{Status: Status_TRAINING}, nil
}

func (*coverageProvider) DeleteModel(_ context.Context, req *ModelID) (*StatusResponse, error) {
	if req.ModelId != "owned" {
		return nil, status.Error(codes.InvalidArgument, "unexpected model")
	}
	return &StatusResponse{Status: Status_DELETED}, nil
}

func startCoverageProvider(t *testing.T, ds *DaemonService, fail bool) *coverageProvider {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if fail {
			return nil, status.Error(codes.Unavailable, "provider unavailable")
		}
		return handler(ctx, req)
	}))
	provider := &coverageProvider{uploads: make(chan []*UploadInput, 1)}
	RegisterModelServer(server, provider)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-done) })
	ds.serviceUrl = "http://" + listener.Addr().String()
	return provider
}

func TestTrainingRPCRejectsInvalidRequests(t *testing.T) {
	for _, method := range []string{"validate_model_price", "validate_model", "train_model_price", "train_model", "delete_model", "update_model"} {
		for _, scenario := range []string{"no authorization", "missing method", "bad signature", "foreign owner"} {
			t.Run(method+"/"+scenario, func(t *testing.T) {
				ds := trainingFixture(t)
				auth, id := trainingAuth(method), "owned"
				ctx := trainingContext(t, method)
				want := ErrAccessToModel
				switch scenario {
				case "no authorization":
					auth = nil
					want = ErrNoAuthorization
				case "missing method":
					ctx = context.Background()
					want = ErrBadAuthorization
				case "bad signature":
					auth.Signature = []byte("invalid")
				case "foreign owner":
					id = "foreign"
				}
				_, err := invokeTrainingRPC(ds, ctx, method, auth, id)
				require.ErrorIs(t, err, want)
				model, err := ds.storage.GetModel("owned")
				require.NoError(t, err)
				require.Equal(t, Status_CREATED, model.Status)
				require.Equal(t, uint64(10), model.ValidatePrice)
			})
		}
	}
}

func invokeTrainingRPC(ds *DaemonService, ctx context.Context, method string, auth *AuthorizationDetails, id string) (any, error) {
	common := &CommonRequest{Authorization: auth, ModelId: id}
	validate := &AuthValidateRequest{Authorization: auth, ModelId: id, TrainingDataLink: "dataset"}
	switch method {
	case "validate_model_price":
		return ds.ValidateModelPrice(ctx, validate)
	case "validate_model":
		return ds.ValidateModel(ctx, validate)
	case "train_model_price":
		return ds.TrainModelPrice(ctx, common)
	case "train_model":
		return ds.TrainModel(ctx, common)
	case "delete_model":
		return ds.DeleteModel(ctx, common)
	default:
		return ds.UpdateModel(ctx, &UpdateModelRequest{Authorization: auth, ModelId: id})
	}
}

func TestTrainingRPCProviderErrors(t *testing.T) {
	for _, method := range []string{"validate_model_price", "validate_model", "train_model_price", "train_model", "delete_model"} {
		t.Run(method, func(t *testing.T) {
			ds := trainingFixture(t)
			startCoverageProvider(t, ds, true)
			_, err := invokeTrainingRPC(ds, trainingContext(t, method), method, trainingAuth(method), "owned")
			require.Error(t, err)
			model, err := ds.storage.GetModel("owned")
			require.NoError(t, err)
			require.Equal(t, Status_CREATED, model.Status)
			require.Equal(t, uint64(10), model.ValidatePrice)
			require.Equal(t, uint64(20), model.TrainPrice)
		})
	}
}

func TestTrainingPricesPersistIndependently(t *testing.T) {
	ds := trainingFixture(t)
	startCoverageProvider(t, ds, false)
	for _, method := range []string{"validate_model_price", "train_model_price"} {
		response, err := invokeTrainingRPC(ds, trainingContext(t, method), method, trainingAuth(method), "owned")
		require.NoError(t, err)
		model, err := ds.storage.GetModel("owned")
		require.NoError(t, err)
		require.Equal(t, uint64(31), model.ValidatePrice)
		if method == "validate_model_price" {
			require.Equal(t, uint64(31), response.(*PriceInBaseUnit).Price)
			require.Equal(t, uint64(20), model.TrainPrice)
		} else {
			require.Equal(t, uint64(47), response.(*PriceInBaseUnit).Price)
			require.Equal(t, uint64(47), model.TrainPrice)
		}
	}
	require.Error(t, ds.updateModelPrices("missing", &PriceInBaseUnit{Price: 1}, nil))
}

type notifyingStorage struct {
	storage.TypedAtomicStorage
	done chan struct{}
}

func (s *notifyingStorage) ExecuteTransaction(req storage.TypedCASRequest) (bool, error) {
	defer close(s.done)
	return s.TypedAtomicStorage.ExecuteTransaction(req)
}

func TestTrainingStartsPendingModel(t *testing.T) {
	for _, method := range []string{"validate_model", "train_model"} {
		t.Run(method, func(t *testing.T) {
			ds := trainingFixture(t)
			startCoverageProvider(t, ds, false)
			done := make(chan struct{})
			ds.pendingStorage.delegate = &notifyingStorage{TypedAtomicStorage: ds.pendingStorage.delegate, done: done}
			response, err := invokeTrainingRPC(ds, trainingContext(t, method), method, trainingAuth(method), "owned")
			require.NoError(t, err)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("pending model update did not finish")
			}
			want := Status_VALIDATING
			if method == "train_model" {
				want = Status_TRAINING
			}
			require.Equal(t, want, response.(*StatusResponse).Status)
			model, err := ds.storage.GetModel("owned")
			require.NoError(t, err)
			require.Equal(t, want, model.Status)
			if method == "validate_model" {
				require.Equal(t, "dataset", model.TrainingLink)
			}
			pending, ok, err := ds.pendingStorage.Get(ds.pendingStorage.buildPendingModelKey())
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, []string{"owned"}, pending.ModelIDs)
		})
	}
}

func TestTrainingUpdateModelDetailsAndAccess(t *testing.T) {
	ds := trainingFixture(t)
	startCoverageProvider(t, ds, false)
	name, description := "renamed", "updated description"
	newUser := "0x0000000000000000000000000000000000000001"
	response, err := ds.UpdateModel(trainingContext(t, "update_model"), &UpdateModelRequest{
		Authorization: trainingAuth("update_model"), ModelId: "owned", ModelName: &name, Description: &description,
		AddressList: []string{testUserAddress, newUser},
	})
	require.NoError(t, err)
	require.Equal(t, name, response.Name)
	require.Equal(t, description, response.Description)
	for _, address := range []string{testUserAddress, newUser} {
		user, ok, err := ds.userStorage.Get(ds.userStorage.buildModelUserKey(address))
		require.NoError(t, err)
		require.True(t, ok)
		require.Contains(t, user.ModelIds, "owned")
	}
}

func TestTrainingDeleteModelAndAccess(t *testing.T) {
	ds := trainingFixture(t)
	startCoverageProvider(t, ds, false)
	deleted, err := ds.DeleteModel(trainingContext(t, "delete_model"), &CommonRequest{Authorization: trainingAuth("delete_model"), ModelId: "owned"})
	require.NoError(t, err)
	require.Equal(t, Status_DELETED, deleted.Status)
	model, err := ds.storage.GetModel("owned")
	require.NoError(t, err)
	require.Equal(t, Status_DELETED, model.Status)
	for _, address := range []string{testUserAddress} {
		user, ok, err := ds.userStorage.Get(ds.userStorage.buildModelUserKey(address))
		require.NoError(t, err)
		require.True(t, ok)
		require.NotContains(t, user.ModelIds, "owned")
	}
}

func TestTrainingMetadataLookup(t *testing.T) {
	ds := trainingFixture(t)
	want := &MethodMetadata{DatasetType: "images", MaxModelsPerUser: 3}
	ds.methodsMetadata = map[string]*MethodMetadata{"ExamplePredict": want}
	ds.trainingMetadata = &TrainingMetadata{TrainingEnabled: true, TrainingInProto: true}
	md, err := ds.GetTrainingMetadata(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, ds.trainingMetadata, md)
	for _, req := range []*MethodMetadataRequest{{ModelId: "owned"}, {GrpcServiceName: "Example", GrpcMethodName: "Predict"}} {
		md, err := ds.GetMethodMetadata(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, want, md)
	}
	mdMethod, err := ds.GetMethodMetadata(context.Background(), &MethodMetadataRequest{GrpcServiceName: "unknown"})
	require.NoError(t, err)
	require.Nil(t, mdMethod)
}

func TestTrainingSignatureBlockWindow(t *testing.T) {
	for _, tc := range []struct {
		method, message string
		allowance       uint64
	}{
		{"get_model", "unified", 600}, {"validate_model_price", "unified", 600},
		{"train_model", "train_model", 5}, {"get_model", "GET_MODEL", 5},
	} {
		t.Run(tc.method+"/"+tc.message, func(t *testing.T) {
			ds := trainingFixture(t)
			calls := 0
			stale := errors.New("stale block")
			ds.blockchain = &coverageBlockchain{check: func(block *big.Int, allowance uint64) error {
				calls++
				require.Equal(t, uint64(1000), block.Uint64())
				require.Equal(t, tc.allowance, allowance)
				return stale
			}}
			require.ErrorIs(t, ds.verifySignature(trainingAuth(tc.message), "/training.Daemon/"+tc.method), stale)
			require.Equal(t, 1, calls)
		})
	}
	ds := trainingFixture(t)
	require.ErrorContains(t, ds.verifySignature(trainingAuth("unified"), "train_model"), "unsupported message")
	require.ErrorContains(t, ds.verifySignature(trainingAuth("get_model"), 42), "invalid method")
	require.NoError(t, ds.verifyCreatedByAddress("owned", strings.ToUpper(testUserAddress)))
	require.ErrorIs(t, ds.verifyCreatedByAddress("foreign", testUserAddress), ErrNotOwnerModel)
	require.ErrorIs(t, ds.verifyCreatedByAddress("missing", testUserAddress), ErrGetModelStorage)
	require.Nil(t, WrapError(nil, "context"))
	require.ErrorIs(t, WrapError(ErrGetModelStorage, "details"), ErrDaemonStorage)
}

type coverageUploadStream struct {
	grpc.ServerStream
	requests []*UploadAndValidateRequest
	response *StatusResponse
	sendErr  error
}

func (s *coverageUploadStream) Recv() (*UploadAndValidateRequest, error) {
	if len(s.requests) == 0 {
		return nil, io.EOF
	}
	req := s.requests[0]
	s.requests = s.requests[1:]
	return req, nil
}

func (s *coverageUploadStream) SendAndClose(response *StatusResponse) error {
	s.response = response
	return s.sendErr
}

func TestTrainingUploadForwardsChunksAndTracksModel(t *testing.T) {
	for _, failClose := range []bool{false, true} {
		name := "success"
		if failClose {
			name = "client close error"
		}
		t.Run(name, func(t *testing.T) {
			ds := trainingFixture(t)
			provider := startCoverageProvider(t, ds, false)
			done := make(chan struct{})
			ds.pendingStorage.delegate = &notifyingStorage{TypedAtomicStorage: ds.pendingStorage.delegate, done: done}
			stream := &coverageUploadStream{}
			if failClose {
				stream.sendErr = errors.New("client disconnected")
			}
			for i, data := range []string{"first", "second"} {
				stream.requests = append(stream.requests, &UploadAndValidateRequest{
					Authorization: trainingAuth("upload_and_validate"),
					UploadInput: &UploadInput{ModelId: "owned", FileName: "dataset.txt", Data: []byte(data),
						FileSize: 11, BatchNumber: uint64(i), BatchCount: 2},
				})
			}
			err := ds.UploadAndValidate(stream)
			require.ErrorIs(t, err, stream.sendErr)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("pending update did not finish")
			}
			require.NotNil(t, stream.response)
			require.Equal(t, Status_VALIDATING, stream.response.Status)
			chunks := <-provider.uploads
			require.Len(t, chunks, 2)
			for i, data := range []string{"first", "second"} {
				require.Equal(t, []byte(data), chunks[i].Data)
				require.Equal(t, "owned", chunks[i].ModelId)
				require.Equal(t, "dataset.txt", chunks[i].FileName)
				require.Equal(t, uint64(11), chunks[i].FileSize)
				require.Equal(t, uint64(i), chunks[i].BatchNumber)
				require.Equal(t, uint64(2), chunks[i].BatchCount)
			}
			model, err := ds.storage.GetModel("owned")
			require.NoError(t, err)
			require.Equal(t, Status_VALIDATING, model.Status)
			pending, ok, err := ds.pendingStorage.Get(ds.pendingStorage.buildPendingModelKey())
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, []string{"owned"}, pending.ModelIDs)
		})
	}
}
