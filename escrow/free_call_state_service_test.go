package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/config"
	"github.com/singnet/snet-daemon/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/big"
	"testing"
)

type FreeCallStateServiceSuite struct {
	suite.Suite
	signerPrivateKey         *ecdsa.PrivateKey
	signerAddress            common.Address
	mpeContractAddress       common.Address
	freeCallUserPrivateKey   *ecdsa.PrivateKey
	freeCallUserAddress      common.Address
	freeCallPaymentValidator *FreeCallPaymentValidator
	serviceMetaData          *blockchain.ServiceMetadata
	orgMetaData              *blockchain.OrganizationMetaData
	stateService             *FreeCallStateService
	memoryStorage            *storage.MemoryStorage
	storage                  *FreeCallUserStorage
	service                  FreeCallUserService
	request                  *FreeCallStateRequest
}

func TestFreeCallStateServiceTestSuite(t *testing.T) {
	suite.Run(t, new(FreeCallStateServiceSuite))
}

func (suite *FreeCallStateServiceSuite) SetupSuite() {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	suite.signerPrivateKey, _ = crypto.HexToECDSA("063C00D18E147F4F734846E47FE6598FC7A6D56307862F7EDC92B9F43CC27EDD")
	suite.freeCallUserPrivateKey = GenerateTestPrivateKey()
	suite.signerAddress = crypto.PubkeyToAddress(suite.signerPrivateKey.PublicKey)
	suite.freeCallUserAddress = crypto.PubkeyToAddress(suite.freeCallUserPrivateKey.PublicKey)
	suite.freeCallPaymentValidator = &FreeCallPaymentValidator{freeCallSigner: suite.signerAddress,
		currentBlock: func() (*big.Int, error) { return big.NewInt(8308168), nil }}
	testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"ExampleOrganizationId\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	testJsonData = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
		"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 12,  \"free_call_signer_address\": \"0x94d04332C4f5273feF69c4a52D24f42a3aF1F207\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

	suite.orgMetaData, _ = blockchain.InitOrganizationMetaDataFromJson(testJsonOrgGroupData)
	suite.serviceMetaData, _ = blockchain.InitServiceMetaDataFromJson(testJsonData)
	suite.memoryStorage = storage.NewMemStorage()
	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
	suite.service = NewFreeCallUserService(suite.storage,
		NewEtcdLocker(suite.memoryStorage), func() ([32]byte, error) { return suite.orgMetaData.GetGroupId(), nil },
		suite.serviceMetaData)
	suite.stateService = NewFreeCallStateService(suite.orgMetaData, suite.serviceMetaData, suite.service, suite.freeCallPaymentValidator)

}

func (suite *FreeCallStateServiceSuite) TestGetFreeCallsAvailable() {
	suite.request = &FreeCallStateRequest{
		UserId:               "ar@gmail.test.com",
		CurrentBlock:         8308168,
		TokenExpiryDateBlock: 99919408168,
	}
	SetAuthToken(suite)
	SetSignature(suite)
	reply, err := suite.stateService.GetFreeCallsAvailable(nil, suite.request)
	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.Equal(suite.T(), reply.FreeCallsAvailable, uint64(12))

}

func (suite *FreeCallStateServiceSuite) TestErrors() {
	suite.request = &FreeCallStateRequest{
		UserId:               "ar@invalid.test.com",
		CurrentBlock:         8308168,
		TokenExpiryDateBlock: 9408168,
	}
	SetAuthToken(suite)
	SetSignature(suite)
	//now change the userId so that signature becomes invalid
	suite.request.UserId = "invalid@test.com"
	reply, err := suite.stateService.GetFreeCallsAvailable(nil, suite.request)
	assert.Contains(suite.T(), err.Error(), "payment signer is not valid")
	assert.Nil(suite.T(), reply)
}

func SetAuthToken(suite *FreeCallStateServiceSuite) {
	message := bytes.Join([][]byte{
		[]byte(suite.request.UserId),
		suite.freeCallUserAddress.Bytes(),
		bigIntToBytes(big.NewInt(int64(suite.request.GetTokenExpiryDateBlock()))),
	}, nil)
	suite.request.TokenForFreeCall = getSignature(message, suite.signerPrivateKey)

}

func SetSignature(suite *FreeCallStateServiceSuite) {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(suite.request.UserId),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(suite.orgMetaData.GetGroupIdString()),
		bigIntToBytes(big.NewInt(int64(suite.request.GetCurrentBlock()))),
		suite.request.TokenForFreeCall,
	}, nil)

	suite.request.Signature = getSignature(message, suite.freeCallUserPrivateKey)

}
