//go:generate nodejs ../resources/blockchain/scripts/generateAbi.js --contract-package singularitynet-token-contracts --contract-name SingularityNetToken --go-package blockchain --output-file singularity_net_token.go

package blockchain

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type SimulatedEthereumEnvironment struct {
	SingnetPrivateKey       *ecdsa.PrivateKey
	SingnetWallet           *bind.TransactOpts
	ClientWallet            *bind.TransactOpts
	ClientPrivateKey        *ecdsa.PrivateKey
	ServerWallet            *bind.TransactOpts
	ServerPrivateKey        *ecdsa.PrivateKey
	Backend                 *backends.SimulatedBackend
	SingularityNetToken     *SingularityNetToken
	MultiPartyEscrowAddress common.Address
	MultiPartyEscrow        *MultiPartyEscrow
}

func GetSimulatedEthereumEnvironment() (env SimulatedEthereumEnvironment) {
	env.SingnetPrivateKey, env.SingnetWallet = getTestWallet()
	env.ClientPrivateKey, env.ClientWallet = getTestWallet()
	env.ServerPrivateKey, env.ServerWallet = getTestWallet()

	alloc := map[common.Address]core.GenesisAccount{
		env.SingnetWallet.From: {Balance: big.NewInt(1000000000000)},
		env.ClientWallet.From:  {Balance: big.NewInt(1000000000000)},
		env.ServerWallet.From:  {Balance: big.NewInt(10000000)},
	}

	env.Backend = backends.NewSimulatedBackend(alloc)
	deployContracts(&env)

	return
}

func getTestWallet() (privateKey *ecdsa.PrivateKey, wallet *bind.TransactOpts) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Unable to generate private key, error: %v", err))
	}

	return privateKey, bind.NewKeyedTransactor(privateKey)
}

func deployContracts(env *SimulatedEthereumEnvironment) {
	tokenAddress, _, token, err := DeploySingularityNetToken(EstimateGas(env.SingnetWallet), env.Backend)
	if err != nil {
		panic(fmt.Sprintf("Unable to deploy SingularityNetToken contract, error: %v", err))
	}
	env.Backend.Commit()
	env.SingularityNetToken = token

	mpeAddress, _, mpe, err := DeployMultiPartyEscrow(EstimateGas(env.SingnetWallet), env.Backend, tokenAddress)
	if err != nil {
		panic(fmt.Sprintf("Unable to deploy MultiPartyEscrow contract, error: %v", err))
	}
	env.Backend.Commit()
	env.MultiPartyEscrow = mpe
	env.MultiPartyEscrowAddress = mpeAddress
}

func EstimateGas(wallet *bind.TransactOpts) (opts *bind.TransactOpts) {
	return &bind.TransactOpts{
		From:     wallet.From,
		Signer:   wallet.Signer,
		Value:    nil,
		GasLimit: 0,
	}
}

func SetGas(wallet *bind.TransactOpts, gasLimit uint64) (opts *bind.TransactOpts) {
	return &bind.TransactOpts{
		From:     wallet.From,
		Signer:   wallet.Signer,
		Value:    nil,
		GasLimit: gasLimit,
	}
}
