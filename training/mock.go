package training

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NoModelSupportService struct {
}

type NoTrainingService struct {
}

func (n NoTrainingService) mustEmbedUnimplementedDaemonServer() {
	panic("implement me")
}

func (n NoTrainingService) CreateModel(ctx context.Context, request *NewModelRequest) (*ModelResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) ValidateModelPrice(ctx context.Context, request *AuthValidateRequest) (*PriceInBaseUnit, error) {
	panic("implement me")
}

func (n NoTrainingService) UploadAndValidate(server Daemon_UploadAndValidateServer) error {
	panic("implement me")
}

func (n NoTrainingService) ValidateModel(ctx context.Context, request *AuthValidateRequest) (*StatusResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) TrainModelPrice(ctx context.Context, request *CommonRequest) (*PriceInBaseUnit, error) {
	panic("implement me")
}

func (n NoTrainingService) TrainModel(ctx context.Context, request *CommonRequest) (*StatusResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) DeleteModel(ctx context.Context, request *CommonRequest) (*StatusResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) GetTrainingMetadata(ctx context.Context, empty *emptypb.Empty) (*TrainingMetadata, error) {
	panic("implement me")
}

func (n NoTrainingService) GetAllModels(ctx context.Context, request *AllModelsRequest) (*ModelsResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) GetModel(ctx context.Context, request *CommonRequest) (*ModelResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) UpdateModel(ctx context.Context, request *UpdateModelRequest) (*ModelResponse, error) {
	panic("implement me")
}

func (n NoTrainingService) GetMethodMetadata(ctx context.Context, request *MethodMetadataRequest) (*MethodMetadata, error) {
	panic("implement me")
}

func (service ModelService) CreateModel(ctx context.Context, model *NewModel) (*ModelID, error) {
	panic("implement me")
}

func (service ModelService) ValidateModelPrice(ctx context.Context, request *ValidateRequest) (*PriceInBaseUnit, error) {
	panic("implement me")
}

func (service ModelService) UploadAndValidate(server Model_UploadAndValidateServer) error {
	panic("implement me")
}

func (service ModelService) ValidateModel(ctx context.Context, request *ValidateRequest) (*StatusResponse, error) {
	panic("implement me")
}

func (service ModelService) TrainModelPrice(ctx context.Context, id *ModelID) (*PriceInBaseUnit, error) {
	panic("implement me")
}

func (service ModelService) TrainModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	panic("implement me")
}

func (service ModelService) DeleteModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	panic("implement me")
}

func (service ModelService) GetModelStatus(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	panic("implement me")
}

func (service ModelService) mustEmbedUnimplementedModelServer() {
	panic("implement me")
}

func (n NoModelSupportService) CreateModel(ctx context.Context, model *NewModel) (*ModelID, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) ValidateModelPrice(ctx context.Context, request *ValidateRequest) (*PriceInBaseUnit, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) UploadAndValidate(server Model_UploadAndValidateServer) error {
	return fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) ValidateModel(ctx context.Context, request *ValidateRequest) (*StatusResponse, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) TrainModelPrice(ctx context.Context, id *ModelID) (*PriceInBaseUnit, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) TrainModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) DeleteModel(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) GetModelStatus(ctx context.Context, id *ModelID) (*StatusResponse, error) {
	return nil, fmt.Errorf("service end point is not defined or is invalid, please contact the AI developer")
}

func (n NoModelSupportService) mustEmbedUnimplementedModelServer() {
	panic("implement me")
}

func (ds *DaemonService) mustEmbedUnimplementedDaemonServer() {
	panic("implement me")
}
