//go:generate protoc -I . ./training_v2.proto --go-grpc_out=. --go_out=.
//go:generate protoc -I . ./training_daemon.proto --go-grpc_out=. --go_out=.

package training

import (
	"context"
	"fmt"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/errs"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	_ "embed"
	"github.com/singnet/snet-daemon/v5/config"
	"github.com/singnet/snet-daemon/v5/escrow"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	DateFormat = "02-01-2006"
)

//go:embed training_v2.proto
var TrainingProtoEmbeded string

//type IService interface {
//}

// ModelService this is remote AI service provider
type ModelService struct {
	serviceMetaData      *blockchain.ServiceMetadata
	organizationMetaData *blockchain.OrganizationMetaData
	channelService       escrow.PaymentChannelService
	storage              *ModelStorage
	userStorage          *ModelUserStorage
	serviceUrl           string
}

type DaemonService struct {
	serviceMetaData      *blockchain.ServiceMetadata
	organizationMetaData *blockchain.OrganizationMetaData
	channelService       escrow.PaymentChannelService
	storage              *ModelStorage
	userStorage          *ModelUserStorage
	serviceUrl           string
	trainingMetadata     *TrainingMetadata
	methodsMetadata      map[string]*MethodMetadata
}

func (ds *DaemonService) CreateModel(c context.Context, request *NewModelRequest) (*ModelResponse, error) {

	if request == nil || request.Authorization == nil {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no authorization provided")
	}

	if err := ds.verifySignature(request.Authorization); err != nil {
		zap.L().Error("unable to create model", zap.Error(err))
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("unable to create model: %v", err)
	}

	if request.GetModel().GrpcServiceName == "" || request.GetModel().GrpcMethodName == "" {
		zap.L().Error("invalid request, no grpc_method_name or grpc_service_name provided")
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no grpc_service_name or grpc_method_name provided")
	}

	request.Model.ServiceId = config.GetString(config.ServiceId)
	request.Model.OrganizationId = config.GetString(config.OrganizationId)
	request.Model.GroupId = config.GetString(config.DaemonGroupName)

	// make a call to the client
	// if the response is successful, store details in etcd
	// send back the response to the client
	conn, client, err := ds.getServiceClient()
	deferConnection(conn)
	if err != nil {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for Model Training %v", err)
	}

	responseModelID, errClient := client.CreateModel(c, request.Model)
	if errClient != nil {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for Model Training %v", errClient)
	}

	if responseModelID == nil {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for model training: service return empty response")
	}

	if responseModelID.ModelId == "" {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("error in invoking service for model training: service return empty modelID")
	}

	//store the details in etcd
	zap.L().Info("Creating model based on response from CreateModel of training service")

	data, err := ds.createModelDetails(request, responseModelID)
	if err != nil {
		zap.L().Error(err.Error())
		return nil, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
	}
	modelResponse := BuildModelResponse(data, Status_CREATED)
	return modelResponse, err
}

func (ds *DaemonService) ValidateModelPrice(ctx context.Context, request *AuthValidateRequest) (*PriceInBaseUnit, error) {
	panic("implement me")
}

func (ds *DaemonService) UploadAndValidate(server Daemon_UploadAndValidateServer) error {
	panic("implement me")
}

func (ds *DaemonService) ValidateModel(ctx context.Context, request *AuthValidateRequest) (*StatusResponse, error) {
	panic("implement me")
}

func (ds *DaemonService) TrainModelPrice(ctx context.Context, request *CommonRequest) (*PriceInBaseUnit, error) {
	panic("implement me")
}

func (ds *DaemonService) TrainModel(ctx context.Context, request *CommonRequest) (*StatusResponse, error) {
	panic("implement me")
}

func (ds *DaemonService) GetTrainingMetadata(ctx context.Context, empty *emptypb.Empty) (*TrainingMetadata, error) {
	return ds.trainingMetadata, nil
}

func (ds *DaemonService) UpdateModel(ctx context.Context, request *UpdateModelRequest) (*ModelResponse, error) {
	panic("implement me")
}

func (ds *DaemonService) GetMethodMetadata(ctx context.Context, request *MethodMetadataRequest) (*MethodMetadata, error) {
	if request.GetModelId() != "" {
		// TODO get from etcd
	}
	key := request.GrpcServiceName + request.GrpcMethodName
	return ds.methodsMetadata[key], nil
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

func (ds *DaemonService) getServiceClient() (conn *grpc.ClientConn, client ModelClient, err error) {
	conn = getConnection(ds.serviceUrl)
	client = NewModelClient(conn)
	return
}

func (ds *DaemonService) createModelDetails(request *NewModelRequest, response *ModelID) (data *ModelData, err error) {
	key := ds.buildModelKey(response.ModelId)
	data = ds.getModelDataToCreate(request, response)
	//store the model details in etcd
	zap.L().Debug("createModelDetails", zap.Any("key", key))
	err = ds.storage.Put(key, data)
	if err != nil {
		zap.L().Error("can't put model in etcd", zap.Error(err))
		return
	}
	//for every accessible address in the list, store the user address and all the model Ids associated with it
	for _, address := range data.AuthorizedAddresses {
		userKey := getModelUserKey(key, address)
		userData := ds.getModelUserData(key, address)
		zap.L().Debug("createModelDetails", zap.Any("userKey", userKey))
		err = ds.userStorage.Put(userKey, userData)
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
		OrganizationId: key.OrganizationId,
		ServiceId:      key.ServiceId,
		GroupId:        key.GroupId,
		//GRPCMethodName:  key.GRPCMethodName,
		//GRPCServiceName: key.GRPCServiceName,
		UserAddress: address,
	}
}

func (ds *DaemonService) getModelUserData(key *ModelKey, address string) *ModelUserData {
	//Check if there are any model Ids already associated with this user
	modelIds := make([]string, 0)
	userKey := getModelUserKey(key, address)
	zap.L().Debug("user model key is" + userKey.String())
	data, ok, err := ds.userStorage.Get(userKey)
	if ok && err == nil && data != nil {
		modelIds = data.ModelIds
	}
	if err != nil {
		zap.L().Error("can't get model data from etcd", zap.Error(err))
	}
	modelIds = append(modelIds, key.ModelId)
	return &ModelUserData{
		OrganizationId: key.OrganizationId,
		ServiceId:      key.ServiceId,
		GroupId:        key.GroupId,
		//GRPCMethodName:  key.GRPCMethodName,
		//GRPCServiceName: key.GRPCServiceName,
		UserAddress: address,
		ModelIds:    modelIds,
	}
}

func (ds DaemonService) deleteUserModelDetails(key *ModelKey, data *ModelData) (err error) {
	for _, address := range data.AuthorizedAddresses {
		userKey := getModelUserKey(key, address)
		if data, ok, err := ds.userStorage.Get(userKey); ok && err == nil && data != nil {
			data.ModelIds = remove(data.ModelIds, key.ModelId)
			err = ds.userStorage.Put(userKey, data)
			if err != nil {
				zap.L().Error(err.Error())
			}
		}
	}
	return
}

func (ds *DaemonService) deleteModelDetails(req *CommonRequest) (data *ModelData, err error) {
	key := ds.getModelKeyToUpdate(req.ModelId)
	ok := false
	data, ok, err = ds.storage.Get(key)
	if ok && err == nil {
		data.Status = Status_DELETED
		data.UpdatedDate = fmt.Sprintf("%v", time.Now().Format(DateFormat))
		err = ds.storage.Put(key, data)
		err = ds.deleteUserModelDetails(key, data)
	}
	return
}

func convertModelDataToBO(data *ModelData) (responseData *ModelResponse) {
	responseData = &ModelResponse{
		ModelId:          data.ModelId,
		GrpcMethodName:   data.GRPCMethodName,
		GrpcServiceName:  data.GRPCServiceName,
		Description:      data.Description,
		IsPublic:         data.IsPublic,
		AddressList:      data.AuthorizedAddresses,
		TrainingDataLink: data.TrainingLink,
		Name:             data.ModelName,
		//OrganizationId:       data.OrganizationId,
		//ServiceId:            data.ServiceId,
		//GroupId:              data.GroupId,
		UpdatedDate: data.UpdatedDate,
		Status:      data.Status,
	}
	return
}

//func (ds *DaemonService) updateModelDetails(request *UpdateModelRequest, response *ModelDetailsResponse) (data *ModelData, err error) {
//	key := ds.getModelKeyToUpdate(request.Mo)
//	oldAddresses := make([]string, 0)
//	var latestAddresses []string
//	// by default add the creator to the Authorized list of Address
//	if request.UpdateModelDetails.AddressList != nil || len(request.UpdateModelDetails.AddressList) > 0 {
//		latestAddresses = request.UpdateModelDetails.AddressList
//	}
//	latestAddresses = append(latestAddresses, request.Authorization.SignerAddress)
//	if data, err = ds.getModelDataForUpdate(request); err == nil && data != nil {
//		oldAddresses = data.AuthorizedAddresses
//		data.AuthorizedAddresses = latestAddresses
//		latestAddresses = append(latestAddresses, request.Authorization.SignerAddress)
//		data.IsPublic = request.UpdateModelDetails.IsPubliclyAccessible
//		data.UpdatedByAddress = request.Authorization.SignerAddress
//		if response != nil {
//			data.Status = response.Status
//		}
//		data.ModelName = request.UpdateModelDetails.ModelName
//		data.UpdatedDate = fmt.Sprintf("%v", time.Now().Format(DateFormat))
//		data.Description = request.UpdateModelDetails.Description
//
//		err = ds.storage.Put(key, data)
//		if err != nil {
//			zap.L().Error("Error in putting data in user storage", zap.Error(err))
//		}
//		//get the difference of all the addresses b/w old and new
//		updatedAddresses := difference(oldAddresses, latestAddresses)
//		for _, address := range updatedAddresses {
//			modelUserKey := getModelUserKey(key, address)
//			modelUserData := service.getModelUserData(key, address)
//			//if the address is present in the request but not in the old address , add it to the storage
//			if isValuePresent(address, request.UpdateModelDetails.AddressList) {
//				modelUserData.ModelIds = append(modelUserData.ModelIds, request.UpdateModelDetails.ModelId)
//			} else { // the address was present in the old data , but not in new , hence needs to be deleted
//				modelUserData.ModelIds = remove(modelUserData.ModelIds, request.UpdateModelDetails.ModelId)
//			}
//			err = ds.userStorage.Put(modelUserKey, modelUserData)
//			if err != nil {
//				zap.L().Error("Error in putting data in storage", zap.Error(err))
//			}
//		}
//
//	}
//	return
//}

// ensure only authorized use
func (ds DaemonService) verifySignerHasAccessToTheModel(modelId string, address string) (err error) {
	key := &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        ds.organizationMetaData.GetGroupIdString(),
		//GRPCMethodName:  methodName,
		//GRPCServiceName: serviceName,
		UserAddress: address,
	}
	data, ok, err := ds.userStorage.Get(key)
	if ok && err == nil {
		if !slices.Contains(data.ModelIds, modelId) {
			return fmt.Errorf("user %v, does not have access to model Id %v", address, modelId)
		}
	}
	return
}

func (ds DaemonService) updateModelDetailsWithLatestStatus(request *CommonRequest, response *ModelResponse) (data *ModelData, err error) {
	key := &ModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        ds.organizationMetaData.GetGroupIdString(),
		//GRPCMethodName:  request.ModelDetails.GrpcMethodName,
		//GRPCServiceName: request.ModelDetails.GrpcServiceName,
		ModelId: request.ModelId,
	}
	ok := false
	zap.L().Debug("updateModelDetailsWithLatestStatus: ", zap.Any("key", key))
	if data, ok, err = ds.storage.Get(key); err == nil && ok {
		data.Status = response.Status
		if err = ds.storage.Put(key, data); err != nil {
			zap.L().Error("issue with retrieving model data from storage", zap.Error(err))
		}
	} else {
		zap.L().Error("can't get model data from etcd", zap.Error(err))
	}
	return
}

func (ds *DaemonService) buildModelKey(modelID string) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        ds.organizationMetaData.GetGroupIdString(),
		//GRPCMethodName:  request.Model.GrpcMethodName,
		//GRPCServiceName: request.Model.GrpcServiceName,
		ModelId: modelID,
	}
	return
}

func (ds *DaemonService) getModelKeyToUpdate(modelID string) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        ds.organizationMetaData.GetGroupIdString(),
		//GRPCMethodName:  request.UpdateModelDetails.GrpcMethodName,
		//GRPCServiceName: request.UpdateModelDetails.GrpcServiceName,
		ModelId: modelID,
	}
	return
}

//func (ds *DaemonService) getModelDataForUpdate(request *UpdateModelRequest) (data *ModelData, err error) {
//	key := ds.getModelKeyToUpdate(request)
//	ok := false
//
//	if data, ok, err = ds.storage.Get(key); err != nil || !ok {
//		zap.L().Warn("unable to retrieve model data from storage", zap.String("Model Id", key.ModelId), zap.Error(err))
//	}
//	return
//}

func (ds *DaemonService) GetAllModels(c context.Context, request *AllModelsRequest) (response *ModelsResponse, err error) {
	if request == nil || request.Authorization == nil {
		return &ModelsResponse{},
			fmt.Errorf("Invalid request , no Authorization provided ")
	}
	if err = ds.verifySignature(request.Authorization); err != nil {
		return &ModelsResponse{},
			fmt.Errorf("Unable to access model, %v", err)
	}
	//if request.GetGrpcMethodName() == "" || request.GetGrpcServiceName() == "" {
	//	return &AccessibleModelsResponse{},
	//		fmt.Errorf("Invalid request, no GrpcMethodName or GrpcServiceName provided")
	//}

	key := &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        ds.organizationMetaData.GetGroupIdString(),
		//GRPCMethodName:  request.GrpcMethodName,
		//GRPCServiceName: request.GrpcServiceName,
		UserAddress: request.Authorization.SignerAddress,
	}

	modelDetailsArray := make([]*ModelResponse, 0)
	if data, ok, err := ds.userStorage.Get(key); data != nil && ok && err == nil {
		for _, modelId := range data.ModelIds {
			modelKey := &ModelKey{
				OrganizationId: config.GetString(config.OrganizationId),
				ServiceId:      config.GetString(config.ServiceId),
				GroupId:        ds.organizationMetaData.GetGroupIdString(),
				//GRPCMethodName:  request.GrpcMethodName,
				//GRPCServiceName: request.GrpcServiceName,
				ModelId: modelId,
			}
			if modelData, modelOk, modelErr := ds.storage.Get(modelKey); modelOk && modelData != nil && modelErr == nil {
				boModel := convertModelDataToBO(modelData)
				modelDetailsArray = append(modelDetailsArray, boModel)
			}
		}
	}

	for _, model := range modelDetailsArray {
		zap.L().Debug("Model", zap.String("Name", model.Name))
	}

	response = &ModelsResponse{
		ListOfModels: modelDetailsArray,
	}
	return
}

func (ds DaemonService) getModelDataToCreate(request *NewModelRequest, response *ModelID) (data *ModelData) {
	data = &ModelData{
		Status:              Status_CREATED,
		GRPCServiceName:     request.Model.GrpcServiceName,
		GRPCMethodName:      request.Model.GrpcMethodName,
		CreatedByAddress:    request.Authorization.SignerAddress,
		UpdatedByAddress:    request.Authorization.SignerAddress,
		AuthorizedAddresses: request.Model.AddressList,
		Description:         request.Model.Description,
		ModelName:           request.Model.Name,
		//TrainingLink:
		IsPublic:       request.Model.IsPublic,
		IsDefault:      false,
		ModelId:        response.ModelId,
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        ds.organizationMetaData.GetGroupIdString(),
		UpdatedDate:    fmt.Sprintf("%v", time.Now().Format(DateFormat)),
	}
	//by default add the creator to the Authorized list of Address
	if data.AuthorizedAddresses == nil {
		data.AuthorizedAddresses = make([]string, 0)
	}
	data.AuthorizedAddresses = append(data.AuthorizedAddresses, data.CreatedByAddress)
	return
}

func BuildModelResponse(data *ModelData, status Status) *ModelResponse {
	return &ModelResponse{
		Status:           status,
		ModelId:          data.ModelId,
		GrpcMethodName:   data.GRPCMethodName,
		GrpcServiceName:  data.GRPCServiceName,
		Description:      data.Description,
		IsPublic:         data.IsPublic,
		AddressList:      data.AuthorizedAddresses,
		TrainingDataLink: data.TrainingLink,
		Name:             data.ModelName,
		//OrganizationId:   config.GetString(config.OrganizationId),
		//ServiceId:        config.GetString(config.ServiceId),
		//GroupId:          data.GroupId,
		//Status:               status.String(),
		UpdatedDate: data.UpdatedDate,
	}
}

//func (ds *DaemonService) UpdateModelAccess(c context.Context, request *UpdateModelRequest) (response *ModelDetailsResponse,
//	err error) {
//	if request == nil || request.Authorization == nil {
//		return &ModelDetailsResponse{Status: Status_ERRORED},
//			fmt.Errorf(" Invalid request , no Authorization provided  , %v", err)
//	}
//	if err = ds.verifySignature(request.Authorization); err != nil {
//		return &ModelDetailsResponse{Status: Status_ERRORED},
//			fmt.Errorf(" Unable to access model , %v", err)
//	}
//	if err = service.verifySignerHasAccessToTheModel(request.UpdateModelDetails.GrpcServiceName,
//		request.UpdateModelDetails.GrpcMethodName, request.UpdateModelDetails.ModelId, request.Authorization.SignerAddress); err != nil {
//		return &ModelDetailsResponse{},
//			fmt.Errorf(" Unable to access model , %v", err)
//	}
//	if request.UpdateModelDetails == nil {
//		return &ModelDetailsResponse{Status: Status_ERRORED},
//			fmt.Errorf(" Invalid request , no UpdateModelDetails provided  , %v", err)
//	}
//
//	zap.L().Info("Updating model based on response from UpdateModel")
//	if data, err := service.updateModelDetails(request, response); err == nil && data != nil {
//		response = BuildModelResponse(data, data.Status)
//
//	} else {
//		return response, fmt.Errorf("issue with storing Model Id in the Daemon Storage %v", err)
//	}
//
//	return
//}

func (ds *DaemonService) DeleteModel(c context.Context, req *CommonRequest) (*StatusResponse, error) {

	if req == nil || req.Authorization == nil {
		return &StatusResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no Authorization provided")
	}

	if req.ModelId == "" {
		return &StatusResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request: ModelId is empty")
	}

	if err := ds.verifySignature(req.Authorization); err != nil {
		return &StatusResponse{Status: Status_ERRORED},
			fmt.Errorf("unable to access model , %v", err)
	}

	if err := ds.verifySignerHasAccessToTheModel(req.ModelId, req.Authorization.SignerAddress); err != nil {
		return &StatusResponse{},
			fmt.Errorf("unable to access model , %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	conn, client, err := ds.getServiceClient()
	if err != nil {
		return &StatusResponse{Status: Status_ERRORED}, fmt.Errorf("error in invoking service for Model Training")
	}
	deferConnection(conn)
	response, err := client.DeleteModel(ctx, &ModelID{ModelId: req.ModelId})
	if response == nil || err != nil {
		zap.L().Error("error in invoking DeleteModel, service-provider should realize it", zap.Error(err))
		return &StatusResponse{Status: Status_ERRORED}, fmt.Errorf("error in invoking DeleteModel, service-provider should realize it")
	}
	data, err := ds.deleteModelDetails(req)
	if err == nil && data != nil {
		zap.L().Error("issue with deleting ModelId in storage", zap.Error(err))
		return response, fmt.Errorf("issue with deleting Model Id in Storage %v", err)
	}
	//responseData := BuildModelResponse(data, response.Status)
	return &StatusResponse{Status: Status_DELETED}, err
}

func (ds *DaemonService) GetModel(c context.Context, request *CommonRequest) (response *ModelResponse,
	err error) {
	if request == nil || request.Authorization == nil {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request, no Authorization provided  , %v", err)
	}
	if err = ds.verifySignature(request.Authorization); err != nil {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("unable to access model , %v", err)
	}
	if err = ds.verifySignerHasAccessToTheModel(request.ModelId, request.Authorization.SignerAddress); err != nil {
		return &ModelResponse{},
			fmt.Errorf("unable to access model , %v", err)
	}
	if request.ModelId == "" {
		return &ModelResponse{Status: Status_ERRORED},
			fmt.Errorf("invalid request: ModelId can't be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	if conn, client, err := ds.getServiceClient(); err == nil {
		responseStatus, err := client.GetModelStatus(ctx, &ModelID{ModelId: request.ModelId})
		if responseStatus == nil || err != nil {
			zap.L().Error("error in invoking GetModelStatus, service-provider should realize it", zap.Error(err))
			return &ModelResponse{Status: Status_ERRORED}, fmt.Errorf("error in invoking GetModelStatus, service-provider should realize it")
		}
		zap.L().Info("[GetModelStatus] response from service-provider", zap.Any("response", response))
		zap.L().Info("[GetModelStatus] updating model status based on response from UpdateModel")
		data, err := ds.updateModelDetailsWithLatestStatus(request, response)
		zap.L().Debug("[GetModelStatus] data that be returned to client", zap.Any("data", data))
		if err == nil && data != nil {
			response = BuildModelResponse(data, responseStatus.Status)
		} else {
			zap.L().Error("[GetModelStatus] BuildModelResponse error", zap.Error(err))
			return response, fmt.Errorf("[GetModelStatus] issue with storing Model Id in the Daemon Storage %v", err)
		}
		deferConnection(conn)
	} else {
		return &ModelResponse{Status: Status_ERRORED}, fmt.Errorf("[GetModelStatus] error in invoking service for Model Training")
	}
	return
}

// NewModelService AI service prodiver
// TODO maybe remove
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

// NewTrainingService daemon self server
func NewTrainingService(channelService escrow.PaymentChannelService, serMetaData *blockchain.ServiceMetadata,
	orgMetadata *blockchain.OrganizationMetaData, storage *ModelStorage, userStorage *ModelUserStorage) DaemonServer {

	linkerFiles := getFileDescriptors(serMetaData.ProtoFiles)
	serMetaData.ProtoDescriptors = linkerFiles
	methodsMD, trainMD, err := parseTrainingMetadata(linkerFiles)
	if err != nil {
		// TODO
		return nil
	}

	serviceURL := config.GetString(config.ModelMaintenanceEndPoint)
	if config.IsValidUrl(serviceURL) && config.GetBool(config.BlockchainEnabledKey) {
		return &DaemonService{
			channelService:       channelService,
			serviceMetaData:      serMetaData,
			organizationMetaData: orgMetadata,
			storage:              storage,
			userStorage:          userStorage,
			serviceUrl:           serviceURL,
			trainingMetadata:     &trainMD,
			methodsMetadata:      methodsMD,
		}
	} else {
		return &NoTrainingService{}
	}
}

// parseTrainingMetadata TODO add comment
func parseTrainingMetadata(protos linker.Files) (methodsMD map[string]*MethodMetadata, trainingMD TrainingMetadata, err error) {
	methodsMD = make(map[string]*MethodMetadata)
	trainingMD.TrainingMethods = make(map[string]*structpb.ListValue)

	for _, protoFile := range protos {
		for servicesCounter := 0; servicesCounter < protoFile.Services().Len(); servicesCounter++ {
			protoService := protoFile.Services().Get(servicesCounter)
			if protoService == nil {
				continue
			}
			for methodsCounter := 0; methodsCounter < protoService.Methods().Len(); methodsCounter++ {
				method := protoFile.Services().Get(servicesCounter).Methods().Get(methodsCounter)
				if method == nil {
					continue
				}
				inputFields := method.Input().Fields()
				if inputFields == nil {
					continue
				}
				for fieldsCounter := 0; fieldsCounter < inputFields.Len(); fieldsCounter++ {
					if inputFields.Get(fieldsCounter).Message() != nil {
						// if the method accepts modelId, then we consider it as training
						if inputFields.Get(fieldsCounter).Message().FullName() == "trainingV2.ModelID" {
							// init array if nil
							if trainingMD.TrainingMethods[string(protoService.Name())] == nil {
								trainingMD.TrainingMethods[string(protoService.Name())], _ = structpb.NewList(nil)
							}
							value := structpb.NewStringValue(string(method.Name()))
							trainingMD.TrainingMethods[string(protoService.Name())].Values = append(trainingMD.TrainingMethods[string(protoService.Name())].Values, value)
						}
					}
				}

				methodOptions, ok := method.Options().(*descriptorpb.MethodOptions)
				if ok && methodOptions != nil {
					key := string(protoService.Name() + method.Name())
					methodsMD[key] = &MethodMetadata{}
					if proto.HasExtension(methodOptions, E_DatasetDescription) {
						if datasetDesc, ok := proto.GetExtension(methodOptions, E_DatasetDescription).(string); ok {
							methodsMD[key].DatasetDescription = datasetDesc
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetType) {
						if datasetType, ok := proto.GetExtension(methodOptions, E_DatasetType).(string); ok {
							methodsMD[key].DatasetType = datasetType
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetFilesType) {
						if datasetDesc, ok := proto.GetExtension(methodOptions, E_DatasetFilesType).(string); ok {
							methodsMD[key].DatasetFilesType = datasetDesc
						}
					}
					if proto.HasExtension(methodOptions, E_MaxModelsPerUser) {
						if datasetDesc, ok := proto.GetExtension(methodOptions, E_MaxModelsPerUser).(uint64); ok {
							methodsMD[key].MaxModelsPerUser = datasetDesc
						}
					}
					if proto.HasExtension(methodOptions, E_DefaultModelId) {
						if defaultModelId, ok := proto.GetExtension(methodOptions, E_DefaultModelId).(string); ok {
							methodsMD[key].DefaultModelId = defaultModelId
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetMaxSizeSingleFileMb) {
						if d, ok := proto.GetExtension(methodOptions, E_DatasetMaxSizeSingleFileMb).(uint64); ok {
							methodsMD[key].DatasetMaxSizeSingleFileMb = d
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetMaxCountFiles) {
						if maxCountFiles, ok := proto.GetExtension(methodOptions, E_DatasetMaxCountFiles).(uint64); ok {
							methodsMD[key].DatasetMaxCountFiles = maxCountFiles
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetMaxSizeMb) {
						if datasetMaxSizeMb, ok := proto.GetExtension(methodOptions, E_DatasetMaxSizeMb).(uint64); ok {
							methodsMD[key].DatasetMaxSizeMb = datasetMaxSizeMb
						}
					}
					if methodsMD[key].DefaultModelId != "" {
						zap.L().Debug("training metadata", zap.String("method", string(method.Name())), zap.String("key", key), zap.Any("metadata", methodsMD[key]))
					}
				}
			}
		}
	}
	zap.L().Debug("training methods", zap.Any("methods", trainingMD.TrainingMethods))
	return
}

func getFileDescriptors(protoFiles map[string]string) linker.Files {
	protoFiles["training_v2.proto"] = TrainingProtoEmbeded
	accessor := protocompile.SourceAccessorFromMap(protoFiles)
	r := protocompile.WithStandardImports(&protocompile.SourceResolver{Accessor: accessor})
	compiler := protocompile.Compiler{
		Resolver:       r,
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := compiler.Compile(context.Background(), slices.Collect(maps.Keys(protoFiles))...)
	if err != nil || fds == nil {
		zap.L().Fatal("failed to analyze protofile"+errs.ErrDescURL(errs.InvalidProto), zap.Error(err))
	}
	return fds
}
