package escrow

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

var testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"YOUR_ORG_ID\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
var testJsonData = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
	"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 10,  \"free_call_signer_address\": \"0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"YOUR_SERVICE_ID\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

type FreeCallPaymentHandlerTestSuite struct {
	suite.Suite
	paymentHandler  freeCallPaymentHandler
	ownerPrivateKey *ecdsa.PrivateKey
	userAddr        common.Address
	key             *FreeCallUserKey
	data            *FreeCallUserData
	memoryStorage   *storage.MemoryStorage
	storage         *FreeCallUserStorage
	metadata        *blockchain.ServiceMetadata
	orgMetadata     *blockchain.OrganizationMetaData
	serviceID       string
}

func (suite *FreeCallPaymentHandlerTestSuite) getKey() *FreeCallUserKey {
	return &FreeCallUserKey{UserId: "", Address: suite.userAddr.Hex(), ServiceId: config.GetString(config.ServiceId),
		OrganizationId: config.GetString(config.OrganizationId), GroupID: suite.orgMetadata.GetGroupIdString()}
}
func (suite *FreeCallPaymentHandlerTestSuite) getKeyAndData(freeCallsMade int) (*FreeCallUserKey, *FreeCallUserData) {
	key := suite.getKey()
	return key, &FreeCallUserData{FreeCallsMade: freeCallsMade, UserID: key.UserId, Address: key.Address, ServiceId: key.ServiceId, GroupID: key.GroupID, OrganizationId: key.OrganizationId}
}

func (suite *FreeCallPaymentHandlerTestSuite) SetupSuite() {

	var err error
	suite.ownerPrivateKey = utils.ParsePrivateKey("e910576986ad6541bad229755afa750701a00b10f9b752ad228cac4873c7d421") // 0x101a018fe784bf01d538b5d4dfa311d047dba491
	assert.Nil(suite.T(), err)
	suite.memoryStorage = storage.NewMemStorage()
	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
	suite.orgMetadata, err = blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	assert.Nil(suite.T(), err)
	suite.serviceID = "YOUR_SERVICE_ID"
	config.Vip().Set(config.OrganizationId, suite.orgMetadata.OrgID)
	config.Vip().Set(config.DaemonGroupName, "default_group")

	suite.metadata, err = blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
	assert.Nil(suite.T(), err)

	suite.userAddr = crypto.PubkeyToAddress(suite.ownerPrivateKey.PublicKey)

	//suite.data = &FreeCallUserData{FreeCallsMade: 10, UserAddress: "0x101a018fe784bf01d538b5d4dfa311d047dba491"}
	//suite.key = suite.getKey("0x101a018fe784bf01d538b5d4dfa311d047dba491")
	suite.paymentHandler = freeCallPaymentHandler{
		orgMetadata:     suite.orgMetadata,
		serviceMetadata: suite.metadata,
		freeCallPaymentValidator: NewFreeCallPaymentValidator(func() (*big.Int, error) {
			return big.NewInt(99), nil
		}, crypto.PubkeyToAddress(suite.ownerPrivateKey.PublicKey), suite.ownerPrivateKey, []common.Address{}),
		service: NewFreeCallUserService(suite.storage, NewEtcdLocker(suite.memoryStorage), func() ([32]byte, error) { return suite.orgMetadata.GetGroupId(), nil }, suite.metadata),
	}
}

func TestFreeCallPaymentHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(FreeCallPaymentHandlerTestSuite))
}

func (suite *FreeCallPaymentHandlerTestSuite) grpcMetadataForFreeCall(user *ecdsa.PrivateKey) metadata.MD {

	token, expBlock := suite.paymentHandler.freeCallPaymentValidator.NewFreeCallToken(crypto.PubkeyToAddress(user.PublicKey).Hex(), nil, nil)
	p := FreeCallPayment{
		Address:                    crypto.PubkeyToAddress(user.PublicKey).Hex(),
		ServiceId:                  suite.serviceID,
		OrganizationId:             suite.orgMetadata.OrgID,
		CurrentBlockNumber:         big.NewInt(99),
		GroupId:                    suite.orgMetadata.GetGroupIdString(),
		AuthToken:                  token,
		AuthTokenExpiryBlockNumber: expBlock,
	}

	SignFreeTestPayment(&p, user)

	md := metadata.New(map[string]string{})
	md.Set(handler.CurrentBlockNumberHeader, strconv.FormatInt(99, 10))
	md.Set(handler.FreeCallUserAddressHeader, crypto.PubkeyToAddress(user.PublicKey).Hex())
	md.Set(handler.PaymentChannelSignatureHeader, string(p.Signature))
	md.Set(handler.FreeCallAuthTokenHeader, string(token))
	md.Set(handler.PaymentTypeHeader, FreeCallPaymentType)
	return md
}

func (suite *FreeCallPaymentHandlerTestSuite) grpcContextForFreeCall(userPrivateKey *ecdsa.PrivateKey) *handler.GrpcStreamContext {
	md := suite.grpcMetadataForFreeCall(userPrivateKey)
	return &handler.GrpcStreamContext{
		MD: md,
	}
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPaymentFromContext() {
	err := suite.storage.Put(suite.getKey(), &FreeCallUserData{FreeCallsMade: 100})
	assert.Nil(suite.T(), err)

	context := suite.grpcContextForFreeCall(suite.ownerPrivateKey)
	transaction, err := suite.paymentHandler.Payment(context)
	assert.NotNil(suite.T(), err)
	assert.Nil(suite.T(), transaction)
	assert.Contains(suite.T(), err.Error(), "free call limit has been exceeded")
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPaymentComplete() {
	key, data := suite.getKeyAndData(9)
	err := suite.storage.Put(key, data)
	assert.Nil(suite.T(), err)

	user, ok, errA := suite.storage.Get(key)
	assert.Nil(suite.T(), errA)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), 9, user.FreeCallsMade)

	paymentTransaction, err := suite.paymentHandler.Payment(suite.grpcContextForFreeCall(suite.ownerPrivateKey))
	assert.Nil(suite.T(), err)
	err = suite.paymentHandler.Complete(paymentTransaction)
	assert.Nil(suite.T(), err)
	updatedUser, ok, errB := suite.storage.Get(key)
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), errB)
	assert.True(suite.T(), strings.EqualFold("0x101a018fe784bf01d538b5d4dfa311d047dba491", updatedUser.Address))
	assert.Equal(suite.T(), 10, updatedUser.FreeCallsMade)
}

func (suite *FreeCallPaymentHandlerTestSuite) TestFreeCallGetPaymentCompleteAfterError() {
	err := suite.storage.Put(suite.getKey(), &FreeCallUserData{FreeCallsMade: 9})
	assert.Nil(suite.T(), err)
	paymentTransaction, err := suite.paymentHandler.Payment(suite.grpcContextForFreeCall(suite.ownerPrivateKey))
	assert.Nil(suite.T(), err)
	errA := suite.paymentHandler.CompleteAfterError(paymentTransaction, fmt.Errorf("test error"))
	assert.Nil(suite.T(), errA)
	updatedUser, ok, errB := suite.storage.Get(suite.getKey())
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), errB)
	assert.Equal(suite.T(), updatedUser.FreeCallsMade, 9)
}

func (suite *FreeCallPaymentHandlerTestSuite) Test_freeCallPaymentHandler_Type() {
	assert.Equal(suite.T(), suite.paymentHandler.Type(), FreeCallPaymentType)
}
