package training

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/escrow"
	"golang.org/x/net/context"
	"reflect"
	"testing"
)

func TestModelService_CreateModel(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		c       context.Context
		request *CreateModelRequest
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantResponse *ModelDetailsResponse
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotResponse, err := service.CreateModel(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResponse, tt.wantResponse) {
				t.Errorf("CreateModel() gotResponse = %v, want %v", gotResponse, tt.wantResponse)
			}
		})
	}
}

func TestModelService_DeleteModel(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		c       context.Context
		request *UpdateModelRequest
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantResponse *ModelDetailsResponse
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotResponse, err := service.DeleteModel(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResponse, tt.wantResponse) {
				t.Errorf("DeleteModel() gotResponse = %v, want %v", gotResponse, tt.wantResponse)
			}
		})
	}
}

func TestModelService_GetAllModels(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		c       context.Context
		request *AccessibleModelsRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *AccessibleModelsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			got, err := service.GetAllModels(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllModels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllModels() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelService_GetModelStatus(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		c       context.Context
		request *ModelDetailsRequest
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantResponse *ModelDetailsResponse
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotResponse, err := service.GetModelStatus(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModelStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResponse, tt.wantResponse) {
				t.Errorf("GetModelStatus() gotResponse = %v, want %v", gotResponse, tt.wantResponse)
			}
		})
	}
}

func TestModelService_UpdateModelAccess(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		c       context.Context
		request *UpdateModelRequest
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantResponse *ModelDetailsResponse
		wantErr      bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotResponse, err := service.UpdateModelAccess(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateModelAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResponse, tt.wantResponse) {
				t.Errorf("UpdateModelAccess() gotResponse = %v, want %v", gotResponse, tt.wantResponse)
			}
		})
	}
}

func TestModelService_createModelData(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *CreateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantData *ModelUserData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if gotData := service.createModelData(tt.args.request, tt.args.response); !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("createModelData() = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestModelService_deleteModelDetails(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *UpdateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.deleteModelDetails(tt.args.request, tt.args.response); (err != nil) != tt.wantErr {
				t.Errorf("deleteModelDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_getMessageBytes(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		prefixMessage string
		request       *AuthorizationDetails
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if got := service.getMessageBytes(tt.args.prefixMessage, tt.args.request); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMessageBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelService_getModelDataForStatusUpdate(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *ModelDetailsRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantData *ModelUserData
		wantErr  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotData, err := service.getModelDataForStatusUpdate(tt.args.request, tt.args.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("getModelDataForStatusUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("getModelDataForStatusUpdate() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestModelService_getModelDataForUpdate(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *UpdateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantData *ModelUserData
		wantErr  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotData, err := service.getModelDataForUpdate(tt.args.request, tt.args.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("getModelDataForUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("getModelDataForUpdate() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestModelService_getModelDetails(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *UpdateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantData *ModelUserData
		wantErr  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotData, err := service.getModelDetails(tt.args.request, tt.args.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("getModelDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("getModelDetails() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestModelService_getModelKeyToCreate(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *CreateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantKey *ModelUserKey
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if gotKey := service.getModelKeyToCreate(tt.args.request, tt.args.response); !reflect.DeepEqual(gotKey, tt.wantKey) {
				t.Errorf("getModelKeyToCreate() = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}

func TestModelService_getModelKeyToUpdate(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request *ModelDetailsRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantKey *ModelUserKey
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if gotKey := service.getModelKeyToUpdate(tt.args.request); !reflect.DeepEqual(gotKey, tt.wantKey) {
				t.Errorf("getModelKeyToUpdate() = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}

func TestModelService_getServiceClient(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	tests := []struct {
		name       string
		fields     fields
		wantClient ModelClient
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			gotClient, err := service.getServiceClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("getServiceClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotClient, tt.wantClient) {
				t.Errorf("getServiceClient() gotClient = %v, want %v", gotClient, tt.wantClient)
			}
		})
	}
}

func TestModelService_storeModelDetails(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *CreateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.storeModelDetails(tt.args.request, tt.args.response); (err != nil) != tt.wantErr {
				t.Errorf("storeModelDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_updateModelDetails(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *UpdateModelRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.updateModelDetails(tt.args.request, tt.args.response); (err != nil) != tt.wantErr {
				t.Errorf("updateModelDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_updateModelDetailsForStatus(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request  *ModelDetailsRequest
		response *ModelDetailsResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.updateModelDetailsForStatus(tt.args.request, tt.args.response); (err != nil) != tt.wantErr {
				t.Errorf("updateModelDetailsForStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_verifySignatureForGetAllModels(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request *AuthorizationDetails
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.verifySignatureForGetAllModels(tt.args.request); (err != nil) != tt.wantErr {
				t.Errorf("verifySignatureForGetAllModels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_verifySignerForCreateModel(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request *AuthorizationDetails
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.verifySignerForCreateModel(tt.args.request); (err != nil) != tt.wantErr {
				t.Errorf("verifySignerForCreateModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_verifySignerForDeleteModel(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request *AuthorizationDetails
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.verifySignerForDeleteModel(tt.args.request); (err != nil) != tt.wantErr {
				t.Errorf("verifySignerForDeleteModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_verifySignerForGetModelStatus(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request *AuthorizationDetails
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.verifySignerForGetModelStatus(tt.args.request); (err != nil) != tt.wantErr {
				t.Errorf("verifySignerForGetModelStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestModelService_verifySignerForUpdateModel(t *testing.T) {
	type fields struct {
		serviceMetaData      *blockchain.ServiceMetadata
		organizationMetaData *blockchain.OrganizationMetaData
		channelService       escrow.PaymentChannelService
		storage              *ModelStorage
		serviceUrl           string
	}
	type args struct {
		request *AuthorizationDetails
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ModelService{
				serviceMetaData:      tt.fields.serviceMetaData,
				organizationMetaData: tt.fields.organizationMetaData,
				channelService:       tt.fields.channelService,
				storage:              tt.fields.storage,
				serviceUrl:           tt.fields.serviceUrl,
			}
			if err := service.verifySignerForUpdateModel(tt.args.request); (err != nil) != tt.wantErr {
				t.Errorf("verifySignerForUpdateModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewModelService(t *testing.T) {
	type args struct {
		channelService escrow.PaymentChannelService
		serMetaData    *blockchain.ServiceMetadata
		orgMetadata    *blockchain.OrganizationMetaData
		storage        *ModelStorage
	}
	tests := []struct {
		name string
		args args
		want ModelServer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewModelService(tt.args.channelService, tt.args.serMetaData, tt.args.orgMetadata, tt.args.storage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewModelService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoModelSupportService_CreateModel(t *testing.T) {
	type args struct {
		c       context.Context
		request *CreateModelRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *ModelDetailsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NoModelSupportService{}
			got, err := n.CreateModel(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateModel() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoModelSupportService_DeleteModel(t *testing.T) {
	type args struct {
		c       context.Context
		request *UpdateModelRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *ModelDetailsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NoModelSupportService{}
			got, err := n.DeleteModel(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeleteModel() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoModelSupportService_GetAllModels(t *testing.T) {
	type args struct {
		c       context.Context
		request *AccessibleModelsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *AccessibleModelsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NoModelSupportService{}
			got, err := n.GetAllModels(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllModels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllModels() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoModelSupportService_GetModelDetails(t *testing.T) {
	type args struct {
		c  context.Context
		id *ModelDetailsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *ModelDetailsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NoModelSupportService{}
			got, err := n.GetModelDetails(tt.args.c, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModelDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetModelDetails() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoModelSupportService_GetModelStatus(t *testing.T) {
	type args struct {
		c  context.Context
		id *ModelDetailsRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *ModelDetailsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NoModelSupportService{}
			got, err := n.GetModelStatus(tt.args.c, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModelStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetModelStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoModelSupportService_UpdateModelAccess(t *testing.T) {
	type args struct {
		c       context.Context
		request *UpdateModelRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *ModelDetailsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NoModelSupportService{}
			got, err := n.UpdateModelAccess(tt.args.c, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateModelAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateModelAccess() got = %v, want %v", got, tt.want)
			}
		})
	}
}
