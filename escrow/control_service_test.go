package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/assert"

	"github.com/singnet/snet-daemon/v6/blockchain"
)

type ControlServiceTestSuite struct {
	suite.Suite
	service         *ProviderControlService
	senderAddress   common.Address
	receiverAddress common.Address
	receiverPvtKy   *ecdsa.PrivateKey
	paymentStorage  *PaymentStorage
	mpeAddress      common.Address
	channelService  PaymentChannelService
	serviceMetaData *blockchain.ServiceMetadata
	orgMetaData     *blockchain.OrganizationMetaData
	storage         *PaymentChannelStorage
}

func (suite *ControlServiceTestSuite) payment() *Payment {
	payment := &Payment{
		Amount:             big.NewInt(12300),
		ChannelID:          big.NewInt(1),
		ChannelNonce:       big.NewInt(0),
		MpeContractAddress: suite.serviceMetaData.GetMpeAddress(),
	}
	SignTestPayment(payment, suite.receiverPvtKy)
	return payment
}

func (suite *ControlServiceTestSuite) putChannel(channelId *big.Int) {
	suite.storage.Put(&PaymentChannelKey{ID: channelId}, &PaymentChannelData{
		ChannelID:        channelId,
		Nonce:            big.NewInt(0),
		Sender:           suite.senderAddress,
		Recipient:        suite.receiverAddress,
		GroupID:          suite.orgMetaData.GetGroupId(),
		FullAmount:       big.NewInt(12345),
		Expiration:       big.NewInt(100),
		Signer:           suite.receiverAddress,
		AuthorizedAmount: big.NewInt(10),
		Signature:        utils.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	})
}

func (suite *ControlServiceTestSuite) mpeChannel() *blockchain.MultiPartyEscrowChannel {
	return &blockchain.MultiPartyEscrowChannel{
		Sender:     suite.senderAddress,
		Recipient:  suite.receiverAddress,
		GroupId:    suite.orgMetaData.GetGroupId(),
		Value:      big.NewInt(12345),
		Nonce:      big.NewInt(0),
		Expiration: big.NewInt(1000),
		Signer:     suite.receiverAddress,
	}
}

func (suite *ControlServiceTestSuite) SetupSuite() {
	//
	var errs error
	suite.receiverPvtKy = GenerateTestPrivateKey()
	println(errs)
	suite.receiverAddress = crypto.PubkeyToAddress(suite.receiverPvtKy.PublicKey)
	orgJson := strings.Replace(testJsonOrgGroupData, "0x671276c61943A35D5F230d076bDFd91B0c47bF09", suite.receiverAddress.Hex(), -1)
	suite.orgMetaData, _ = blockchain.InitOrganizationMetaDataFromJson([]byte(orgJson))
	suite.serviceMetaData, _ = blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
	b := blockchain.NewMockProcessor(true)
	println("suite.orgMetaData.GetPaymentAddress().Hex() " + suite.orgMetaData.GetPaymentAddress().Hex())
	println("suite.receiverAddress.Hex()" + suite.receiverAddress.Hex())

	memoryStorage := storage.NewMemStorage()
	suite.storage = NewPaymentChannelStorage(memoryStorage)
	suite.paymentStorage = NewPaymentStorage(memoryStorage)
	suite.channelService = NewPaymentChannelService(
		suite.storage,
		suite.paymentStorage,
		&BlockchainChannelReader{
			readChannelFromBlockchain: func(channelID *big.Int) (*blockchain.MultiPartyEscrowChannel, bool, error) {
				return suite.mpeChannel(), true, nil
			},
			recipientPaymentAddress: func() common.Address {
				return suite.receiverAddress
			},
		},
		NewEtcdLocker(memoryStorage),
		&ChannelPaymentValidator{
			currentBlock:               func() (*big.Int, error) { return big.NewInt(99), nil },
			paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) },
		}, func() [32]byte {
			return [32]byte{123}
		})

	suite.service = NewProviderControlService(b, suite.channelService, suite.serviceMetaData, suite.orgMetaData)
}

func TestControlServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ControlServiceTestSuite))
}

func (suite *ControlServiceTestSuite) SignStartClaimForMultipleChannels(request *StartMultipleClaimRequest) {
	message := bytes.Join([][]byte{
		[]byte("__StartClaimForMultipleChannels_"),
		suite.serviceMetaData.GetMpeAddress().Bytes(),
		bigIntToBytes(big.NewInt(1)),
		bigIntToBytes(big.NewInt(2)),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	request.Signature = getSignature(message, suite.receiverPvtKy)
}

func (suite *ControlServiceTestSuite) SignListInProgress(request *GetPaymentsListRequest) {
	message := bytes.Join([][]byte{
		[]byte("__list_in_progress"),
		suite.serviceMetaData.GetMpeAddress().Bytes(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	request.Signature = getSignature(message, suite.receiverPvtKy)
}

// Build Request
// Validate Signature
// Add 2 elements in ChannelData
// After claims, check if 2 payments have come in
func (suite *ControlServiceTestSuite) TestStartClaimForMultipleChannels() {
	suite.putChannel(big.NewInt(1))
	suite.putChannel(big.NewInt(2))
	reply, err := suite.service.listChannels()
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), len(reply.Payments), 2)
	ids := make([]uint64, 0)
	//
	ids = append(ids, 2)
	ids = append(ids, 1)
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	startMultipleClaimRequest := &StartMultipleClaimRequest{
		MpeAddress: suite.serviceMetaData.GetMpeAddress().Hex(), ChannelIds: ids, CurrentBlock: blockchain.MockedCurrentBlock,
		Signature: nil}
	suite.SignStartClaimForMultipleChannels(startMultipleClaimRequest)
	replyMultipleClaims, err := suite.service.StartClaimForMultipleChannels(nil, startMultipleClaimRequest)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), bytesToBigInt(replyMultipleClaims.Payments[0].ChannelId).Int64() > 0)
	assert.True(suite.T(), bytesToBigInt(replyMultipleClaims.Payments[1].ChannelId).Int64() > 0)
	paymentsListRequest := &GetPaymentsListRequest{MpeAddress: suite.serviceMetaData.MpeAddress, CurrentBlock: blockchain.MockedCurrentBlock}
	suite.SignListInProgress(paymentsListRequest)
	replyListInProgress, err := suite.service.GetListInProgress(nil, paymentsListRequest)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), replyListInProgress.Payments[0].Signature)
	assert.NotNil(suite.T(), replyListInProgress.Payments[0].Signature)
}

func (suite *ControlServiceTestSuite) TestProviderControlService_checkMpeAddress() {
	servicemetadata := blockchain.ServiceMetadata{}
	servicemetadata.MpeAddress = "0xE8D09a6C296aCdd4c01b21f407ac93fdfC63E78C"
	control_service := NewProviderControlService(nil, nil, &servicemetadata, nil)
	err := control_service.checkMpeAddress("0xe8D09a6C296aCdd4c01b21f407ac93fdfC63E78C")
	assert.Nil(suite.T(), err)
	err = control_service.checkMpeAddress("0xe9D09a6C296aCdd4c01b21f407ac93fdfC63E78C")
	assert.Equal(suite.T(), err.Error(), "the mpeAddress: 0xe9D09a6C296aCdd4c01b21f407ac93fdfC63E78C passed does not match to what has been registered")
}

func (suite *ControlServiceTestSuite) TestBeginClaimOnChannel() {
	control_service := NewProviderControlService(nil, &paymentChannelServiceMock{}, &blockchain.ServiceMetadata{MpeAddress: "0xe9D09a6C296aCdd4c01b21f407ac93fdfC63E78C"}, nil)
	_, err := control_service.beginClaimOnChannel(big.NewInt(12345))
	assert.Equal(suite.T(), err.Error(), "channel Id 12345 was not found on blockchain or storage")
}

func (suite *ControlServiceTestSuite) TestVerifyInvalidSignature() {
	unclaimedRequests := &GetPaymentsListRequest{MpeAddress: suite.serviceMetaData.MpeAddress, CurrentBlock: blockchain.MockedCurrentBlock}
	suite.SignListInProgress(unclaimedRequests)
	reply, err := suite.service.GetListUnclaimed(nil, unclaimedRequests)
	assert.Nil(suite.T(), reply)
	assert.Contains(suite.T(), err.Error(), "does not match to what has been expected / registered")
}
