package training

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type model struct {
	modelId         string
	name            string
	desc            string
	grpcMethodName  string
	grpcServiceName string
	addressList     []string
	isPublic        bool
	serviceId       string
	groupId         string
	status          Status
}

func startTestService(address string) *grpc.Server {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	var trainingServer TestTrainServer
	RegisterModelServer(grpcServer, &trainingServer)

	trainingServer.curModelId = 0

	go func() {
		zap.L().Info("Starting test service", zap.String("address", address))
		if err := grpcServer.Serve(lis); err != nil {
			zap.L().Fatal("Error in starting grpcServer", zap.Error(err))
		}
	}()

	return grpcServer
}

type TestTrainServer struct {
	UnimplementedModelServer
	curModelId int
	models     []model
}

func (s *TestTrainServer) CreateModel(ctx context.Context, newModel *NewModel) (*ModelID, error) {
	modelIdStr := fmt.Sprintf("%v", s.curModelId)
	createdModel := &model{
		modelId:         modelIdStr,
		name:            newModel.Name,
		desc:            newModel.Description,
		grpcMethodName:  newModel.GrpcMethodName,
		grpcServiceName: newModel.GrpcServiceName,
		addressList:     newModel.AddressList,
		isPublic:        newModel.IsPublic,
		serviceId:       newModel.ServiceId,
		groupId:         newModel.GroupId,
		status:          Status_CREATED,
	}
	s.models = append(s.models, *createdModel)

	s.curModelId += 1

	return &ModelID{
		ModelId: fmt.Sprintf("%v", s.curModelId),
	}, nil
}

func (s *TestTrainServer) ValidateModelPrice(ctx context.Context, request *ValidateRequest) (*PriceInBaseUnit, error) {
	return &PriceInBaseUnit{
		Price: 1,
	}, nil
}

func (s *TestTrainServer) UploadAndValidate(server Model_UploadAndValidateServer) error {
	panic("implement me")
}

func (s *TestTrainServer) ValidateModel(ctx context.Context, request *ValidateRequest) (*StatusResponse, error) {
	return &StatusResponse{
		Status: Status_VALIDATING,
	}, nil
}

func (s *TestTrainServer) TrainModelPrice(ctx context.Context, id *ModelID) (*PriceInBaseUnit, error) {
	return &PriceInBaseUnit{
		Price: 1,
	}, nil
}

func (s *TestTrainServer) TrainModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return &StatusResponse{
		Status: Status_TRAINING,
	}, nil
}

func (s *TestTrainServer) DeleteModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return &StatusResponse{
		Status: Status_DELETED,
	}, nil
}

func (s *TestTrainServer) GetModelStatus(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return &StatusResponse{
		Status: Status_VALIDATED,
	}, nil
}

func (s *TestTrainServer) mustEmbedUnimplementedModelServer() {
	panic("implement me")
}
