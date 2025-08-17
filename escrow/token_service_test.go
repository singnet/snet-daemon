package escrow

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/token"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TokenServiceTestSuite struct {
	suite.Suite
	service         *TokenService
	senderAddress   common.Address
	senderPvtKy     *ecdsa.PrivateKey
	receiverAddress common.Address
	receiverPvtKy   *ecdsa.PrivateKey
	paymentStorage  *PaymentStorage
	mpeAddress      common.Address
	channelService  PaymentChannelService
	serviceMetaData *blockchain.ServiceMetadata
	orgMetaData     *blockchain.OrganizationMetaData
	storage         *PaymentChannelStorage
	channelID       *big.Int
}

func (suite *TokenServiceTestSuite) payment() *Payment {
	payment := &Payment{
		Amount:             big.NewInt(12300),
		ChannelID:          big.NewInt(1),
		ChannelNonce:       big.NewInt(0),
		MpeContractAddress: suite.serviceMetaData.GetMpeAddress(),
	}
	SignTestPayment(payment, suite.receiverPvtKy)
	return payment
}

func (suite *TokenServiceTestSuite) putChannel(channelId *big.Int) {
	suite.storage.Put(&PaymentChannelKey{ID: channelId}, &PaymentChannelData{
		ChannelID:        channelId,
		Nonce:            big.NewInt(0),
		Sender:           suite.senderAddress,
		Recipient:        suite.receiverAddress,
		GroupID:          suite.orgMetaData.GetGroupId(),
		FullAmount:       big.NewInt(20),
		Expiration:       big.NewInt(1000),
		Signer:           suite.receiverAddress,
		AuthorizedAmount: big.NewInt(10),
		Signature:        utils.HexToBytes("0xa4d2ae6f3edd1f7fe77e4f6f78ba18d62e6093bcae01ef86d5de902d33662fa372011287ea2d8d8436d9db8a366f43480678df25453b484c67f80941ef2c05ef01"),
	})
}

func (suite *TokenServiceTestSuite) mpeChannel() *blockchain.MultiPartyEscrowChannel {
	return &blockchain.MultiPartyEscrowChannel{
		Sender:     suite.senderAddress,
		Recipient:  suite.receiverAddress,
		GroupId:    suite.orgMetaData.GetGroupId(),
		Value:      big.NewInt(20),
		Nonce:      big.NewInt(0),
		Expiration: big.NewInt(1000),
		Signer:     suite.receiverAddress,
	}
}

func (suite *TokenServiceTestSuite) SetupSuite() {
	//
	suite.channelID = big.NewInt(1)
	suite.receiverPvtKy = GenerateTestPrivateKey()
	suite.senderPvtKy = GenerateTestPrivateKey()
	suite.senderAddress = crypto.PubkeyToAddress(suite.senderPvtKy.PublicKey)
	suite.receiverAddress = crypto.PubkeyToAddress(suite.receiverPvtKy.PublicKey)
	orgJson := strings.Replace(testJsonOrgGroupData, "0x671276c61943A35D5F230d076bDFd91B0c47bF09", suite.receiverAddress.Hex(), -1)
	suite.orgMetaData, _ = blockchain.InitOrganizationMetaDataFromJson([]byte(orgJson))
	suite.serviceMetaData, _ = blockchain.InitServiceMetaDataFromJson([]byte(testJsonData))
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

	tokenManager := token.NewJWTTokenService(*suite.orgMetaData)
	suite.service = NewTokenService(suite.channelService, NewPrePaidService(NewPrepaidStorage(storage.NewMemStorage()), nil, nil), tokenManager,
		&ChannelPaymentValidator{currentBlock: func() (*big.Int, error) { return big.NewInt(99), nil },
			paymentExpirationThreshold: func() *big.Int { return big.NewInt(0) }}, suite.serviceMetaData)
	suite.putChannel(big.NewInt(1))

}

func (suite *TokenServiceTestSuite) SignRequest(request *TokenRequest, privateKey *ecdsa.PrivateKey) {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		suite.serviceMetaData.GetMpeAddress().Bytes(),
		bigIntToBytes(suite.channelID),
		bigIntToBytes(big.NewInt(0)),
		bigIntToBytes(big.NewInt(0).SetUint64(request.SignedAmount)),
	}, nil)

	request.ClaimSignature = getSignature(message, privateKey)
	message = bytes.Join([][]byte{
		request.ClaimSignature,
		bigIntToBytes(big.NewInt(0).SetUint64(request.CurrentBlock)),
	}, nil)
	request.Signature = getSignature(message, privateKey)

}
func TestTokenServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TokenServiceTestSuite))
}

func (suite *TokenServiceTestSuite) TestGetToken() {
	request := &TokenRequest{
		ChannelId:    1,
		SignedAmount: 11,
		CurrentNonce: 0,
		CurrentBlock: 100,
	}
	suite.SignRequest(request, suite.senderPvtKy)
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), reply.UsedAmount, big.NewInt(0).Uint64())
	assert.Equal(suite.T(), reply.PlannedAmount, big.NewInt(1).Uint64())

	plannedusage, ok, err := suite.service.prePaidUsageService.GetUsage(
		PrePaidDataKey{UsageType: PLANNED_AMOUNT, ChannelID: suite.channelID})
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), plannedusage.Amount, big.NewInt(1))

	channel, ok, err := suite.service.channelService.PaymentChannel(&PaymentChannelKey{ID: suite.channelID})
	assert.Equal(suite.T(), channel.AuthorizedAmount, big.NewInt(11))
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), err)
	// Request a Token for the same last signed amount
	reply, err = suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), reply.UsedAmount, big.NewInt(0).Uint64())
	assert.Equal(suite.T(), reply.PlannedAmount, big.NewInt(1).Uint64())
	request.SignedAmount = 13
	suite.SignRequest(request, suite.senderPvtKy)
	reply, err = suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), reply.UsedAmount, big.NewInt(0).Uint64())
	assert.Equal(suite.T(), reply.PlannedAmount, big.NewInt(3).Uint64())

}

func (suite *TokenServiceTestSuite) TestGetTokenForOverSignedAmount() {
	request := &TokenRequest{
		ChannelId:    1,
		SignedAmount: 3000,
		CurrentNonce: 0,
		CurrentBlock: 100,
	}
	suite.SignRequest(request, suite.senderPvtKy)
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "signed amount for token request cannot be greater than full amount in channel")

}

func (suite *TokenServiceTestSuite) TestGetTokenLesserSignedAmount() {
	request := &TokenRequest{
		ChannelId:    1,
		SignedAmount: 3,
		CurrentNonce: 0,
		CurrentBlock: 100,
	}
	suite.SignRequest(request, suite.senderPvtKy)
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "signed amount for token request needs to be greater than last signed amount")

}

func (suite *TokenServiceTestSuite) TestInvalidChannelID() {
	request := &TokenRequest{
		ChannelId: 99,
	}
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "channel is not found, channelId: 99")

}

func (suite *TokenServiceTestSuite) TestInvalidSignature() {
	request := &TokenRequest{
		ChannelId:    1,
		SignedAmount: 14,
	}
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "incorrect signature")

}

func (suite *TokenServiceTestSuite) TestInvalidSigner() {
	request := &TokenRequest{
		ChannelId:    1,
		CurrentNonce: 0,
		CurrentBlock: 100,
		SignedAmount: 13,
	}
	suite.SignRequest(request, GenerateTestPrivateKey())
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "only channel signer/sender/receiver can get a Valid Token")

}

func (suite *TokenServiceTestSuite) TestExpiredSignature() {
	request := &TokenRequest{
		ChannelId:    1,
		CurrentNonce: 0,
		CurrentBlock: 1000,
		SignedAmount: 13,
	}
	suite.SignRequest(request, suite.senderPvtKy)
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "authentication failed as the signature passed has expired")

}

func (suite *TokenServiceTestSuite) TestIncorrectChannelNonce() {
	request := &TokenRequest{
		ChannelId:    1,
		CurrentNonce: 1,
		CurrentBlock: 100,
		SignedAmount: 13,
	}
	suite.SignRequest(request, suite.senderPvtKy)
	reply, err := suite.service.GetToken(nil, request)
	assert.Nil(suite.T(), reply)
	assert.Equal(suite.T(), err.Error(), "incorrect payment channel nonce, latest: 0, sent: 1")

}
