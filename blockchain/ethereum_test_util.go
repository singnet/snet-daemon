package blockchain

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type SimulatedEthereumEnvironment struct {
	SingnetPrivateKey       *ecdsa.PrivateKey
	SingnetWallet           *bind.TransactOpts
	ClientWallet            *bind.TransactOpts
	ClientPrivateKey        *ecdsa.PrivateKey
	ServerWallet            *bind.TransactOpts
	ServerPrivateKey        *ecdsa.PrivateKey
	Backend                 *simulated.Backend
	FetToken                *FetchToken
	MultiPartyEscrowAddress common.Address
	MultiPartyEscrow        *MultiPartyEscrow
}

func (env *SimulatedEthereumEnvironment) SnetTransferTokens(to *bind.TransactOpts, amount int64) *SimulatedEthereumEnvironment {
	_, err := env.FetToken.Transfer(EstimateGas(env.SingnetWallet), to.From, big.NewInt(amount))
	if err != nil {
		panic(fmt.Sprintf("Unable to transfer tokens: %v", err))
	}
	return env
}

func (env *SimulatedEthereumEnvironment) SnetApproveMpe(from *bind.TransactOpts, amount int64) *SimulatedEthereumEnvironment {
	_, err := env.FetToken.Approve(EstimateGas(from), env.MultiPartyEscrowAddress, big.NewInt(amount))
	if err != nil {
		panic(fmt.Sprintf("Unable to aprove tokens transfer to MPE: %v", err))
	}
	return env
}

func (env *SimulatedEthereumEnvironment) MpeDeposit(from *bind.TransactOpts, amount int64) *SimulatedEthereumEnvironment {
	_, err := env.MultiPartyEscrow.Deposit(EstimateGas(from), big.NewInt(amount))
	if err != nil {
		panic(fmt.Sprintf("Unable to deposit tokens to MPE: %v", err))
	}
	return env
}

func (env *SimulatedEthereumEnvironment) MpeOpenChannel(from *bind.TransactOpts, to *bind.TransactOpts, amount int64, expiration int64, groupId [32]byte) *SimulatedEthereumEnvironment {
	_, err := env.MultiPartyEscrow.OpenChannel(EstimateGas(from), from.From, to.From, groupId, big.NewInt(amount), big.NewInt(expiration))
	if err != nil {
		panic(fmt.Sprintf("Unable to open MPE payment channel: %v", err))
	}
	return env
}

func (env *SimulatedEthereumEnvironment) Commit() *SimulatedEthereumEnvironment {
	env.Backend.Commit()
	return env
}

func GetSimulatedEthereumEnvironment() (env SimulatedEthereumEnvironment) {
	var chainID = big.NewInt(11155111)
	env.SingnetPrivateKey, env.SingnetWallet, _ = getTestWallet(chainID)
	env.ClientPrivateKey, env.ClientWallet, _ = getTestWallet(chainID)
	env.ServerPrivateKey, env.ServerWallet, _ = getTestWallet(chainID)

	alloc := map[common.Address]types.Account{
		env.SingnetWallet.From: {Balance: big.NewInt(1000000000000)},
		env.ClientWallet.From:  {Balance: big.NewInt(1000000000000)},
		env.ServerWallet.From:  {Balance: big.NewInt(10000000)},
	}

	b := simulated.NewBackend(alloc, simulated.WithBlockGasLimit(0))

	env.Backend = b
	deployContracts(&env)

	return
}

func getTestWallet(chainID *big.Int) (privateKey *ecdsa.PrivateKey, wallet *bind.TransactOpts, err error) {
	privateKey, err = crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Unable to generate private key, error: %v", err))
	}

	wallet, err = bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, wallet, err
}

func deployContracts(env *SimulatedEthereumEnvironment) {
	tokenAddress, _, token, err := DeployFetchToken(EstimateGas(env.SingnetWallet), env.Backend.Client(), "Fetch Token", "ASI", big.NewInt(1000000000))
	if err != nil {
		panic(fmt.Sprintf("Unable to deploy FetchToken contract, error: %v", err))
	}
	env.Backend.Commit()
	env.FetToken = token

	mpeAddress, _, mpe, err := DeployMultiPartyEscrow(EstimateGas(env.SingnetWallet), env.Backend.Client(), tokenAddress)
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
