package integrationtests

import (
	"context"
	"log"
	"net"

	"github.com/singnet/snet-daemon/v5/training"
	"google.golang.org/grpc"
)

type model struct {
	name   string
	method string
	desc   string
}

func startTestService(address string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	var trainingServer TrainServer
	training.RegisterModelServer(grpcServer, &trainingServer)

	go func() {
		log.Println("Starting server on 127.0.0.1:5001")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpcServer failed: %v", err)
		}
	}()
}

type TrainServer struct {
	training.UnimplementedModelServer
}

func (s *TrainServer) CreateModel(ctx context.Context, newModel *training.NewModel) (*training.ModelID, error) {
	return &training.ModelID{
		ModelId: "1", // TODO random gen
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
