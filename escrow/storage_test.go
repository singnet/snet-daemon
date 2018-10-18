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
		delegate: NewMemStorage(),
		errors:   make(map[memoryStorageKey]bool),
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
			ID:    big.NewInt(42),
			Nonce: big.NewInt(3),
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
		ID:    big.NewInt(42),
		Nonce: big.NewInt(3),
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
		ID:    big.NewInt(0),
		Nonce: big.NewInt(0),
	}
	depositAndOpenChannel(t, 12345, expiration, 0)

	channel, ok, err := storageTest.combinedStorage.Get(expectedKey)

	assert.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, toJSON(expectedChannel), toJSON(channel))
}

func depositAndOpenChannel(t *testing.T, amount int64, expiration time.Time, groupId int64) {
	ethereum := storageTest.ethereum

	_, err := ethereum.SingularityNetToken.TransferTokens(
		blockchain.EstimateGas(ethereum.SingnetWallet),
		ethereum.ClientWallet.From,
		big.NewInt(amount),
	)
	assert.Nil(t, err)
	ethereum.Backend.Commit()

	tokenBalance, err := ethereum.SingularityNetToken.BalanceOf(nil, ethereum.ClientWallet.From)
	assert.Nil(t, err)
	assert.Equal(t, amount, tokenBalance.Int64())

	_, err = ethereum.SingularityNetToken.Approve(
		blockchain.EstimateGas(ethereum.ClientWallet),
		ethereum.MultiPartyEscrowAddress,
		big.NewInt(amount))
	assert.Nil(t, err)
	ethereum.Backend.Commit()

	_, err = ethereum.MultiPartyEscrow.Deposit(
		blockchain.EstimateGas(ethereum.ClientWallet),
		big.NewInt(amount),
	)
	assert.Nil(t, err)
	ethereum.Backend.Commit()

	mpeBalance, err := ethereum.MultiPartyEscrow.Balances(nil, ethereum.ClientWallet.From)
	assert.Nil(t, err)
	assert.Equal(t, amount, mpeBalance.Int64())

	_, err = ethereum.MultiPartyEscrow.OpenChannel(
		blockchain.EstimateGas(ethereum.ClientWallet),
		ethereum.ServerWallet.From,
		big.NewInt(amount),
		big.NewInt(expiration.Unix()),
		big.NewInt(groupId),
	)
	assert.Nil(t, err)
	ethereum.Backend.Commit()

	id, err := ethereum.MultiPartyEscrow.NextChannelId(nil)
	assert.Nil(t, err)

	ch, err := ethereum.MultiPartyEscrow.Channels(nil, (&big.Int{}).Sub(id, big.NewInt(1)))
	assert.Nil(t, err)
	assert.NotEqual(t, zeroAddress, ch.Sender)
}

/*
	storageTest.combinedStorage.channels = func(opts *bind.CallOpts, arg0 *big.Int) (struct {
		Sender     common.Address
		Recipient  common.Address
		ReplicaId  *big.Int
		Value      *big.Int
		Nonce      *big.Int
		Expiration *big.Int
	}, error) {
		return struct {
			Sender     common.Address
			Recipient  common.Address
			ReplicaId  *big.Int
			Value      *big.Int
			Nonce      *big.Int
			Expiration *big.Int
		}{
			Sender:     storageTest.senderAddress,
			Recipient:  storageTest.recipientAddress,
			ReplicaId:  big.NewInt(0),
			Value:      big.NewInt(12345),
			Nonce:      big.NewInt(0),
			Expiration: big.NewInt(time.Now().Add(time.Hour).Unix()),
		}, nil
	}
*/
