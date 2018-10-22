package escrow

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

type storageTestType struct {
	ethereum            blockchain.SimulatedEthereumEnvironment
	defaultChannel      PaymentChannelData
	defaultKey          PaymentChannelKey
	mpeContractAddress  common.Address
	senderPrivateKey    *ecdsa.PrivateKey
	senderAddress       common.Address
	recipientPrivateKey *ecdsa.PrivateKey
	recipientAddress    common.Address
	delegateStorage     *storageMockType
	combinedStorage     combinedStorage
}

var storageTest = func() storageTestType {
	ethereum := blockchain.GetSimulatedEthereumEnvironment()

	mpeContractAddress := ethereum.MultiPartyEscrowAddress
	senderPrivateKey := ethereum.ClientPrivateKey
	senderAddress := ethereum.ClientWallet.From
	recipientPrivateKey := ethereum.ServerPrivateKey
	recipientAddress := ethereum.ServerWallet.From
	delegateStorage := &storageMockType{
		delegate: NewPaymentChannelStorage(NewMemStorage()),
		errors:   make(map[string]bool),
	}
	return storageTestType{
		ethereum: ethereum,
		defaultChannel: PaymentChannelData{
			Nonce:            big.NewInt(3),
			State:            Open,
			Sender:           senderAddress,
			Recipient:        recipientAddress,
			GroupId:          big.NewInt(0),
			FullAmount:       big.NewInt(12345),
			Expiration:       time.Now().Add(time.Hour),
			AuthorizedAmount: big.NewInt(12300),
			Signature:        getSignature(&mpeContractAddress, 42, 3, 12300, senderPrivateKey),
		},
		defaultKey: PaymentChannelKey{
			ID: big.NewInt(42),
		},
		mpeContractAddress:  mpeContractAddress,
		senderPrivateKey:    senderPrivateKey,
		senderAddress:       senderAddress,
		recipientPrivateKey: recipientPrivateKey,
		recipientAddress:    recipientAddress,
		delegateStorage:     delegateStorage,
		combinedStorage: combinedStorage{
			delegate: delegateStorage,
			mpe:      ethereum.MultiPartyEscrow,
		},
	}
}()

type multiPartyEscrowMock struct {
	blockchain.MultiPartyEscrow
	channels func(opts *bind.CallOpts, arg0 *big.Int) (struct {
		Sender     common.Address
		Recipient  common.Address
		ReplicaId  *big.Int
		Value      *big.Int
		Nonce      *big.Int
		Expiration *big.Int
	}, error)
}

func (mpe *multiPartyEscrowMock) Channels(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	ReplicaId  *big.Int
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
}, error) {
	return mpe.channels(opts, arg0)
}

func TestCombinedStorageGetAlreadyInStorage(t *testing.T) {
	expectedChannel := &PaymentChannelData{
		Nonce:            big.NewInt(3),
		State:            Open,
		Sender:           storageTest.senderAddress,
		Recipient:        storageTest.recipientAddress,
		GroupId:          big.NewInt(0),
		FullAmount:       big.NewInt(12345),
		Expiration:       time.Now().Add(time.Hour),
		AuthorizedAmount: big.NewInt(12300),
		Signature:        getSignature(&storageTest.mpeContractAddress, 42, 3, 12300, storageTest.senderPrivateKey),
	}
	expectedKey := &PaymentChannelKey{
		ID: big.NewInt(42),
	}
	storageTest.delegateStorage.Put(expectedKey, expectedChannel)
	defer storageTest.delegateStorage.Clear()

	channel, ok, err := storageTest.combinedStorage.Get(expectedKey)

	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, toJSON(expectedChannel), toJSON(channel))
}

func TestCombinedStorageGetReadFromBlockchain(t *testing.T) {
	expiration := time.Unix(time.Now().Add(time.Hour).Unix(), 0)
	expectedChannel := &PaymentChannelData{
		Nonce:            big.NewInt(0),
		State:            Open,
		Sender:           storageTest.senderAddress,
		Recipient:        storageTest.recipientAddress,
		GroupId:          big.NewInt(0),
		FullAmount:       big.NewInt(12345),
		Expiration:       expiration,
		AuthorizedAmount: big.NewInt(0),
		Signature:        nil,
	}
	expectedKey := &PaymentChannelKey{
		ID: big.NewInt(0),
	}
	ethereum := storageTest.ethereum
	ethereum.SnetTransferTokens(ethereum.ClientWallet, 12345).SnetApproveMpe(ethereum.ClientWallet, 12345).MpeDeposit(ethereum.ClientWallet, 12345).MpeOpenChannel(ethereum.ClientWallet, ethereum.ServerWallet, 12345, expiration, 0).Commit()

	channel, ok, err := storageTest.combinedStorage.Get(expectedKey)

	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, toJSON(expectedChannel), toJSON(channel))
}
