package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FreeCallStateServiceSuite struct {
	suite.Suite
	signerPrivateKey         *ecdsa.PrivateKey
	signerAddress            common.Address
	mpeContractAddress       common.Address
	userPrivateKey           *ecdsa.PrivateKey
	userAddress              common.Address
	freeCallPaymentValidator *FreeCallPaymentValidator
	serviceMetaData          *blockchain.ServiceMetadata
	orgMetaData              *blockchain.OrganizationMetaData
	stateService             *FreeCallStateService
	memoryStorage            *storage.MemoryStorage
	storage                  *FreeCallUserStorage
	service                  FreeCallUserService
	useFreeCallRequest       *FreeCallStateRequest
	newTokenRequest          *GetFreeCallTokenRequest
}

type MockedERC20 struct {
	//
}

func (m MockedERC20) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	return big.NewInt(0).SetUint64(1_000_000_000_000_000_000), nil
}

func TestFreeCallStateServiceTestSuite(t *testing.T) {
	suite.Run(t, new(FreeCallStateServiceSuite))
}

func (suite *FreeCallStateServiceSuite) SetupSuite() {
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Vip().Set(config.ServiceEndpointKey, "http://localhost:5000")
	config.Vip().Set(config.PvtKeyForFreeCalls, "aeaa9fb59c0dd868260af55ea65be077dbcaa063c067dfc0865845a0af5de84c")

	err := config.Validate()
	assert.Nil(suite.T(), err)

	suite.signerPrivateKey, err = crypto.HexToECDSA("aeaa9fb59c0dd868260af55ea65be077dbcaa063c067dfc0865845a0af5de84c")
	assert.Nil(suite.T(), err)
	suite.signerAddress = crypto.PubkeyToAddress(suite.signerPrivateKey.PublicKey)
	suite.T().Logf("signer addr: %v", suite.signerAddress)

	suite.userPrivateKey = GenerateTestPrivateKey()
	suite.userAddress = crypto.PubkeyToAddress(suite.userPrivateKey.PublicKey)
	suite.T().Logf("user addr: %v", suite.userAddress)

	suite.freeCallPaymentValidator = &FreeCallPaymentValidator{freeCallSignerAddress: suite.signerAddress, freeCallSigner: suite.signerPrivateKey,
		currentBlock: func() (*big.Int, error) { return big.NewInt(99), nil }}
	testJsonOrgGroupData = "{   \"org_name\": \"organization_name\",   \"org_id\": \"ExampleOrganizationId\",   \"groups\": [     {       \"group_name\": \"default_group2\",       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     },      {       \"group_name\": \"default_group\",       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"payment\": {         \"payment_address\": \"0x671276c61943A35D5F230d076bDFd91B0c47bF09\",         \"payment_expiration_threshold\": 40320,         \"payment_channel_storage_type\": \"etcd\",         \"payment_channel_storage_client\": {           \"connection_timeout\": \"15s\",           \"request_timeout\": \"13s\",           \"endpoints\": [             \"http://127.0.0.1:2379\"           ]         }       }     }   ] }"
	testJsonData = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\", " +
		"  \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {    \"free_calls\": 10,  \"free_call_signer_address\": \"0xF627CE8635cdC34b2f619FDDb4E4b61308D6BD68\",  \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",           \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"default\":true,         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],       \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

	suite.orgMetaData, _ = blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	suite.serviceMetaData, _ = blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
	suite.memoryStorage = storage.NewMemStorage()
	suite.storage = NewFreeCallUserStorage(suite.memoryStorage)
	suite.service = NewFreeCallUserService(suite.storage,
		NewEtcdLocker(suite.memoryStorage), func() ([32]byte, error) { return suite.orgMetaData.GetGroupId(), nil },
		suite.serviceMetaData)
	erc20 := MockedERC20{}
	suite.stateService = NewFreeCallStateService(suite.orgMetaData, suite.serviceMetaData, suite.service, suite.freeCallPaymentValidator, erc20, big.NewInt(1))
}

func (suite *FreeCallStateServiceSuite) TestGetFreeCallsAvailable() {

	suite.newTokenRequest = &GetFreeCallTokenRequest{
		Address:      suite.userAddress.Hex(),
		CurrentBlock: 99,
	}
	suite.SetSignatureForNewToken()

	freeCallTokenResp, err := suite.stateService.GetFreeCallToken(nil, suite.newTokenRequest)
	assert.Nil(suite.T(), err)

	suite.useFreeCallRequest = &FreeCallStateRequest{
		Address:       suite.userAddress.Hex(),
		CurrentBlock:  99,
		FreeCallToken: freeCallTokenResp.Token,
	}

	suite.SetSignatureWithToken()
	reply, err := suite.stateService.GetFreeCallsAvailable(nil, suite.useFreeCallRequest)

	assert.Nil(suite.T(), err, "Unexpected error: %v", err)
	assert.Equal(suite.T(), reply.FreeCallsAvailable, uint64(10))
}

func (suite *FreeCallStateServiceSuite) TestErrors() {
	suite.useFreeCallRequest = &FreeCallStateRequest{
		Address:      suite.signerAddress.Hex(),
		CurrentBlock: 99,
	}

	token, _ := suite.freeCallPaymentValidator.NewFreeCallToken(suite.signerAddress.Hex(), nil, nil)
	suite.useFreeCallRequest.FreeCallToken = getSignature(token, suite.signerPrivateKey)
	suite.SetSignatureWithToken()

	//now change the address so that signature becomes invalid
	suite.useFreeCallRequest.Address = "0xInvalidAddress"
	suite.useFreeCallRequest.FreeCallToken = token
	reply, err := suite.stateService.GetFreeCallsAvailable(nil, suite.useFreeCallRequest)
	assert.NotNil(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "sign is not valid")
	assert.Nil(suite.T(), reply)
}

func (suite *FreeCallStateServiceSuite) SetSignatureWithToken() {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(suite.userAddress.Hex()),
		[]byte(suite.useFreeCallRequest.GetUserId()),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(suite.orgMetaData.GetGroupIdString()),
		bigIntToBytes(big.NewInt(int64(suite.useFreeCallRequest.GetCurrentBlock()))),
		suite.useFreeCallRequest.FreeCallToken,
	}, nil)

	suite.useFreeCallRequest.Signature = getSignature(message, suite.userPrivateKey)
}

func (suite *FreeCallStateServiceSuite) SetSignatureForNewToken() {
	message := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(suite.userAddress.Hex()),
		[]byte(suite.newTokenRequest.GetUserId()),
		[]byte(config.GetString(config.OrganizationId)),
		[]byte(config.GetString(config.ServiceId)),
		[]byte(suite.orgMetaData.GetGroupIdString()),
		bigIntToBytes(big.NewInt(int64(suite.newTokenRequest.GetCurrentBlock()))),
	}, nil)

	suite.newTokenRequest.Signature = getSignature(message, suite.userPrivateKey)
}
