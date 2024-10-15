package training

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var count int = 0

type ModelServiceTestSuite struct {
	suite.Suite
	serviceURL            string
	server                *grpc.Server
	mockService           MockServiceModelGRPCImpl
	service               ModelServer
	serviceNotImplemented ModelServer
	senderPvtKy           *ecdsa.PrivateKey
	senderAddress         common.Address

	alternateUserPvtKy   *ecdsa.PrivateKey
	alternateUserAddress common.Address
}

func TestModelServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ModelServiceTestSuite))
}
func (suite *ModelServiceTestSuite) getGRPCServerAndServe() {
	config.Vip().Set(config.ModelMaintenanceEndPoint, "http://localhost:2222")
	config.Vip().Set(config.ModelTrainingEndpoint, "http://localhost:2222")
	ch := make(chan int)
	go func() {
		listener, err := net.Listen("tcp", ":2222")
		if err != nil {
			panic(err)
		}
		suite.server = grpc.NewServer()

		RegisterModelServer(suite.server, suite.mockService)
		ch <- 0
		suite.server.Serve(listener)

	}()

	_ = <-ch
}
func (suite *ModelServiceTestSuite) SetupSuite() {
	suite.serviceNotImplemented = NewModelService(nil, nil, nil, nil, nil)
	config.Vip().Set(config.ModelMaintenanceEndPoint, "localhost:2222")
	suite.mockService = MockServiceModelGRPCImpl{}
	suite.serviceURL = config.GetString(config.ModelMaintenanceEndPoint)
	suite.getGRPCServerAndServe()

	testJsonOrgGroupData := "{   \"org_name\": \"organization_name\",   \"org_id\": \"ExampleOrganizationId\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	testJsonData := "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
		"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x94d04332C4f5273feF69c4a52D24f42a3aF1F207\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

	orgMetaData, _ := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	serviceMetaData, _ := blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
	suite.service = NewModelService(nil, serviceMetaData, orgMetaData,
		NewModelStorage(storage.NewMemStorage()), NewUerModelStorage(storage.NewMemStorage()))
	suite.senderPvtKy, _ = crypto.GenerateKey()
	suite.senderAddress = crypto.PubkeyToAddress(suite.senderPvtKy.PublicKey)
	suite.alternateUserPvtKy, _ = crypto.GenerateKey()
	suite.alternateUserAddress = crypto.PubkeyToAddress(suite.alternateUserPvtKy.PublicKey)

	config.Vip().Set(config.ModelMaintenanceEndPoint, "localhost:2222")

}

type MockServiceModelGRPCImpl struct {
	count int
}

func (m MockServiceModelGRPCImpl) mustEmbedUnimplementedModelServer() {
	//TODO implement me
	panic("implement me")
}

func (m MockServiceModelGRPCImpl) CreateModel(context context.Context, request *CreateModelRequest) (*ModelDetailsResponse, error) {
	zap.L().Info("In Service CreateModel")
	count = count + 1
	zap.L().Debug("Count", zap.Int("value", count))
	return &ModelDetailsResponse{Status: Status_CREATED,
		ModelDetails: &ModelDetails{
			ModelId: fmt.Sprintf("%d", count),
		}}, nil
}

func (m MockServiceModelGRPCImpl) UpdateModelAccess(context context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_IN_PROGRESS,
		ModelDetails: &ModelDetails{
			ModelId: request.UpdateModelDetails.ModelId,
		}}, nil
}

func (m MockServiceModelGRPCImpl) DeleteModel(context context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_DELETED,
		ModelDetails: &ModelDetails{
			ModelId: request.UpdateModelDetails.ModelId,
		}}, nil
}

func (m MockServiceModelGRPCImpl) GetModelStatus(context context.Context, request *ModelDetailsRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_IN_PROGRESS,
		ModelDetails: &ModelDetails{
			ModelId: request.ModelDetails.ModelId,
		}}, nil
}

func (m MockServiceModelGRPCImpl) GetAllModels(context context.Context, request *AccessibleModelsRequest) (*AccessibleModelsResponse, error) {
	//Ideally client should take a list of all models and update the status of each and send back a response
	return &AccessibleModelsResponse{}, nil
}

func (suite *ModelServiceTestSuite) TearDownSuite() {
	suite.server.GracefulStop()
}

func (suite *ModelServiceTestSuite) getSignature(text string, blockNumber int, privateKey *ecdsa.PrivateKey) (signature []byte) {
	message := bytes.Join([][]byte{
		[]byte(text),
		crypto.PubkeyToAddress(privateKey.PublicKey).Bytes(),
		math.U256Bytes(big.NewInt(int64(blockNumber))),
	}, nil)
	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	signature, err := crypto.Sign(hash, suite.senderPvtKy)
	if err != nil {
		panic(fmt.Sprintf("Cannot sign test message: %v", err))
	}
	return signature
}

func (suite *ModelServiceTestSuite) TestModelService_UndefinedTrainingService() {
	//when AI developer has not implemented the training.prot , ensure we get back an error when daemon is called
	response, err := suite.serviceNotImplemented.CreateModel(context.TODO(), &CreateModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), response.Status, Status_ERRORED)

	response2, err := suite.serviceNotImplemented.UpdateModelAccess(context.TODO(), &UpdateModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), response2.Status, Status_ERRORED)

	response3, err := suite.serviceNotImplemented.DeleteModel(context.TODO(), &UpdateModelRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), response3.Status, Status_ERRORED)

	response4, err := suite.serviceNotImplemented.GetModelStatus(context.TODO(), &ModelDetailsRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), response4.Status, Status_ERRORED)

	response5, err := suite.serviceNotImplemented.GetAllModels(context.TODO(), &AccessibleModelsRequest{})
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), len(response5.ListOfModels), 0)

}

func (suite *ModelServiceTestSuite) TestModelService_CreateModel() {
	//No Authorization request
	response, err := suite.service.CreateModel(context.TODO(), nil)
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), response)

	// valid request
	request := &CreateModelRequest{
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Message:       "__CreateModel",
			Signature:     suite.getSignature("__CreateModel", 1200, suite.senderPvtKy),
			CurrentBlock:  1200,
		},
		ModelDetails: &ModelDetails{
			GrpcServiceName:      " ",
			GrpcMethodName:       "/example_service.Calculator/train_add",
			Description:          "Just Testing",
			IsPubliclyAccessible: false,
			ModelName:            "ABCD",
			TrainingDataLink:     " ",
			AddressList:          []string{"A1", "A2", "A3"},
		},
	}
	zap.L().Debug("Sender address", zap.Any("value", suite.senderAddress.String()))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response, err = suite.service.CreateModel(ctx, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), "1", response.ModelDetails.ModelId)
	assert.Equal(suite.T(), response.ModelDetails.AddressList, []string{"A1", "A2", "A3", suite.senderAddress.String()})
	userKey := &ModelUserKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  "/example_service.Calculator/train_add",
		GRPCServiceName: " ",
		UserAddress:     suite.senderAddress.String(),
	}
	//check if we have stored the user's associated model Ids
	data, ok, err := suite.service.(*ModelService).userStorage.Get(userKey)
	assert.Equal(suite.T(), []string{"1"}, data.ModelIds)
	assert.Equal(suite.T(), ok, true)
	assert.Nil(suite.T(), err)

	//check if the model Id stored has all the details
	key := &ModelKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  "/example_service.Calculator/train_add",
		GRPCServiceName: " ",
		ModelId:         "1",
	}
	modelData, ok, err := suite.service.(*ModelService).storage.Get(key)
	assert.Equal(suite.T(), []string{"A1", "A2", "A3", suite.senderAddress.String()}, modelData.AuthorizedAddresses)
	assert.Equal(suite.T(), ok, true)
	assert.Nil(suite.T(), err)

	//send a bad signature
	request.Authorization.Signature = suite.getSignature("Different message", 1200, suite.senderPvtKy)
	response, err = suite.service.CreateModel(ctx, request)
	assert.NotNil(suite.T(), err)

	// valid request
	request2 := &CreateModelRequest{
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Message:       "__CreateModel",
			Signature:     suite.getSignature("__CreateModel", 1200, suite.senderPvtKy),
			CurrentBlock:  1200,
		},
		ModelDetails: &ModelDetails{
			GrpcServiceName:      " ",
			GrpcMethodName:       "/example_service.Calculator/train_add",
			Description:          "Just Testing",
			IsPubliclyAccessible: false,
		},
	}
	zap.L().Debug("Sender address", zap.Any("value", suite.senderAddress.String()))
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response, err = suite.service.CreateModel(ctx, request2)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), response.ModelDetails.AddressList, []string{suite.senderAddress.String()})
	//Create an Other Model Id for the same user !!!
	response, err = suite.service.CreateModel(ctx, request)

	request3 := &AccessibleModelsRequest{
		GrpcServiceName: " ",
		GrpcMethodName:  "/example_service.Calculator/train_add",
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Message:       "__UpdateModelAccess",
			Signature:     suite.getSignature("__UpdateModelAccess", 1200, suite.senderPvtKy),
			CurrentBlock:  1200,
		},
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response2, err := suite.service.GetAllModels(ctx, request3)
	assert.Nil(suite.T(), err)
	fmt.Println(response2)
	assert.Equal(suite.T(), len(response2.ListOfModels) > 1, true)

}

func (suite *ModelServiceTestSuite) TestModelService_GetModelStatus() {
	request := &ModelDetailsRequest{
		ModelDetails: &ModelDetails{
			ModelId:         "1",
			GrpcServiceName: " ",
			GrpcMethodName:  "/example_service.Calculator/train_add",
		},
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Message:       "__GetModelStatus",
			Signature:     suite.getSignature("__GetModelStatus", 1200, suite.senderPvtKy),
			CurrentBlock:  1200,
		},
	}
	zap.L().Debug("Sender address", zap.Any("value", suite.senderAddress.String()))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response, err := suite.service.GetModelStatus(ctx, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), Status_IN_PROGRESS, response.Status)
}

func (suite *ModelServiceTestSuite) TestModelService_UpdateModelAccess() {

	userKey := &ModelUserKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  "/example_service.Calculator/train_add",
		GRPCServiceName: " ",
		UserAddress:     suite.senderAddress.String(),
	}
	//check if we have stored the user's associated model Ids
	err := suite.service.(*ModelService).userStorage.Put(userKey, &ModelUserData{
		ModelIds:        []string{"1", "2"},
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  "/example_service.Calculator/train_add",
		GRPCServiceName: " ",
		UserAddress:     suite.senderAddress.String(),
	})

	modelState, _, _ := suite.service.(*ModelService).userStorage.Get(userKey)

	zap.L().Debug("Model state", zap.Any("value", modelState))
	//	assert.Equal(suite.T(), ok, true)
	assert.Nil(suite.T(), err)
	modelData := &ModelData{
		IsPublic:            false,
		AuthorizedAddresses: []string{suite.senderAddress.String()},
		Status:              Status_IN_PROGRESS,
		CreatedByAddress:    suite.senderAddress.String(),
		ModelId:             "1",
		UpdatedByAddress:    suite.senderAddress.String(),
		OrganizationId:      config.GetString(config.OrganizationId),
		ServiceId:           config.GetString(config.ServiceId),
		GroupId:             suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:      "/example_service.Calculator/train_add",
		GRPCServiceName:     " ",
		Description:         "",
		IsDefault:           false,
		TrainingLink:        "",
	}

	_, err = suite.service.(*ModelService).storage.PutIfAbsent(&ModelKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  "/example_service.Calculator/train_add",
		GRPCServiceName: " ",
		ModelId:         "1",
	}, modelData)
	//	assert.Equal(suite.T(), ok, true)
	assert.Nil(suite.T(), err)

	data, _, _ := suite.service.(*ModelService).storage.Get(&ModelKey{
		OrganizationId:  config.GetString(config.OrganizationId),
		ServiceId:       config.GetString(config.ServiceId),
		GroupId:         suite.service.(*ModelService).organizationMetaData.GetGroupIdString(),
		GRPCMethodName:  "/example_service.Calculator/train_add",
		GRPCServiceName: " ",
		ModelId:         "1",
	})

	zap.L().Debug("Model data", zap.Any("value", data))

	request := &UpdateModelRequest{
		UpdateModelDetails: &ModelDetails{
			ModelId:              "1",
			GrpcServiceName:      " ",
			GrpcMethodName:       "/example_service.Calculator/train_add",
			IsPubliclyAccessible: false,
			ModelName:            "ABCD",
			Description:          "How are you",
		},
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Message:       "__UpdateModelAccess",
			Signature:     suite.getSignature("__UpdateModelAccess", 1200, suite.senderPvtKy),
			CurrentBlock:  1200,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response, err := suite.service.UpdateModelAccess(ctx, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), len(response.GetModelDetails().AddressList), 1)

	//update a model id which is not there
	request.UpdateModelDetails.ModelId = "25"
	response, err = suite.service.UpdateModelAccess(ctx, request)
	assert.NotNil(suite.T(), err)

	//update request with someone who does not have access
	request.Authorization.Signature = suite.getSignature("__UpdateModelAccess", 1200, suite.alternateUserPvtKy)
	response, err = suite.service.UpdateModelAccess(ctx, request)
	assert.NotNil(suite.T(), err)

	//update request with someone who does not have access
	request.Authorization = nil
	response, err = suite.service.UpdateModelAccess(ctx, request)
	assert.NotNil(suite.T(), err)

}

func (suite *ModelServiceTestSuite) TestModelService_GetAllAccessibleModels() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	request2 := &AccessibleModelsRequest{
		GrpcServiceName: " ",
		GrpcMethodName:  "/example_service.Calculator/train_add",
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Message:       "__UpdateModelAccess",
			Signature:     suite.getSignature("__UpdateModelAccess", 1200, suite.alternateUserPvtKy),
			CurrentBlock:  1200,
		},
	}
	response, err := suite.service.GetAllModels(ctx, request2)
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), response)
	request2.Authorization.Signature = suite.getSignature("__UpdateModelAccess", 1200, suite.alternateUserPvtKy)
	response, err = suite.service.GetAllModels(ctx, request2)
	assert.NotNil(suite.T(), err)

	request2.Authorization = nil
	response, err = suite.service.GetAllModels(ctx, request2)
	assert.NotNil(suite.T(), err)

}

func (suite *ModelServiceTestSuite) TestModelService_remove() {
	sample1 := []string{"a", "b", "c"}
	sample2 := []string{"b", "c"}
	output := remove(sample1, "a")
	assert.Equal(suite.T(), output, sample2)
	output = remove(output, "a")
	assert.Equal(suite.T(), output, sample2)
}

func (suite *ModelServiceTestSuite) TestModelService_difference() {
	sample1 := []string{"a", "b", "c"}
	sample2 := []string{"b", "c", "e", "f"}
	output := difference(sample1, sample2)
	expected := []string{"a", "e", "f"}
	assert.Equal(suite.T(), expected, output)
}

func (suite *ModelServiceTestSuite) TestModelService_isValuePresent() {
	sample1 := []string{"a", "b", "c"}
	assert.Equal(suite.T(), isValuePresent("a", sample1), true)
	assert.Equal(suite.T(), isValuePresent("d", sample1), false)
}

func (suite *ModelServiceTestSuite) TestModelService_UDeleteModel() {
	response, err := suite.service.DeleteModel(context.Background(), nil)
	assert.NotNil(suite.T(), err)
	//unauthorized signer
	request := &UpdateModelRequest{
		UpdateModelDetails: &ModelDetails{
			ModelId:         "1",
			GrpcServiceName: " ",
			GrpcMethodName:  "/example_service.Calculator/train_add",
		},
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.alternateUserAddress.String(),
			Message:       "__GetModelStatus",
			Signature:     suite.getSignature("__GetModelStatus", 1200, suite.alternateUserPvtKy),
			CurrentBlock:  1200,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response, err = suite.service.DeleteModel(ctx, request)
	assert.NotNil(suite.T(), err)

	zap.L().Debug("Sender address", zap.Any("value", suite.senderAddress.String()))
	//valid signer
	request.Authorization.SignerAddress = suite.senderAddress.String()
	request.Authorization.Signature = suite.getSignature("__GetModelStatus", 1200, suite.senderPvtKy)
	response, err = suite.service.DeleteModel(ctx, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), Status_DELETED, response.Status)

	//bad signer
	request.Authorization.Message = "blah"
	response, err = suite.service.DeleteModel(ctx, request)
	assert.NotNil(suite.T(), err)

	request.Authorization = nil
	response, err = suite.service.DeleteModel(ctx, request)
	assert.NotNil(suite.T(), err)

}
