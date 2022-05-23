package training

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/storage"
	"github.com/soheilhy/cmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type ModelServiceTestSuite struct {
	suite.Suite
	serviceURL    string
	server        *grpc.Server
	mockService   MockServiceModelGRPCImpl
	service       ModelServer
	senderPvtKy   *ecdsa.PrivateKey
	senderAddress common.Address

	alternateUserPvtKy   *ecdsa.PrivateKey
	alternateUserAddress common.Address
}

func TestModelServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ModelServiceTestSuite))
}
func (suite *ModelServiceTestSuite) GetGRPCServerAndServe() (server *grpc.Server) {
	server = grpc.NewServer()
	ch := make(chan int)
	go func() {
		lis, err := net.Listen("tcp", ":2222")
		if err != nil {
			panic(err)
		}
		mux := cmux.New(lis)
		grpcWebServer := grpcweb.WrapServer(server, grpcweb.WithCorsForRegisteredEndpointsOnly(false))
		httpHandler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsAcceptableGrpcCorsRequest(req) {
				grpcWebServer.ServeHTTP(resp, req)
			} else {
				if strings.Split(req.URL.Path, "/")[1] == "register" {
					resp.Header().Set("Access-Control-Allow-Origin", "*")
					fmt.Fprintln(resp, "Registering service...... ")
				} else if strings.Split(req.URL.Path, "/")[1] == "heartbeat" {
					resp.Header().Set("Access-Control-Allow-Origin", "*")
					fmt.Fprint(resp, "{\"serviceID\":\"SERVICE001\",\"status\":\"SERVING\"}")
				} else {
					http.NotFound(resp, req)

				}
			}
		})
		suite.server = server
		RegisterModelServer(server, suite.mockService)
		httpL := mux.Match(cmux.HTTP1Fast())
		grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
		go server.Serve(grpcL)
		go http.Serve(httpL, httpHandler)
		go mux.Serve()
		ch <- 0
	}()

	_ = <-ch
	return
}
func (suite *ModelServiceTestSuite) SetupSuite() {
	config.Vip().Set(config.ModelTrainingEndpoint, "http://localhost:2222")
	suite.mockService = MockServiceModelGRPCImpl{}
	suite.serviceURL = config.GetString(config.ModelTrainingEndpoint)
	suite.server = suite.GetGRPCServerAndServe()

	testJsonOrgGroupData := "{   \"org_name\": \"organization_name\",   \"org_id\": \"ExampleOrganizationId\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	testJsonData := "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
		"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x94d04332C4f5273feF69c4a52D24f42a3aF1F207\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

	orgMetaData, _ := blockchain.InitOrganizationMetaDataFromJson(testJsonOrgGroupData)
	serviceMetaData, _ := blockchain.InitServiceMetaDataFromJson(testJsonData)
	suite.service = NewModelService(nil, serviceMetaData, orgMetaData,
		NewUserModelStorage(storage.NewMemStorage()))
	suite.senderPvtKy, _ = crypto.GenerateKey()
	suite.senderAddress = crypto.PubkeyToAddress(suite.senderPvtKy.PublicKey)
	suite.alternateUserPvtKy, _ = crypto.GenerateKey()
	suite.alternateUserAddress = crypto.PubkeyToAddress(suite.alternateUserPvtKy.PublicKey)

}

type MockServiceModelGRPCImpl struct {
}

func (m MockServiceModelGRPCImpl) CreateModel(context context.Context, request *CreateModelRequest) (*ModelDetailsResponse, error) {
	print("In Service CreateModel")
	return &ModelDetailsResponse{Status: Status_CREATED,
		ModelDetails: &ModelDetails{
			ModelId: "1",
		}}, nil
}

func (m MockServiceModelGRPCImpl) UpdateModelAccess(context context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_IN_PROGRESS,
		ModelDetails: &ModelDetails{
			ModelId: request.ModelDetailsRequest.ModelDetails.ModelId,
		}}, nil
}

func (m MockServiceModelGRPCImpl) DeleteModel(context context.Context, request *UpdateModelRequest) (*ModelDetailsResponse, error) {
	return &ModelDetailsResponse{Status: Status_DELETED,
		ModelDetails: &ModelDetails{
			ModelId: request.ModelDetailsRequest.ModelDetails.ModelId,
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
func getSignature(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte) {

	return signature
}
func (suite *ModelServiceTestSuite) getSignature(text string, blockNumber int, privateKey *ecdsa.PrivateKey) (signature []byte) {
	message := bytes.Join([][]byte{
		[]byte(text),
		crypto.PubkeyToAddress(privateKey.PublicKey).Bytes(),
		abi.U256(big.NewInt(int64(blockNumber))),
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

func (suite *ModelServiceTestSuite) TestModelService_CreateModel() {
	response, err := suite.service.CreateModel(context.TODO(), nil)
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), response)

	request := &CreateModelRequest{
		Authorization: &AuthorizationDetails{
			SignerAddress: suite.senderAddress.String(),
			Signature:     suite.getSignature("__CreateModel", 1200, suite.senderPvtKy),
			CurrentBlock:  1200,
		},
		MethodName:           "TESTMETHOD",
		ModelDescription:     "Just Testing",
		IsPubliclyAccessible: false,
	}
	fmt.Println(suite.senderAddress.String())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()
	response, err = suite.service.CreateModel(ctx, request)
	assert.Nil(suite.T(), err)

}

/*
func (suite *ModelServiceTestSuite) TestModelService_DeleteModel(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_GetAllModels(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_GetModelStatus(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_UpdateModelAccess(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_createModelData(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_deleteModelDetails(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getMessageBytes(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getModelDataForStatusUpdate(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getModelDataForUpdate(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getModelDetails(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getModelKeyToCreate(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getModelKeyToUpdate(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_getServiceClient(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_storeModelDetails(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_updateModelDetails(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_updateModelDetailsForStatus(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_verifySignatureForGetAllModels(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_verifySignerForCreateModel(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_verifySignerForDeleteModel(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_verifySignerForGetModelStatus(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestModelService_verifySignerForUpdateModel(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNewModelService(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNoModelSupportService_CreateModel(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNoModelSupportService_DeleteModel(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNoModelSupportService_GetAllModels(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNoModelSupportService_GetModelDetails(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNoModelSupportService_GetModelStatus(t *testing.T) {

}

func (suite *ModelServiceTestSuite) TestNoModelSupportService_UpdateModelAccess(t *testing.T) {

}
*/
