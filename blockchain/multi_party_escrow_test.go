//go:generate nodejs ../resources/blockchain/scripts/generateAbi.js --contract-package singularitynet-token-contracts --contract-name SingularityNetToken --go-package blockchain --output-file singularity_net_token_test.go

package blockchain

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"testing"
)

var mpeTest = getSimulatedEthereumEnvironment()
var multiPartyEscrowContract = getMultiPartyEscrowContract(mpeTest.singnetWallet)

type ethereumEnvironment struct {
	singnetWallet *bind.TransactOpts
	clientWallet  *bind.TransactOpts
	serverWallet  *bind.TransactOpts
	eth           *backends.SimulatedBackend
}

func getTestWallet() (wallet *bind.TransactOpts) {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Unable to generate private key, error: %v", err))
	}

	return bind.NewKeyedTransactor(key)
}

func getSimulatedEthereumEnvironment() (env ethereumEnvironment) {
	env.singnetWallet = getTestWallet()
	env.clientWallet = getTestWallet()
	env.serverWallet = getTestWallet()

	alloc := map[common.Address]core.GenesisAccount{
		env.singnetWallet.From: {Balance: big.NewInt(1000000000000)},
		env.clientWallet.From:  {Balance: big.NewInt(1000000000000)},
		env.serverWallet.From:  {Balance: big.NewInt(0)},
	}

	env.eth = backends.NewSimulatedBackend(alloc)
	return
}

func getMultiPartyEscrowContract(deployer *bind.TransactOpts) (mpe *MultiPartyEscrow) {
	tokenAddress, _, _, err := DeploySingularityNetToken(estimateGas(deployer), mpeTest.eth)
	if err != nil {
		panic(fmt.Sprintf("Unable to deploy SingularityNetToken contract, error: %v", err))
	}
	mpeTest.eth.Commit()

	_, _, mpe, err = DeployMultiPartyEscrow(estimateGas(deployer), mpeTest.eth, tokenAddress)
	if err != nil {
		panic(fmt.Sprintf("Unable to deploy MultiPartyEscrow contract, error: %v", err))
	}
	mpeTest.eth.Commit()

	return mpe
}

func estimateGas(wallet *bind.TransactOpts) (opts *bind.TransactOpts) {
	return &bind.TransactOpts{
		From:     wallet.From,
		Signer:   wallet.Signer,
		Value:    nil,
		GasLimit: 0,
	}
}

func setGas(wallet *bind.TransactOpts, gasLimit uint64) (opts *bind.TransactOpts) {
	return &bind.TransactOpts{
		From:     wallet.From,
		Signer:   wallet.Signer,
		Value:    nil,
		GasLimit: gasLimit,
	}
}

func TestMain(m *testing.M) {
	result := m.Run()

	os.Exit(result)
}

func TestDepositAndOpenChannel(t *testing.T) {

	_, err := multiPartyEscrowContract.DepositAndOpenChannel(
		setGas(mpeTest.clientWallet, 100000),
		mpeTest.serverWallet.From,
		big.NewInt(1000),
		big.NewInt(42),
		big.NewInt(1),
	)
	mpeTest.eth.Commit()

	assert.Nil(t, err)
}
