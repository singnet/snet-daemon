//go:generate protoc -I . ./training.proto --go_out=plugins=grpc:.
package training

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/escrow"
	"github.com/singnet/snet-daemon/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"math/big"
	"time"
)

type IService interface {
}
type ModelService struct {
	serviceMetaData      *blockchain.ServiceMetadata
	organizationMetaData *blockchain.OrganizationMetaData
	channelService       escrow.PaymentChannelService
	storage              *ModelStorage
	serviceUrl           string
}
type NoModelSupportService struct {
}

func (n NoModelSupportService) CreateModel(c context.Context, request *CreateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) DeleteModel(c context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) GetModelDetails(c context.Context, id *ModelDetailsRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) GetTrainingStatus(c context.Context, id *ModelDetailsRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid for training , please contact the AI developer")
}

func (m ModelService) getServiceClient() (client ModelClient, err error) {
	conn, err := grpc.Dial(m.serviceUrl, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("unable to connect to grpc endpoint: %v", err)
		return nil, err
	}
	defer conn.Close()
	// create the client instance
	client = NewModelClient(conn)
	return
}
func (m ModelService) storeModelDetails(request *CreateModelRequest, response *ModelDetailsResponse) (err error) {
	key := m.getModelKeyToCreate(request, response)
	data := m.createModelData(request, response)
	err = m.storage.Put(key, data)
	return
}

func (m ModelService) deleteModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (err error) {
	key := m.getModelKeyToUpdate(request)
	data, ok, err := m.storage.Get(key)
	if ok && err != nil {
		data.Status = "DELETED"
		err = m.storage.Put(key, data)
	}
	return
}

func (m ModelService) getModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (data *ModelUserData, err error) {
	key := m.getModelKeyToUpdate(request)
	data, ok, err := m.storage.Get(key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("error in retreving data from storage for key %v", key)
	}
	return
}

func (m ModelService) updateModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (err error) {
	key := m.getModelKeyToUpdate(request)
	if data, err := m.getModelDataForUpdate(request); err != nil {
		err = m.storage.Put(key, data)
	}
	return
}
func (m ModelService) getModelKeyToCreate(request *CreateModelRequest, response *ModelDetailsResponse) (key *ModelUserKey) {
	key = &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        m.organizationMetaData.GetGroupIdString(),
		MethodName:     request.MethodName,
		ModelId:        response.ModelDetails.ModelId,
	}
	return
}

func (m ModelService) getModelKeyToUpdate(request *UpdateModelRequest) (key *ModelUserKey) {
	key = &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        m.organizationMetaData.GetGroupIdString(),
		MethodName:     request.ModelDetails.MethodName,
		ModelId:        request.ModelDetails.ModelId,
	}
	return
}

func (m ModelService) getModelDataForUpdate(request *UpdateModelRequest) (data *ModelUserData, err error) {
	key := m.getModelKeyToUpdate(request)
	if data, ok, err := m.storage.Get(key); err != nil && ok {
		data.AuthorizedAddresses = request.AddressList
		data.isPublic = request.IsPubliclyAccessible
		data.UpdatedByAddress = request.Authorization.UserAddress
	}
	return
}

func (m ModelService) createModelData(request *CreateModelRequest, response *ModelDetailsResponse) (data *ModelUserData) {
	data = &ModelUserData{
		Status:              string(response.Status),
		CreatedByAddress:    request.Authorization.UserAddress,
		AuthorizedAddresses: request.AddressList,
		isPublic:            request.IsPubliclyAccessible,
		ModelId:             response.ModelDetails.ModelId,
	}
	return
}

func (m ModelService) CreateModel(c context.Context, request *CreateModelRequest) (response *ModelDetailsResponse,
	err error) {
	// verify the request
	if err = m.verifySignerForCreateModel(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	// make a call to the client
	// if the response is successful , store details in etcd
	// send back the response to the client
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*200)
	defer cancel()
	if client, err := m.getServiceClient(); err == nil {
		response, err = client.CreateModel(ctx, request)
		if err == nil {
			//store the details in etcd
			log.Infof("Creating model based on response from CreateModel")
			if err = m.storeModelDetails(request, response); err != nil {
				return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
			}
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf("error in invoking service for Model Training %v", err)
	}
	return
}

func (m ModelService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if err = m.verifySignerForUpdateModel(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if client, err := m.getServiceClient(); err != nil {
		response, err = client.UpdateModelAccess(ctx, request)
		log.Infof("Updating model based on response from UpdateModel")
		if err = m.updateModelDetails(request, response); err != nil {
			return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

func (m ModelService) DeleteModel(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if err = m.verifySignerForDeleteModel(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if client, err := m.getServiceClient(); err != nil {
		response, err = client.DeleteModel(ctx, request)
		log.Infof("Deleting model based on response from DeleteModel")
		if err = m.deleteModelDetails(request, response); err != nil {
			return response, fmt.Errorf("issue with deleting Model Id in Storage %v", err)
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}

	return
}

func (m ModelService) GetModelDetails(c context.Context, request *ModelDetailsRequest) (response *ModelDetailsResponse,
	err error) {
	if err = m.verifySignerForGetModelDetails(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if client, err := m.getServiceClient(); err != nil {
		response, err = client.GetModelDetails(ctx, request)
		log.Infof("Updating model based on response from GetModelDetails")
		//todo update data from client and return data stored in etcd

	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

func (m ModelService) GetTrainingStatus(c context.Context, request *ModelDetailsRequest) (response *ModelDetailsResponse,
	err error) {
	if err = m.verifySignerForGetTrainingStatus(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if client, err := m.getServiceClient(); err != nil {
		response, err = client.GetTrainingStatus(ctx, request)
		log.Infof("Updating model based on response from GetTrainingStatus")
		//todo update data from client and return data stored in etcd

	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

//message used to sign is of the form ("__create_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForCreateModel(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__create_model", request),
		request.GetSignature(), utils.ToChecksumAddress(request.UserAddress))
}

//message used to sign is of the form ("__update_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForUpdateModel(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__update_model", request),
		request.GetSignature(), utils.ToChecksumAddress(request.UserAddress))
}

//message used to sign is of the form ("__delete_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForDeleteModel(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__delete_model", request),
		request.GetSignature(), utils.ToChecksumAddress(request.UserAddress))
}

//message used to sign is of the form ("__delete_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForGetModelDetails(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__get_model_details", request),
		request.GetSignature(), utils.ToChecksumAddress(request.UserAddress))
}

//message used to sign is of the form ("__get_training_status", mpe_address, current_block_number)
func (service *ModelService) verifySignerForGetTrainingStatus(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__get_training_status", request),
		request.GetSignature(), utils.ToChecksumAddress(request.UserAddress))
}

//"__methodName", user_address, current_block_number
func (service *ModelService) getMessageBytes(prefixMessage string, request *AuthorizationDetails) []byte {
	userAddress := utils.ToChecksumAddress(request.UserAddress)
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		userAddress.Bytes(),
		abi.U256(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return message
}

func NewModelService(channelService escrow.PaymentChannelService, serMetaData *blockchain.ServiceMetadata,
	orgMetadata *blockchain.OrganizationMetaData, storage *ModelStorage) ModelServer {
	serviceURL := config.GetString(config.ModelTrainingEndpoint)
	if config.IsValidUrl(serviceURL) {
		return &ModelService{
			channelService:       channelService,
			serviceMetaData:      serMetaData,
			organizationMetaData: orgMetadata,
			storage:              storage,
			serviceUrl:           serviceURL,
		}
	} else {
		return &NoModelSupportService{}
	}
}

type IModelService interface {
}
