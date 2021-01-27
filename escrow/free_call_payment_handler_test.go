package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/storage"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"org_id1\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
var testJsonData = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
	"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

type FreeCallPaymentHandlerTestSuite struct {
	suite.Suite
	paymentHandler freeCallPaymentHandler
	privateKey     *ecdsa.PrivateKey
	key            *FreeCallUserKey
	data           *FreeCallUserData
	memoryStorage  *storage.MemoryStorage
	storage        *FreeCallUserStorage
	metadata       *blockchain.ServiceMetadata
}

func (suite *FreeCallPaymentHandlerTestSuite) getKey(userId string) *FreeCallUserKey {
	return &FreeCallUserKey{UserId: userId, ServiceId: config.GetString(config.ServiceId),
		OrganizationId: config.GetString(config.OrganizationId), GroupID: "fQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}

}
func (suite *FreeCallPaymentHandlerTestSuite) SetupSuite() {

	suite.privateKey = GenerateTestPrivateKey()
	suite.memoryStorage = storage.NewMemStorage()
	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
	orgMetadata, _ := blockchain.InitOrganizationMetaDataFromJson(testJsonOrgGroupData)
	suite.metadata, _ = blockchain.InitServiceMetaDataFromJson(testJsonData)
	suite.data = &FreeCallUserData{FreeCallsMade: 12, UserId: "user1"}
	suite.key = suite.getKey("user1")
	suite.paymentHandler = freeCallPaymentHandler{
		orgMetadata:     orgMetadata,
		serviceMetadata: suite.metadata,
		freeCallPaymentValidator: NewFreeCallPaymentValidator(func() (*big.Int, error) {
			return big.NewInt(99), nil
		}, crypto.PubkeyToAddress(suite.privateKey.PublicKey)),
		service: NewFreeCallUserService(suite.storage, NewEtcdLocker(suite.memoryStorage), func() ([32]byte, error) { return [32]byte{125}, nil }, suite.metadata),
	}
}

func SignTestFreeCallPayment(privateKey *ecdsa.PrivateKey, currentBlock int64, user string) []byte {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(user),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		bigIntToBytes(big.NewInt(currentBlock)),
	}, nil)

	return getSignature(message, privateKey)
}

func TestFreeCallPaymentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(FreeCallPaymentHandlerTestSuite))
}

func (suite *FreeCallPaymentHandlerTestSuite) grpcMetadataForFreeCall(user string, currentBlock int64) metadata.MD {
	md := metadata.New(map[string]string{})
	md.Set(handler.FreeCallUserIdHeader, user)
	md.Set(handler.CurrentBlockNumberHeader, strconv.FormatInt(currentBlock, 10))
	md.Set(handler.FreeCallAuthTokenExpiryBlockNumberHeader, strconv.FormatInt(currentBlock*100000, 10))
	md.Set(handler.PaymentChannelSignatureHeader, string(SignTestFreeCallPayment(suite.privateKey, currentBlock, user)))
	md.Set(handler.FreeCallAuthTokenHeader, string(SignTestFreeCallPayment(suite.privateKey, currentBlock, user)))
	println("************************************************************")
	println((SignTestFreeCallPayment(suite.privateKey, currentBlock, user)))
	println("************************************************************")
	return md
}

func (suite *FreeCallPaymentHandlerTestSuite) grpcContextForFreeCall(patch func(*metadata.MD), userId string) *handler.GrpcStreamContext {
	md := suite.grpcMetadataForFreeCall(userId, 99)
	patch(&md)
	return &handler.GrpcStreamContext{
		MD: md,
	}
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPaymentFromContext() {
	context := suite.grpcContextForFreeCall(func(md *metadata.MD) {}, "user1")
	suite.storage.Put(suite.key, suite.data)
	transaction, err := suite.paymentHandler.Payment(context)
	assert.Nil(suite.T(), transaction)
	assert.Contains(suite.T(), err.Err().Error(), "free call limit has been exceeded")
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPaymentComplete() {
	suite.storage.Put(suite.getKey("user2"), &FreeCallUserData{FreeCallsMade: 9})
	paymentTransaction, err := suite.paymentHandler.Payment(suite.grpcContextForFreeCall(func(md *metadata.MD) {}, "user2"))
	assert.Nil(suite.T(), err)
	err = suite.paymentHandler.Complete(paymentTransaction)
	assert.Nil(suite.T(), err)
	updatedUser, ok, errB := suite.storage.Get(suite.getKey("user2"))
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), errB)
	assert.Equal(suite.T(), updatedUser.FreeCallsMade, 10)
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPaymentCompleteAfterError() {
	suite.storage.Put(suite.getKey("user3"), &FreeCallUserData{FreeCallsMade: 9})
	paymentTransaction, err := suite.paymentHandler.Payment(suite.grpcContextForFreeCall(func(md *metadata.MD) {}, "user2"))
	assert.Nil(suite.T(), err)
	errA := suite.paymentHandler.CompleteAfterError(paymentTransaction, fmt.Errorf("Test Error"))
	assert.Nil(suite.T(), errA)
	updatedUser, ok, errB := suite.storage.Get(suite.getKey("user3"))
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), errB)
	assert.Equal(suite.T(), updatedUser.FreeCallsMade, 9)
}

func (suite *FreeCallPaymentHandlerTestSuite) Test_freeCallPaymentHandler_Type() {
	assert.Equal(suite.T(), suite.paymentHandler.Type(), FreeCallPaymentType)
}
