package training

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceFailureStorage struct {
	storage.TypedAtomicStorage
	getErr, putErr, transactionErr error
}

func (s *serviceFailureStorage) Get(key any) (any, bool, error) {
	if s.getErr != nil {
		return nil, false, s.getErr
	}
	return s.TypedAtomicStorage.Get(key)
}

func (s *serviceFailureStorage) Put(key, value any) error {
	if s.putErr != nil {
		return s.putErr
	}
	return s.TypedAtomicStorage.Put(key, value)
}

func (s *serviceFailureStorage) ExecuteTransaction(request storage.TypedCASRequest) (bool, error) {
	if s.transactionErr != nil {
		return false, s.transactionErr
	}
	return s.TypedAtomicStorage.ExecuteTransaction(request)
}

type trainingErrorProvider struct {
	UnimplementedModelServer
	modelID     string
	createErr   error
	modelStatus Status
	statusErr   error
	uploadErr   error
}

func (p *trainingErrorProvider) CreateModel(context.Context, *NewModel) (*ModelID, error) {
	return &ModelID{ModelId: p.modelID}, p.createErr
}

func (p *trainingErrorProvider) GetModelStatus(context.Context, *ModelID) (*StatusResponse, error) {
	return &StatusResponse{Status: p.modelStatus}, p.statusErr
}

func (p *trainingErrorProvider) UploadAndValidate(stream Model_UploadAndValidateServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return p.uploadErr
		}
		if err != nil {
			return err
		}
	}
}

func startTrainingErrorProvider(t *testing.T, ds *DaemonService, provider *trainingErrorProvider) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	RegisterModelServer(server, provider)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { require.NoError(t, stopCoverageProvider(server, done)) })
	ds.serviceUrl = "http://" + listener.Addr().String()
}

func TestTrainingCreateModelFailures(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, scenario := range []string{"provider", "empty model ID", "model storage", "user storage"} {
		t.Run(scenario, func(t *testing.T) {
			ds := trainingFixture(t)
			provider := &trainingErrorProvider{modelID: "new-model"}
			want := ErrDaemonStorage
			switch scenario {
			case "provider":
				provider.createErr = status.Error(codes.Unavailable, "provider unavailable")
				want = ErrServiceInvocation
			case "empty model ID":
				provider.modelID = ""
				want = ErrEmptyModelID
			case "model storage":
				ds.storage.delegate = &serviceFailureStorage{TypedAtomicStorage: ds.storage.delegate, putErr: failure}
			case "user storage":
				ds.userStorage.delegate = &serviceFailureStorage{TypedAtomicStorage: ds.userStorage.delegate, putErr: failure}
			}
			startTrainingErrorProvider(t, ds, provider)
			_, err := ds.CreateModel(trainingContext(t, "create_model"), &NewModelRequest{
				Authorization: trainingAuth("create_model"),
				Model:         &NewModel{Name: "new", GrpcServiceName: "Example", GrpcMethodName: "Predict"},
			})
			require.ErrorIs(t, err, want)
			if scenario == "model storage" || scenario == "user storage" {
				require.ErrorContains(t, err, failure.Error())
			}
			index, ok, readErr := ds.userStorage.Get(ds.userStorage.buildModelUserKey(testUserAddress))
			require.NoError(t, readErr)
			require.True(t, ok)
			require.Equal(t, []string{"owned"}, index.ModelIds, "failed creation must not add a user index entry")
		})
	}
}

type trainingReceiveErrorStream struct {
	*coverageUploadStream
	recvErr error
}

func (s *trainingReceiveErrorStream) Recv() (*UploadAndValidateRequest, error) { return nil, s.recvErr }

func TestTrainingUploadRejectsInvalidChunks(t *testing.T) {
	for _, scenario := range []string{"missing authorization", "bad signature", "foreign owner", "receive error"} {
		t.Run(scenario, func(t *testing.T) {
			ds := trainingFixture(t)
			startCoverageProvider(t, ds, false)
			request := &UploadAndValidateRequest{
				Authorization: trainingAuth("upload_and_validate"), UploadInput: &UploadInput{ModelId: "owned", Data: []byte("data")},
			}
			want := ErrNoAuthorization
			switch scenario {
			case "missing authorization":
				request.Authorization = nil
			case "bad signature":
				request.Authorization.Signature = []byte("invalid")
			case "foreign owner":
				request.UploadInput.ModelId = "foreign"
				want = ErrAccessToModel
			}
			stream := &coverageUploadStream{requests: []*UploadAndValidateRequest{request}}
			var err error
			if scenario == "receive error" {
				want = errors.New("client disconnected while uploading")
				err = ds.UploadAndValidate(&trainingReceiveErrorStream{coverageUploadStream: stream, recvErr: want})
			} else {
				err = ds.UploadAndValidate(stream)
			}
			require.ErrorIs(t, err, want)
			require.Nil(t, stream.response)
			model, readErr := ds.storage.GetModel("owned")
			require.NoError(t, readErr)
			require.Equal(t, Status_CREATED, model.Status)
			_, pending, readErr := ds.pendingStorage.Get(ds.pendingStorage.buildPendingModelKey())
			require.NoError(t, readErr)
			require.False(t, pending)
		})
	}
}

func TestTrainingStatusWorkerHandlesFailures(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, scenario := range []string{"read error", "missing model", "provider error", "unchanged", "write error", "pending error", "completed"} {
		t.Run(scenario, func(t *testing.T) {
			ds := trainingFixture(t)
			provider := &trainingErrorProvider{modelStatus: Status_CREATED}
			key := ds.pendingStorage.buildPendingModelKey()
			require.NoError(t, ds.pendingStorage.AddPendingModelId(key, "owned"))
			id := "owned"
			switch scenario {
			case "read error":
				ds.storage.delegate = &serviceFailureStorage{TypedAtomicStorage: ds.storage.delegate, getErr: failure}
			case "missing model":
				id = "missing"
			case "provider error":
				provider.statusErr = status.Error(codes.Unavailable, "provider unavailable")
			case "write error":
				provider.modelStatus = Status_TRAINING
				ds.storage.delegate = &serviceFailureStorage{TypedAtomicStorage: ds.storage.delegate, putErr: failure}
			case "pending error":
				ds.pendingStorage.delegate = &serviceFailureStorage{TypedAtomicStorage: ds.pendingStorage.delegate, transactionErr: failure}
			case "completed":
				provider.modelStatus = Status_VALIDATED
			}
			startTrainingErrorProvider(t, ds, provider)
			ds.startUpdateModelStatusWorker(trainingContext(t, "get_model"), id)
			if scenario == "read error" {
				ds.storage.delegate = ds.storage.delegate.(*serviceFailureStorage).TypedAtomicStorage
			}
			model, err := ds.storage.GetModel("owned")
			require.NoError(t, err)
			wantStatus := Status_CREATED
			if scenario == "completed" {
				wantStatus = Status_VALIDATED
			}
			require.Equal(t, wantStatus, model.Status)
			pending, _, err := ds.pendingStorage.Get(key)
			require.NoError(t, err)
			if scenario == "unchanged" || scenario == "completed" {
				require.Empty(t, pending.ModelIDs)
			} else {
				require.Equal(t, []string{"owned"}, pending.ModelIDs)
			}
		})
	}
}

func TestTrainingModelStatusMissingModel(t *testing.T) {
	ds := trainingFixture(t)
	_, err := ds.updateModelStatus("missing", Status_TRAINING)
	require.ErrorIs(t, err, ErrGetModelStorage)
}

func TestTrainingUploadProviderFailurePreservesModelState(t *testing.T) {
	ds := trainingFixture(t)
	startTrainingErrorProvider(t, ds, &trainingErrorProvider{uploadErr: status.Error(codes.Unavailable, "provider failed validation")})
	stream := &coverageUploadStream{requests: []*UploadAndValidateRequest{{
		Authorization: trainingAuth("upload_and_validate"),
		UploadInput:   &UploadInput{ModelId: "owned", Data: []byte("dataset")},
	}}}
	err := ds.UploadAndValidate(stream)
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Nil(t, stream.response)
	model, err := ds.storage.GetModel("owned")
	require.NoError(t, err)
	require.Equal(t, Status_CREATED, model.Status)
	_, pending, err := ds.pendingStorage.Get(ds.pendingStorage.buildPendingModelKey())
	require.NoError(t, err)
	require.False(t, pending)
}

func TestTrainingPriceWriteFailurePreservesStoredPrice(t *testing.T) {
	ds := trainingFixture(t)
	failure := errors.New("price storage unavailable")
	ds.storage.delegate = &serviceFailureStorage{TypedAtomicStorage: ds.storage.delegate, putErr: failure}
	err := ds.updateModelPrices("owned", &PriceInBaseUnit{Price: 99}, nil)
	require.ErrorContains(t, err, failure.Error())
	model, err := ds.storage.GetModel("owned")
	require.NoError(t, err)
	require.Equal(t, uint64(10), model.ValidatePrice)
	require.Equal(t, uint64(20), model.TrainPrice)
}
