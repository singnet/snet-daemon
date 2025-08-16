package training

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"
)

type NoModelSupportTrainingService struct {
}

type NoTrainingDaemonServer struct {
}

func (n NoTrainingDaemonServer) mustEmbedUnimplementedDaemonServer() {
	panic("implement me")
}

func (n NoTrainingDaemonServer) CreateModel(ctx context.Context, request *NewModelRequest) (*ModelResponse, error) {
	return &ModelResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) ValidateModelPrice(ctx context.Context, request *AuthValidateRequest) (*PriceInBaseUnit, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) UploadAndValidate(server Daemon_UploadAndValidateServer) error {
	return fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) ValidateModel(ctx context.Context, request *AuthValidateRequest) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) TrainModelPrice(ctx context.Context, request *CommonRequest) (*PriceInBaseUnit, error) {
	return nil,
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) TrainModel(ctx context.Context, request *CommonRequest) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) DeleteModel(ctx context.Context, request *CommonRequest) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) GetTrainingMetadata(ctx context.Context, empty *emptypb.Empty) (*TrainingMetadata, error) {
	return &TrainingMetadata{TrainingEnabled: false, TrainingInProto: false}, fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) GetAllModels(ctx context.Context, request *AllModelsRequest) (*ModelsResponse, error) {
	return &ModelsResponse{ListOfModels: []*ModelResponse{}},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) GetModel(ctx context.Context, request *CommonRequest) (*ModelResponse, error) {
	return &ModelResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) UpdateModel(ctx context.Context, request *UpdateModelRequest) (*ModelResponse, error) {
	return &ModelResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoTrainingDaemonServer) GetMethodMetadata(ctx context.Context, request *MethodMetadataRequest) (*MethodMetadata, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportTrainingService) CreateModel(ctx context.Context, model *NewModel) (*ModelID, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) ValidateModelPrice(ctx context.Context, request *ValidateRequest) (*PriceInBaseUnit, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) UploadAndValidate(server Model_UploadAndValidateServer) error {
	return fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) ValidateModel(ctx context.Context, request *ValidateRequest) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) TrainModelPrice(ctx context.Context, id *ModelID) (*PriceInBaseUnit, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) TrainModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) DeleteModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) GetModelStatus(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return &StatusResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportTrainingService) mustEmbedUnimplementedModelServer() {
	panic("implement me")
}

func (ds *DaemonService) mustEmbedUnimplementedDaemonServer() {
	panic("implement me")
}
