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

func (service ModelService) GetAllModels(c context.Context, request *AccessibleModelsRequest) (response *AccessibleModelsResponse, err error) {
	if request == nil || request.Authorization == nil {
		return &AccessibleModelsResponse{Status: Status_ERROR},
			fmt.Errorf(" INVALID  REQUEST DETAILS , no Authorization provided ")
	}
	//TODO
	return
}

type NoModelSupportService struct {
}

func (n NoModelSupportService) GetAllModels(c context.Context, request *AccessibleModelsRequest) (*AccessibleModelsResponse, error) {
	return &AccessibleModelsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
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

func (n NoModelSupportService) GetModelStatus(c context.Context, id *ModelDetailsRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERROR},
		fmt.Errorf("service end point is not defined or is invalid for training , please contact the AI developer")
}

func (service ModelService) getServiceClient() (client ModelClient, err error) {
	conn, err := grpc.Dial(service.serviceUrl, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("unable to connect to grpc endpoint: %v", err)
		return nil, err
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.WithError(err).Errorf("error in closing Client Connection")
		}
	}(conn)
	// create the client instance
	client = NewModelClient(conn)
	return
}
func (service ModelService) storeModelDetails(request *CreateModelRequest, response *ModelDetailsResponse) (err error) {
	key := service.getModelKeyToCreate(request, response)
	data := service.createModelData(request, response)
	err = service.storage.Put(key, data)
	return
}

func (service ModelService) deleteModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (err error) {
	key := service.getModelKeyToUpdate(request.ModelDetailsRequest)
	data, ok, err := service.storage.Get(key)
	if ok && err != nil {
		data.Status = "DELETED"
		err = service.storage.Put(key, data)
	}
	return
}

func (service ModelService) getModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (data *ModelUserData, err error) {

	key := service.getModelKeyToUpdate(request.ModelDetailsRequest)
	data, ok, err := service.storage.Get(key)

	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("error in retreving data from storage for key %v", key)
	}
	data.Status = string(response.Status)
	err = service.storage.Put(key, data)
	return
}

func (service ModelService) updateModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (err error) {
	key := service.getModelKeyToUpdate(request.ModelDetailsRequest)
	if data, err := service.getModelDataForUpdate(request, response); err != nil {
		if data, ok, err := service.storage.Get(key); err != nil && ok {
			data.AuthorizedAddresses = request.AddressList
			data.isPublic = request.IsPubliclyAccessible
			data.UpdatedByAddress = request.ModelDetailsRequest.Authorization.SignerAddress
			data.Status = string(response.Status)
		}

		err = service.storage.Put(key, data)
	}
	return
}

func (service ModelService) updateModelDetailsForStatus(request *ModelDetailsRequest, response *ModelDetailsResponse) (err error) {
	key := service.getModelKeyToUpdate(request)
	if data, err := service.getModelDataForStatusUpdate(request, response); err != nil {
		err = service.storage.Put(key, data)
	}
	return
}
func (service ModelService) getModelKeyToCreate(request *CreateModelRequest, response *ModelDetailsResponse) (key *ModelUserKey) {
	key = &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        service.organizationMetaData.GetGroupIdString(),
		MethodName:     request.MethodName,
		ModelId:        response.ModelDetails.ModelId,
	}
	return
}

func (service ModelService) getModelKeyToUpdate(request *ModelDetailsRequest) (key *ModelUserKey) {
	key = &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupID:        service.organizationMetaData.GetGroupIdString(),
		MethodName:     request.ModelDetails.MethodName,
		ModelId:        request.ModelDetails.ModelId,
	}
	return
}

func (service ModelService) getModelDataForUpdate(request *UpdateModelRequest, response *ModelDetailsResponse) (data *ModelUserData, err error) {
	data, err = service.getModelDataForStatusUpdate(request.ModelDetailsRequest, response)
	return
}

func (service ModelService) getModelDataForStatusUpdate(request *ModelDetailsRequest, response *ModelDetailsResponse) (data *ModelUserData, err error) {
	key := service.getModelKeyToUpdate(request)
	ok := false

	if data, ok, err = service.storage.Get(key); err != nil && !ok {
		log.WithError(fmt.Errorf("Issue with retrieving model data from storage"))
	}
	return
}

func (service ModelService) createModelData(request *CreateModelRequest, response *ModelDetailsResponse) (data *ModelUserData) {
	data = &ModelUserData{
		Status:              string(response.Status),
		CreatedByAddress:    request.Authorization.SignerAddress,
		AuthorizedAddresses: request.AddressList,
		isPublic:            request.IsPubliclyAccessible,
		ModelId:             response.ModelDetails.ModelId,
	}
	return
}

func (service ModelService) CreateModel(c context.Context, request *CreateModelRequest) (response *ModelDetailsResponse,
	err error) {
	// verify the request
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" INVALID  REQUEST DETAILS , no Authorization provided  , %v", err)
	}
	if err = service.verifySignerForCreateModel(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	// make a call to the client
	// if the response is successful , store details in etcd
	// send back the response to the client
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*200)
	defer cancel()
	if client, err := service.getServiceClient(); err == nil {
		response, err = client.CreateModel(ctx, request)
		if err == nil {
			//store the details in etcd
			log.Infof("Creating model based on response from CreateModel")
			if err = service.storeModelDetails(request, response); err != nil {
				return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
			}
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf("error in invoking service for Model Training %v", err)
	}
	return
}

func (service ModelService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.ModelDetailsRequest == nil || request.ModelDetailsRequest.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" INVALID  REQUEST DETAILS , no Authorization provided  , %v", err)
	}
	if err = service.verifySignerForUpdateModel(request.ModelDetailsRequest.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if client, err := service.getServiceClient(); err != nil {
		response, err = client.UpdateModelAccess(ctx, request)
		log.Infof("Updating model based on response from UpdateModel")
		if err = service.updateModelDetails(request, response); err != nil {
			return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

func (service ModelService) DeleteModel(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.ModelDetailsRequest == nil || request.ModelDetailsRequest.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" INVALID  REQUEST DETAILS , no Authorization provided  , %v", err)
	}
	if err = service.verifySignerForDeleteModel(request.ModelDetailsRequest.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if client, err := service.getServiceClient(); err != nil {
		response, err = client.DeleteModel(ctx, request)
		log.Infof("Deleting model based on response from DeleteModel")
		if err = service.deleteModelDetails(request, response); err != nil {
			return response, fmt.Errorf("issue with deleting Model Id in Storage %v", err)
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}

	return
}

func (service ModelService) GetModelStatus(c context.Context, request *ModelDetailsRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" INVALID  REQUEST DETAILS , no Authorization provided  , %v", err)
	}
	if err = service.verifySignerForGetModelStatus(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" authentication FAILED , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if client, err := service.getServiceClient(); err != nil {
		response, err = client.GetModelStatus(ctx, request)
		log.Infof("Updating modelG based on response from UpdateModel")
		if err = service.updateModelDetailsForStatus(request, response); err != nil {
			return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
		}
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

//message used to sign is of the form ("__create_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForCreateModel(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__CreateModel", request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

//message used to sign is of the form ("__update_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForUpdateModel(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__UpdateModel", request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

//message used to sign is of the form ("__delete_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForDeleteModel(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__DeleteModel", request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

//message used to sign is of the form ("__delete_model", mpe_address, current_block_number)
func (service *ModelService) verifySignerForGetModelStatus(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__GetModelStatus", request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

//message used to sign is of the form ("__get_training_status", mpe_address, current_block_number)
func (service *ModelService) verifySignatureForGetAllModels(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes("__GetAllModels", request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

//"__methodName", user_address, current_block_number
func (service *ModelService) getMessageBytes(prefixMessage string, request *AuthorizationDetails) []byte {
	userAddress := utils.ToChecksumAddress(request.SignerAddress)
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
