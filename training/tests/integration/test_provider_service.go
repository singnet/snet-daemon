package tests

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/singnet/snet-daemon/v5/training"
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
	status          training.Status
}

func startTestService(address string) *grpc.Server {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	var trainingServer TrainServer
	training.RegisterModelServer(grpcServer, &trainingServer)

	trainingServer.curModelId = 0

	go func() {
		zap.L().Info("Starting test service", zap.String("address", address))
		if err := grpcServer.Serve(lis); err != nil {
			zap.L().Fatal("Error in starting grpcServer", zap.Error(err))
		}
	}()

	return grpcServer
}

type TrainServer struct {
	training.UnimplementedModelServer
	curModelId int
	models     []model
}

func (s *TrainServer) CreateModel(ctx context.Context, newModel *training.NewModel) (*training.ModelID, error) {
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
		status:          training.Status_CREATED,
	}
	s.models = append(s.models, *createdModel)

	s.curModelId += 1

	return &training.ModelID{
		ModelId: fmt.Sprintf("%v", s.curModelId),
	}, nil
}

func (s *TrainServer) ValidateModelPrice(ctx context.Context, request *training.ValidateRequest) (*training.PriceInBaseUnit, error) {
	return &training.PriceInBaseUnit{
		Price: 1,
	}, nil
}

func (s *TrainServer) UploadAndValidate(server training.Model_UploadAndValidateServer) error {
	panic("implement me")
}

func (s *TrainServer) ValidateModel(ctx context.Context, request *training.ValidateRequest) (*training.StatusResponse, error) {
	return &training.StatusResponse{
		Status: training.Status_VALIDATING,
	}, nil
}

func (s *TrainServer) TrainModelPrice(ctx context.Context, id *training.ModelID) (*training.PriceInBaseUnit, error) {
	return &training.PriceInBaseUnit{
		Price: 1,
	}, nil
}

func (s *TrainServer) TrainModel(ctx context.Context, id *training.ModelID) (*training.StatusResponse, error) {
	return &training.StatusResponse{
		Status: training.Status_TRAINING,
	}, nil
}

func (s *TrainServer) DeleteModel(ctx context.Context, id *training.ModelID) (*training.StatusResponse, error) {
	return &training.StatusResponse{
		Status: training.Status_DELETED,
	}, nil
}

func (s *TrainServer) GetModelStatus(ctx context.Context, id *training.ModelID) (*training.StatusResponse, error) {
	return &training.StatusResponse{
		Status: training.Status_VALIDATED,
	}, nil
}

func (s *TrainServer) mustEmbedUnimplementedModelServer() {
	panic("implement me")
}
