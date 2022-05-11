//go:generate protoc -I . ./training.proto --go_out=plugins=grpc:.
package training

import (
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/escrow"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"time"
)

type ModelService struct {
	serviceMetaData      *blockchain.ServiceMetadata
	organizationMetaData *blockchain.OrganizationMetaData
	channelService       escrow.PaymentChannelService
	storage              *ModelStorage
}

func getServiceClient() (client ModelClient, err error) {
	serviceURL := config.GetString(config.ModelTrainingEndpoint)
	conn, err := grpc.Dial(serviceURL, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("unable to connect to grpc endpoint: %v", err)
		return nil, err
	}
	defer conn.Close()
	// create the client instance
	client = NewModelClient(conn)
	return
}

func (m ModelService) CreateModel(c context.Context, request *CreateModelRequest) (*ModelDetailsResponse, error) {
	// verify the request that has come in
	// make a call to the client
	// if the response is successful , store details in etcd
	// send back the response to the client
	//TODO implement me
	fmt.Println("Adding addresses to etcd ")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*200)
	defer cancel()
	if client, err := getServiceClient(); err == nil {
		return client.CreateModel(ctx, request)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("Error in invoking service for Model Training")
	}

}

func (m ModelService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	fmt.Println("Updating model access addresses to etcd ")
	if client, err := getServiceClient(); err != nil {
		return client.UpdateModelAccess(ctx, request)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
}

func (m ModelService) DeleteModel(c context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	fmt.Println("Deleting model addresses from etcd ")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if client, err := getServiceClient(); err != nil {
		return client.DeleteModel(ctx, request)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
}

func (m ModelService) GetModelDetails(c context.Context, id *ModelId) (*ModelDetailsResponse, error) {
	fmt.Println("Getting model Details from etcd ......")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if client, err := getServiceClient(); err != nil {
		return client.GetModelDetails(ctx, id)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service method GetModelDetails for Model Training")
	}
}

func (m ModelService) GetTrainingStatus(c context.Context, id *ModelId) (*ModelDetailsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	fmt.Println("Getting Training Status details from etcd .....")
	defer cancel()
	if client, err := getServiceClient(); err != nil {
		return client.GetModelDetails(ctx, id)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service method GetTrainingStatus for Model Training")
	}
}

func NewModelService(channelService escrow.PaymentChannelService, serMetaData *blockchain.ServiceMetadata,
	orgMetadata *blockchain.OrganizationMetaData, storage *ModelStorage) *ModelService {
	return &ModelService{
		channelService:       channelService,
		serviceMetaData:      serMetaData,
		organizationMetaData: orgMetadata,
		storage:              storage,
	}
}

type IModelService interface {
}
