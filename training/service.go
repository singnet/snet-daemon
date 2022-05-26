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
	userStorage          *ModelUserStorage
	serviceUrl           string
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
func deferConnection(conn *grpc.ClientConn) {
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.WithError(err).Errorf("error in closing Client Connection")
		}
	}(conn)
}
func (service ModelService) getServiceClient() (conn *grpc.ClientConn, client ModelClient, err error) {
	conn, err = grpc.Dial(service.serviceUrl, grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Warningf("unable to connect to grpc endpoint: %v", err)
		return nil, nil, err
	}
	// create the client instance
	client = NewModelClient(conn)
	return
}
func (service ModelService) createModelDetails(request *CreateModelRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
	key := service.getModelKeyToCreate(request, response)
	data = service.getModelDataToCreate(request, response)
	//store the model details in etcd
	err = service.storage.Put(key, data)
	//for every accessible address in the list , store the user address and all the model Ids associated with it
	for _, address := range data.AuthorizedAddresses {
		userKey := getModelUserKey(key, address)
		userData := service.getModelUserData(key, address)
		err = service.userStorage.Put(userKey, userData)
	}
	return
}
func getModelUserKey(key *ModelKey, address string) *ModelUserKey {
	return &ModelUserKey{
		OrganizationId:  key.OrganizationId,
		ServiceId:       key.ServiceId,
		GroupId:         key.GroupId,
		GRPCMethodName:  key.GRPCMethodName,
		GRPCServiceName: key.GRPCServiceName,
		UserAddress:     address,
	}
}

func (service ModelService) getModelUserData(key *ModelKey, address string) *ModelUserData {
	//Check if there are any model Ids already associated with this user
	modelIds := make([]string, 0)
	userKey := getModelUserKey(key, address)
	if data, ok, err := service.userStorage.Get(userKey); ok && err != nil && data != nil {
		modelIds = data.ModelIds
	}
	modelIds = append(modelIds, key.ModelId)
	return &ModelUserData{
		OrganizationId: key.OrganizationId,
		ServiceId:      key.ServiceId,
		GroupId:        key.GroupId,
		GRPCMethodName: key.GRPCMethodName,
		UserAddress:    address,
		ModelIds:       modelIds,
	}
}

func (service ModelService) deleteUserModelDetails(key *ModelKey, data *ModelData) (err error) {

	for _, address := range data.AuthorizedAddresses {
		userKey := getModelUserKey(key, address)
		if data, ok, err := service.userStorage.Get(userKey); ok && err != nil && data != nil {
			data.ModelIds = remove(data.ModelIds, key.ModelId)
			err = service.userStorage.Put(userKey, data)
		}
	}
	return
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func (service ModelService) deleteModelDetails(request *UpdateModelRequest) (err error) {
	key := service.getModelKeyToUpdate(request.ModelDetailsRequest)
	data, ok, err := service.storage.Get(key)
	if ok && err != nil {
		data.Status = "DELETED"
		err = service.storage.Put(key, data)
		err = service.deleteUserModelDetails(key, data)
	}
	return
}
func convertModelDataToBO(data *ModelData) (responseData *ModelDetails) {
	responseData = &ModelDetails{
		ModelId:        data.ModelId,
		GrpcMethodName: data.GRPCMethodName,
		Description:    data.Description,
	}
	return
}

func (service ModelService) updateModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (err error) {
	key := service.getModelKeyToUpdate(request.ModelDetailsRequest)
	oldAddresses := make([]string, 0)

	if data, err := service.getModelDataForUpdate(request, response); err != nil {
		copy(oldAddresses, data.AuthorizedAddresses)
		if data, ok, err := service.storage.Get(key); err != nil && ok {
			data.AuthorizedAddresses = request.AddressList
			data.IsPublic = request.IsPubliclyAccessible
			data.UpdatedByAddress = request.ModelDetailsRequest.Authorization.SignerAddress
			data.Status = string(response.Status)
		}
		//get the difference of all the addresses b/w old and new
		updatedAddresses := difference(oldAddresses, request.AddressList)
		for _, address := range updatedAddresses {
			modelUserKey := getModelUserKey(key, address)
			modelUserData := service.getModelUserData(key, address)
			//if the address is present in the request but not in the old address , add it to the storage
			if isValuePresent(address, request.AddressList) {
				modelUserData.ModelIds = append(modelUserData.ModelIds, request.ModelDetailsRequest.ModelDetails.ModelId)
			} else { // the address was present in the old data , but not in new , hence needs to be deleted
				modelUserData.ModelIds = remove(modelUserData.ModelIds, request.ModelDetailsRequest.ModelDetails.ModelId)
			}
			err = service.userStorage.Put(modelUserKey, modelUserData)
			log.WithError(err)

		}
		err = service.storage.Put(key, data)
	}
	return
}

func difference(oldAddresses []string, newAddresses []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range oldAddresses {
			found := false
			for _, s2 := range newAddresses {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			oldAddresses, newAddresses = newAddresses, oldAddresses
		}
	}
	return diff
}

func isValuePresent(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func (service ModelService) updateModelDetailsForStatus(request *ModelDetailsRequest, response *ModelDetailsResponse) (err error) {
	key := service.getModelKeyToUpdate(request)
	if data, err := service.getModelDataForStatusUpdate(request, response); err != nil {
		err = service.storage.Put(key, data)
	}
	return
}
func (service ModelService) getModelKeyToCreate(request *CreateModelRequest, response *ModelDetailsResponse) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  request.ModelDetails.GrpcMethodName,
		GRPCServiceName: request.ModelDetails.GrpcServiceName,
		ModelId:         response.ModelDetails.ModelId,
	}
	return
}

func (service ModelService) getModelKeyToUpdate(request *ModelDetailsRequest) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName: request.ModelDetails.GrpcMethodName,
		ModelId:        request.ModelDetails.ModelId,
	}
	return
}

func (service ModelService) getModelDataForUpdate(request *UpdateModelRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
	data, err = service.getModelDataForStatusUpdate(request.ModelDetailsRequest, response)
	return
}

func (service ModelService) getModelDataForStatusUpdate(request *ModelDetailsRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
	key := service.getModelKeyToUpdate(request)
	ok := false

	if data, ok, err = service.storage.Get(key); err != nil && !ok {
		log.WithError(fmt.Errorf("issue with retrieving model data from storage"))
	}
	return
}
func (service ModelService) GetAllModels(c context.Context, request *AccessibleModelsRequest) (response *AccessibleModelsResponse, err error) {
	if request == nil || request.Authorization == nil {
		return &AccessibleModelsResponse{Status: Status_ERROR},
			fmt.Errorf(" Invalid request , no Authorization provided ")
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &AccessibleModelsResponse{Status: Status_ERROR},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	key := &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName: request.MethodName,
		UserAddress:    request.Authorization.SignerAddress,
	}
	modelDetailsArray := make([]*ModelDetails, 0)
	if data, ok, err := service.userStorage.Get(key); data != nil && ok && err != nil {
		for _, modelId := range data.ModelIds {
			modelKey := &ModelKey{
				OrganizationId: config.GetString(config.OrganizationId),
				ServiceId:      config.GetString(config.ServiceId),
				GroupId:        service.organizationMetaData.GetGroupIdString(),
				GRPCMethodName: request.MethodName,
				ModelId:        modelId,
			}
			if modelData, modelOk, modelErr := service.storage.Get(modelKey); modelOk && modelData != nil && modelErr != nil {
				modelDetailsArray = append(modelDetailsArray, convertModelDataToBO(modelData))
			}
		}
	}
	response = &AccessibleModelsResponse{
		ListOfModels: modelDetailsArray,
	}
	return
}

func (service ModelService) getModelDataToCreate(request *CreateModelRequest, response *ModelDetailsResponse) (data *ModelData) {

	data = &ModelData{
		Status:              string(response.Status),
		GRPCServiceName:     request.ModelDetails.GrpcServiceName,
		GRPCMethodName:      request.ModelDetails.GrpcMethodName,
		CreatedByAddress:    request.Authorization.SignerAddress,
		UpdatedByAddress:    request.Authorization.SignerAddress,
		AuthorizedAddresses: request.ModelDetails.AddressList,
		IsPublic:            request.ModelDetails.IsPubliclyAccessible,
		IsDefault:           request.ModelDetails.IsDefaultModel,
		ModelId:             response.ModelDetails.ModelId,
		OrganizationId:      config.GetString(config.OrganizationId),
		ServiceId:           config.GetString(config.ServiceId),
		GroupId:             service.organizationMetaData.GetGroupIdString(),
	}
	//by default add the creator to the Authorized list of Address
	if data.AuthorizedAddresses == nil {
		data.AuthorizedAddresses = make([]string, 0)
	}
	data.AuthorizedAddresses = append(data.AuthorizedAddresses, data.CreatedByAddress)
	return
}

func (service ModelService) CreateModel(c context.Context, request *CreateModelRequest) (response *ModelDetailsResponse,
	err error) {
	// verify the request
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	// make a call to the client
	// if the response is successful , store details in etcd
	// send back the response to the client

	if conn, client, err := service.getServiceClient(); err == nil {
		response, err = client.CreateModel(c, request)
		if err == nil {
			//store the details in etcd
			log.Infof("Creating model based on response from CreateModel of training service")
			if data, err := service.createModelDetails(request, response); err == nil {
				response = BuildCreateModelResponse(data)
			} else {
				return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
			}
		}
		deferConnection(conn)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf("error in invoking service for Model Training %v", err)
	}

	return
}
func BuildCreateModelResponse(data *ModelData) *ModelDetailsResponse {
	return &ModelDetailsResponse{
		Status: 0,
		ModelDetails: &ModelDetails{
			ModelId:              data.ModelId,
			GrpcMethodName:       data.GRPCMethodName,
			GrpcServiceName:      data.GRPCServiceName,
			Description:          data.Description,
			IsPubliclyAccessible: data.IsPublic,
			AddressList:          data.AuthorizedAddresses,
			TrainingDataLink:     data.TrainingLink,
			IsDefaultModel:       data.IsDefault,
			OrganizationId:       data.OrganizationId,
			ServiceId:            data.ServiceId,
		},
	}
}
func (service ModelService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.ModelDetailsRequest == nil || request.ModelDetailsRequest.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.ModelDetailsRequest.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if conn, client, err := service.getServiceClient(); err == nil {
		response, err = client.UpdateModelAccess(ctx, request)
		log.Infof("Updating model based on response from UpdateModel")
		if err = service.updateModelDetails(request, response); err != nil {
			return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
		}
		deferConnection(conn)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

func (service ModelService) DeleteModel(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.ModelDetailsRequest == nil || request.ModelDetailsRequest.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.ModelDetailsRequest.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*200)
	defer cancel()
	if conn, client, err := service.getServiceClient(); err == nil {
		response, err = client.DeleteModel(ctx, request)
		log.Infof("Deleting model based on response from DeleteModel")
		if err = service.deleteModelDetails(request); err != nil {
			return response, fmt.Errorf("issue with deleting Model Id in Storage %v", err)
		}
		deferConnection(conn)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}

	return
}

func (service ModelService) GetModelStatus(c context.Context, request *ModelDetailsRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERROR},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*200)
	defer cancel()

	if conn, client, err := service.getServiceClient(); err == nil {
		response, err = client.GetModelStatus(ctx, request)
		log.Infof("Updating model status based on response from UpdateModel")
		if err = service.updateModelDetailsForStatus(request, response); err != nil {
			return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
		}
		deferConnection(conn)
	} else {
		return &ModelDetailsResponse{Status: Status_ERROR}, fmt.Errorf("error in invoking service for Model Training")
	}
	return
}

//message used to sign is of the form ("__create_model", mpe_address, current_block_number)
func (service *ModelService) verifySignature(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes(request.Message, request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

//"user passed message	", user_address, current_block_number
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
	orgMetadata *blockchain.OrganizationMetaData, storage *ModelStorage, userStorage *ModelUserStorage) ModelServer {
	serviceURL := config.GetString(config.ModelTrainingEndpoint)
	if config.IsValidUrl(serviceURL) && config.GetBool(config.BlockchainEnabledKey) {
		return &ModelService{
			channelService:       channelService,
			serviceMetaData:      serMetaData,
			organizationMetaData: orgMetadata,
			storage:              storage,
			userStorage:          userStorage,
			serviceUrl:           serviceURL,
		}
	} else {
		return &NoModelSupportService{}
	}
}
