//go:generate protoc -I . ./training.proto --go-grpc_out=. --go_out=.
package training

import (
	"bytes"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"math/big"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/escrow"
	"github.com/singnet/snet-daemon/v5/utils"

	"github.com/ethereum/go-ethereum/common/math"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	DateFormat = "02-01-2006"
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

func (service ModelService) mustEmbedUnimplementedModelServer() {
	//TODO implement me
	panic("implement me")
}

type NoModelSupportService struct {
}

func (n NoModelSupportService) mustEmbedUnimplementedModelServer() {
	//TODO implement me
	panic("implement me")
}

func (n NoModelSupportService) GetAllModels(c context.Context, request *AccessibleModelsRequest) (*AccessibleModelsResponse, error) {
	return &AccessibleModelsResponse{},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) CreateModel(c context.Context, request *CreateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) DeleteModel(c context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid , please contact the AI developer")
}

func (n NoModelSupportService) GetModelStatus(c context.Context, id *ModelDetailsRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_ERRORED},
		fmt.Errorf("service end point is not defined or is invalid for training , please contact the AI developer")
}
func deferConnection(conn *grpc.ClientConn) {
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			zap.L().Error("error in closing Client Connection", zap.Error(err))
		}
	}(conn)
}
func getConnection(endpoint string) (conn *grpc.ClientConn) {
	options := grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024),
		grpc.MaxCallSendMsgSize(config.GetInt(config.MaxMessageSizeInMB)*1024*1024))

	passthroughURL, err := url.Parse(endpoint)
	if err != nil {
		zap.L().Panic("error parsing passthrough endpoint", zap.Error(err))
	}
	if strings.Compare(passthroughURL.Scheme, "https") == 0 {
		conn, err = grpc.NewClient(passthroughURL.Host,
			grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")), options)
		if err != nil {
			zap.L().Panic("error dialing service", zap.Error(err))
		}
	} else {
		conn, err = grpc.NewClient(passthroughURL.Host, grpc.WithTransportCredentials(insecure.NewCredentials()), options)

		if err != nil {
			zap.L().Panic("error dialing service", zap.Error(err))
		}
	}
	return
}

func (service ModelService) getServiceClient() (conn *grpc.ClientConn, client ModelClient, err error) {
	conn = getConnection(service.serviceUrl)
	client = NewModelClient(conn)
	return
}

func (service ModelService) createModelDetails(request *CreateModelRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
	key := service.getModelKeyToCreate(request, response)
	data = service.getModelDataToCreate(request, response)
	//store the model details in etcd
	zap.L().Debug("createModelDetails", zap.Any("key", key))
	err = service.storage.Put(key, data)
	if err != nil {
		zap.L().Error("can't put model in etcd", zap.Error(err))
		return
	}
	//for every accessible address in the list , store the user address and all the model Ids associated with it
	for _, address := range data.AuthorizedAddresses {
		userKey := getModelUserKey(key, address)
		userData := service.getModelUserData(key, address)
		zap.L().Debug("createModelDetails", zap.Any("userKey", userKey))
		err = service.userStorage.Put(userKey, userData)
		if err != nil {
			zap.L().Error("can't put in user storage", zap.Error(err))
			return
		}
		zap.L().Debug("creating training model", zap.String("userKey", userKey.String()), zap.String("userData", userData.String()))
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
	zap.L().Debug("user model key is" + userKey.String())
	data, ok, err := service.userStorage.Get(userKey)
	if ok && err == nil && data != nil {
		modelIds = data.ModelIds
	}
	if err != nil {
		zap.L().Error("can't get model data from etcd", zap.Error(err))
	}
	modelIds = append(modelIds, key.ModelId)
	return &ModelUserData{
		OrganizationId:  key.OrganizationId,
		ServiceId:       key.ServiceId,
		GroupId:         key.GroupId,
		GRPCMethodName:  key.GRPCMethodName,
		GRPCServiceName: key.GRPCServiceName,
		UserAddress:     address,
		ModelIds:        modelIds,
	}
}

func (service ModelService) deleteUserModelDetails(key *ModelKey, data *ModelData) (err error) {

	for _, address := range data.AuthorizedAddresses {
		userKey := getModelUserKey(key, address)
		if data, ok, err := service.userStorage.Get(userKey); ok && err == nil && data != nil {
			data.ModelIds = remove(data.ModelIds, key.ModelId)
			err = service.userStorage.Put(userKey, data)
			if err != nil {
				zap.L().Error(err.Error())
			}
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

func (service ModelService) deleteModelDetails(request *UpdateModelRequest) (data *ModelData, err error) {
	key := service.getModelKeyToUpdate(request)
	ok := false
	data, ok, err = service.storage.Get(key)
	if ok && err == nil {
		data.Status = Status_DELETED
		data.UpdatedDate = fmt.Sprintf("%v", time.Now().Format(DateFormat))
		err = service.storage.Put(key, data)
		err = service.deleteUserModelDetails(key, data)
	}
	return
}
func convertModelDataToBO(data *ModelData) (responseData *ModelDetails) {
	responseData = &ModelDetails{
		ModelId:              data.ModelId,
		GrpcMethodName:       data.GRPCMethodName,
		GrpcServiceName:      data.GRPCServiceName,
		Description:          data.Description,
		IsPubliclyAccessible: data.IsPublic,
		AddressList:          data.AuthorizedAddresses,
		TrainingDataLink:     data.TrainingLink,
		ModelName:            data.ModelName,
		OrganizationId:       data.OrganizationId,
		ServiceId:            data.ServiceId,
		GroupId:              data.GroupId,
		UpdatedDate:          data.UpdatedDate,
		Status:               data.Status.String(),
	}
	return
}

func (service ModelService) updateModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
	key := service.getModelKeyToUpdate(request)
	oldAddresses := make([]string, 0)
	var latestAddresses []string
	// by default add the creator to the Authorized list of Address
	if request.UpdateModelDetails.AddressList != nil || len(request.UpdateModelDetails.AddressList) > 0 {
		latestAddresses = request.UpdateModelDetails.AddressList
	}
	latestAddresses = append(latestAddresses, request.Authorization.SignerAddress)
	if data, err = service.getModelDataForUpdate(request); err == nil && data != nil {
		oldAddresses = data.AuthorizedAddresses
		data.AuthorizedAddresses = latestAddresses
		latestAddresses = append(latestAddresses, request.Authorization.SignerAddress)
		data.IsPublic = request.UpdateModelDetails.IsPubliclyAccessible
		data.UpdatedByAddress = request.Authorization.SignerAddress
		if response != nil {
			data.Status = response.Status
		}
		data.ModelName = request.UpdateModelDetails.ModelName
		data.UpdatedDate = fmt.Sprintf("%v", time.Now().Format(DateFormat))
		data.Description = request.UpdateModelDetails.Description

		err = service.storage.Put(key, data)
		if err != nil {
			zap.L().Error("Error in putting data in user storage", zap.Error(err))
		}
		//get the difference of all the addresses b/w old and new
		updatedAddresses := difference(oldAddresses, latestAddresses)
		for _, address := range updatedAddresses {
			modelUserKey := getModelUserKey(key, address)
			modelUserData := service.getModelUserData(key, address)
			//if the address is present in the request but not in the old address , add it to the storage
			if isValuePresent(address, request.UpdateModelDetails.AddressList) {
				modelUserData.ModelIds = append(modelUserData.ModelIds, request.UpdateModelDetails.ModelId)
			} else { // the address was present in the old data , but not in new , hence needs to be deleted
				modelUserData.ModelIds = remove(modelUserData.ModelIds, request.UpdateModelDetails.ModelId)
			}
			err = service.userStorage.Put(modelUserKey, modelUserData)
			if err != nil {
				zap.L().Error("Error in putting data in storage", zap.Error(err))
			}
		}

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

// ensure only authorized use
func (service ModelService) verifySignerHasAccessToTheModel(serviceName string, methodName string, modelId string, address string) (err error) {
	key := &ModelUserKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  methodName,
		GRPCServiceName: serviceName,
		UserAddress:     address,
	}
	data, ok, err := service.userStorage.Get(key)
	if ok && err == nil {
		if !slices.Contains(data.ModelIds, modelId) {
			return fmt.Errorf("user %v, does not have access to model Id %v", address, modelId)
		}
	}
	return
}

func (service ModelService) updateModelDetailsWithLatestStatus(request *ModelDetailsRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
	key := &ModelKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  request.ModelDetails.GrpcMethodName,
		GRPCServiceName: request.ModelDetails.GrpcServiceName,
		ModelId:         request.ModelDetails.ModelId,
	}
	ok := false
	zap.L().Debug("updateModelDetailsWithLatestStatus: ", zap.Any("key", key))
	if data, ok, err = service.storage.Get(key); err == nil && ok {
		data.Status = response.Status
		if err = service.storage.Put(key, data); err != nil {
			zap.L().Error("issue with retrieving model data from storage", zap.Error(err))
		}
	} else {
		zap.L().Error("can't get model data from etcd", zap.Error(err))
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

func (service ModelService) getModelKeyToUpdate(request *UpdateModelRequest) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  request.UpdateModelDetails.GrpcMethodName,
		GRPCServiceName: request.UpdateModelDetails.GrpcServiceName,
		ModelId:         request.UpdateModelDetails.ModelId,
	}
	return
}

func (service ModelService) getModelDataForUpdate(request *UpdateModelRequest) (data *ModelData, err error) {
	key := service.getModelKeyToUpdate(request)
	ok := false

	if data, ok, err = service.storage.Get(key); err != nil || !ok {
		zap.L().Warn("unable to retrieve model data from storage", zap.String("Model Id", key.ModelId), zap.Error(err))
	}
	return
}

func (service ModelService) GetAllModels(c context.Context, request *AccessibleModelsRequest) (response *AccessibleModelsResponse, err error) {
	if request == nil || request.Authorization == nil {
		return &AccessibleModelsResponse{},
			fmt.Errorf("Invalid request , no Authorization provided ")
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &AccessibleModelsResponse{},
			fmt.Errorf("Unable to access model, %v", err)
	}
	if request.GetGrpcMethodName() == "" || request.GetGrpcServiceName() == "" {
		return &AccessibleModelsResponse{},
			fmt.Errorf("Invalid request, no GrpcMethodName or GrpcServiceName provided")
	}

	key := &ModelUserKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         service.organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  request.GrpcMethodName,
		GRPCServiceName: request.GrpcServiceName,
		UserAddress:     request.Authorization.SignerAddress,
	}

	modelDetailsArray := make([]*ModelDetails, 0)
	if data, ok, err := service.userStorage.Get(key); data != nil && ok && err == nil {
		for _, modelId := range data.ModelIds {
			modelKey := &ModelKey{
				OrganizationId:  config.GetString(config.OrganizationId),
				ServiceId:       config.GetString(config.ServiceId),
				GroupId:         service.organizationMetaData.GetGroupIdString(),
				GRPCMethodName:  request.GrpcMethodName,
				GRPCServiceName: request.GrpcServiceName,
				ModelId:         modelId,
			}
			if modelData, modelOk, modelErr := service.storage.Get(modelKey); modelOk && modelData != nil && modelErr == nil {
				boModel := convertModelDataToBO(modelData)
				modelDetailsArray = append(modelDetailsArray, boModel)
			}
		}
	}

	for _, model := range modelDetailsArray {
		zap.L().Debug("Model", zap.String("Name", model.ModelName))
	}

	response = &AccessibleModelsResponse{
		ListOfModels: modelDetailsArray,
	}
	return
}

func (service ModelService) getModelDataToCreate(request *CreateModelRequest, response *ModelDetailsResponse) (data *ModelData) {

	data = &ModelData{
		Status:              response.Status,
		GRPCServiceName:     request.ModelDetails.GrpcServiceName,
		GRPCMethodName:      request.ModelDetails.GrpcMethodName,
		CreatedByAddress:    request.Authorization.SignerAddress,
		UpdatedByAddress:    request.Authorization.SignerAddress,
		AuthorizedAddresses: request.ModelDetails.AddressList,
		Description:         request.ModelDetails.Description,
		ModelName:           request.ModelDetails.ModelName,
		TrainingLink:        request.ModelDetails.TrainingDataLink,
		IsPublic:            request.ModelDetails.IsPubliclyAccessible,
		IsDefault:           false,
		ModelId:             response.ModelDetails.ModelId,
		OrganizationId:      config.GetString(config.OrganizationId),
		ServiceId:           config.GetString(config.ServiceId),
		GroupId:             service.organizationMetaData.GetGroupIdString(),
		UpdatedDate:         fmt.Sprintf("%v", time.Now().Format(DateFormat)),
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
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		zap.L().Error(err.Error())
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("unable to create Model: %v", err)
	}
	if request.GetModelDetails().GrpcServiceName == "" || request.GetModelDetails().GrpcMethodName == "" {
		zap.L().Error("Error in getting grpc service name", zap.Error(err))
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no GrpcServiceName or GrpcMethodName provided  , %v", err)
	}

	// make a call to the client
	// if the response is successful, store details in etcd
	// send back the response to the client
	conn, client, err := service.getServiceClient()
	if err != nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for Model Training %v", err)
	}

	response, err = client.CreateModel(c, request)
	if err != nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for Model Training %v", err)
	}

	if response.ModelDetails == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for Model Training: service return empty ModelDetails")
	}

	//store the details in etcd
	zap.L().Info("Creating model based on response from CreateModel of training service")

	data, err := service.createModelDetails(request, response)
	if err != nil {
		zap.L().Error(err.Error())
		return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
	}
	response = BuildModelResponseFrom(data, response.Status)
	deferConnection(conn)
	return
}

func BuildModelResponseFrom(data *ModelData, status Status) *ModelDetailsResponse {
	return &ModelDetailsResponse{
		Status: status,
		ModelDetails: &ModelDetails{
			ModelId:              data.ModelId,
			GrpcMethodName:       data.GRPCMethodName,
			GrpcServiceName:      data.GRPCServiceName,
			Description:          data.Description,
			IsPubliclyAccessible: data.IsPublic,
			AddressList:          data.AuthorizedAddresses,
			TrainingDataLink:     data.TrainingLink,
			ModelName:            data.ModelName,
			OrganizationId:       config.GetString(config.OrganizationId),
			ServiceId:            config.GetString(config.ServiceId),
			GroupId:              data.GroupId,
			Status:               status.String(),
			UpdatedDate:          data.UpdatedDate,
		},
	}
}
func (service ModelService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	if err = service.verifySignerHasAccessToTheModel(request.UpdateModelDetails.GrpcServiceName,
		request.UpdateModelDetails.GrpcMethodName, request.UpdateModelDetails.ModelId, request.Authorization.SignerAddress); err != nil {
		return &ModelDetailsResponse{},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	if request.UpdateModelDetails == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf(" Invalid request , no UpdateModelDetails provided  , %v", err)
	}

	zap.L().Info("Updating model based on response from UpdateModel")
	if data, err := service.updateModelDetails(request, response); err == nil && data != nil {
		response = BuildModelResponseFrom(data, data.Status)

	} else {
		return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
	}

	return
}

func (service ModelService) DeleteModel(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	if err = service.verifySignerHasAccessToTheModel(request.UpdateModelDetails.GrpcServiceName,
		request.UpdateModelDetails.GrpcMethodName, request.UpdateModelDetails.ModelId, request.Authorization.SignerAddress); err != nil {
		return &ModelDetailsResponse{},
			fmt.Errorf(" Unable to access model , %v", err)
	}
	if request.UpdateModelDetails == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf(" Invalid request: UpdateModelDetails are empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	if conn, client, err := service.getServiceClient(); err == nil {
		response, err = client.DeleteModel(ctx, request)
		if response == nil || err != nil {
			zap.L().Error("error in invoking DeleteModel, service-provider should realize it", zap.Error(err))
			return &ModelDetailsResponse{Status: Status_ERRORED}, fmt.Errorf("error in invoking DeleteModel, service-provider should realize it")
		}
		if data, err := service.deleteModelDetails(request); err == nil && data != nil {
			response = BuildModelResponseFrom(data, response.Status)
		} else {
			zap.L().Error("issue with deleting ModelId in storage", zap.Error(err))
			return response, fmt.Errorf("issue with deleting Model Id in Storage %v", err)
		}
		deferConnection(conn)
	} else {
		return &ModelDetailsResponse{Status: Status_ERRORED}, fmt.Errorf("error in invoking service for Model Training")
	}

	return
}

func (service ModelService) GetModelStatus(c context.Context, request *ModelDetailsRequest) (response *ModelDetailsResponse,
	err error) {
	if request == nil || request.Authorization == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no Authorization provided  , %v", err)
	}
	if err = service.verifySignature(request.Authorization); err != nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("unable to access model , %v", err)
	}
	if err = service.verifySignerHasAccessToTheModel(request.ModelDetails.GrpcServiceName,
		request.ModelDetails.GrpcMethodName, request.ModelDetails.ModelId, request.Authorization.SignerAddress); err != nil {
		return &ModelDetailsResponse{},
			fmt.Errorf("unable to access model , %v", err)
	}
	if request.ModelDetails == nil {
		return &ModelDetailsResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request: ModelDetails can't be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	if conn, client, err := service.getServiceClient(); err == nil {
		response, err = client.GetModelStatus(ctx, request)
		if response == nil || err != nil {
			zap.L().Error("error in invoking GetModelStatus, service-provider should realize it", zap.Error(err))
			return &ModelDetailsResponse{Status: Status_ERRORED}, fmt.Errorf("error in invoking GetModelStatus, service-provider should realize it")
		}
		zap.L().Info("[GetModelStatus] response from service-provider", zap.Any("response", response))
		zap.L().Info("[GetModelStatus] updating model status based on response from UpdateModel")
		data, err := service.updateModelDetailsWithLatestStatus(request, response)
		zap.L().Debug("[GetModelStatus] data that be returned to client", zap.Any("data", data))
		if err == nil && data != nil {
			response = BuildModelResponseFrom(data, response.Status)
		} else {
			zap.L().Error("[GetModelStatus] BuildModelResponseFrom error", zap.Error(err))
			return response, fmt.Errorf("[GetModelStatus] issue with storing Model Id in the Daemon Storage %v", err)
		}
		deferConnection(conn)
	} else {
		return &ModelDetailsResponse{Status: Status_ERRORED}, fmt.Errorf("[GetModelStatus] error in invoking service for Model Training")
	}
	return
}

// message used to sign is of the form ("__create_model", mpe_address, current_block_number)
func (service *ModelService) verifySignature(request *AuthorizationDetails) error {
	return utils.VerifySigner(service.getMessageBytes(request.Message, request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

// "user passed message	", user_address, current_block_number
func (service *ModelService) getMessageBytes(prefixMessage string, request *AuthorizationDetails) []byte {
	userAddress := utils.ToChecksumAddress(request.SignerAddress)
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		userAddress.Bytes(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return message
}

func NewModelService(channelService escrow.PaymentChannelService, serMetaData *blockchain.ServiceMetadata,
	orgMetadata *blockchain.OrganizationMetaData, storage *ModelStorage, userStorage *ModelUserStorage) ModelServer {
	serviceURL := config.GetString(config.ModelMaintenanceEndPoint)
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
