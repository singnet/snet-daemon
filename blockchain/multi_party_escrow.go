// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package blockchain

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// MultiPartyEscrowABI is the input ABI used to generate the binding from.
const MultiPartyEscrowABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"channels\",\"outputs\":[{\"name\":\"sender\",\"type\":\"address\"},{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"groupId\",\"type\":\"uint256\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\"},{\"name\":\"expiration\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nextChannelId\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"channelId\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"groupId\",\"type\":\"uint256\"}],\"name\":\"EventChannelOpen\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"expiration\",\"type\":\"uint256\"},{\"name\":\"groupId\",\"type\":\"uint256\"}],\"name\":\"openChannel\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"expiration\",\"type\":\"uint256\"},{\"name\":\"groupId\",\"type\":\"uint256\"}],\"name\":\"depositAndOpenChannel\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"signature\",\"type\":\"bytes\"},{\"name\":\"isSendback\",\"type\":\"bool\"}],\"name\":\"channelClaim\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"newExpiration\",\"type\":\"uint256\"}],\"name\":\"channelExtend\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"channelAddFunds\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"newExpiration\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"channelExtendAndAddFunds\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"}],\"name\":\"channelClaimTimeout\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// MultiPartyEscrowBin is the compiled bytecode used for deploying new contracts.
const MultiPartyEscrowBin = `0x608060405234801561001057600080fd5b50604051602080610cf7833981016040525160038054600160a060020a031916600160a060020a03909216919091179055610ca7806100506000396000f3006080604052600436106100c45763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630c19d0ec81146100c9578063114d104e146100e957806327e235e3146101275780632e1a7d4d1461015a578063447fdacb1461017257806345059a5d146101d6578063b6b55f25146101f1578063baea65b514610209578063da2a5b4f14610221578063e5949b5d1461023c578063f36bdb4914610293578063f4606f00146102bd578063fc0c546a146102d2575b600080fd5b3480156100d557600080fd5b506100e7600435602435604435610303565b005b3480156100f557600080fd5b50610113600160a060020a0360043516602435604435606435610332565b604080519115158252519081900360200190f35b34801561013357600080fd5b50610148600160a060020a036004351661036a565b60408051918252519081900360200190f35b34801561016657600080fd5b5061011360043561037c565b34801561017e57600080fd5b50604080516020600460443581810135601f81018490048402850184019095528484526100e7948235946024803595369594606494920191908190840183828082843750949750505050913515159250610478915050565b3480156101e257600080fd5b50610113600435602435610635565b3480156101fd57600080fd5b50610113600435610684565b34801561021557600080fd5b506100e76004356107d5565b34801561022d57600080fd5b50610113600435602435610822565b34801561024857600080fd5b506102546004356108d3565b60408051600160a060020a039788168152959096166020860152848601939093526060840191909152608083015260a082015290519081900360c00190f35b34801561029f57600080fd5b50610113600160a060020a0360043516602435604435606435610915565b3480156102c957600080fd5b50610148610a5a565b3480156102de57600080fd5b506102e7610a60565b60408051600160a060020a039092168252519081900360200190f35b61030d8383610635565b151561031857600080fd5b6103228382610822565b151561032d57600080fd5b505050565b600061033d84610684565b151561034857600080fd5b61035485858585610915565b151561035f57600080fd5b506001949350505050565b60016020526000908152604090205481565b3360009081526001602052604081205482111561039857600080fd5b600354604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152336004820152602481018590529051600160a060020a039092169163a9059cbb916044808201926020929091908290030181600087803b15801561040557600080fd5b505af1158015610419573d6000803e3d6000fd5b505050506040513d602081101561042f57600080fd5b5051151561043c57600080fd5b3360009081526001602052604090205461045c908363ffffffff610a6f16565b3360009081526001602081905260409091209190915592915050565b6000848152602081905260408120600381015490919085111561049a57600080fd5b6001820154600160a060020a031633146104b357600080fd5b61057430878460040154886040516020018085600160a060020a0316600160a060020a03166c010000000000000000000000000281526014018481526020018381526020018281526020019450505050506040516020818303038152906040526040518082805190602001908083835b602083106105425780518252601f199092019160209182019101610523565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040518091039020610a81565b8254909150600160a060020a031661058c8286610b2b565b600160a060020a03161461059f57600080fd5b6000868152602081905260409020600301546105c1908663ffffffff610a6f16565b600087815260208181526040808320600301939093553382526001905220546105f0908663ffffffff610bb216565b3360009081526001602052604090205582156106145761060f86610bc5565b61062d565b6000868152602081905260409020600401805460010190555b505050505050565b60008281526020819052604081208054600160a060020a0316331461065957600080fd5b600581015483101561066a57600080fd5b505060009182526020829052604090912060050155600190565b600354604080517f23b872dd000000000000000000000000000000000000000000000000000000008152336004820152306024820152604481018490529051600092600160a060020a0316916323b872dd91606480830192602092919082900301818787803b1580156106f657600080fd5b505af115801561070a573d6000803e3d6000fd5b505050506040513d602081101561072057600080fd5b505115156107b557604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602860248201527f556e61626c6520746f207472616e7366657220746f6b656e20746f207468652060448201527f636f6e7472616374000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b3360009081526001602052604090205461045c908363ffffffff610bb216565b600081815260208190526040902054600160a060020a031633146107f857600080fd5b60008181526020819052604090206005015443101561081657600080fd5b61081f81610bc5565b50565b33600090815260016020526040812054819083111561084057600080fd5b5060008381526020819052604090208054600160a060020a0316331461086557600080fd5b33600090815260016020526040902054610885908463ffffffff610a6f16565b3360009081526001602090815260408083209390935586825281905220600301546108b6908463ffffffff610bb216565b600085815260208190526040902060030155600191505092915050565b600060208190529081526040902080546001820154600283015460038401546004850154600590950154600160a060020a039485169593909416939192909186565b3360009081526001602052604081205484111561093157600080fd5b6040805160c08101825233808252600160a060020a038881166020808501918252848601888152606086018b815260006080880181815260a089018d81526002805484528387528b84209a518b54908a1673ffffffffffffffffffffffffffffffffffffffff19918216178c5597516001808d01805492909b169190991617909855935196890196909655905160038801559351600487015551600590950194909455918152915220546109eb908563ffffffff610a6f16565b3360008181526001602090815260409182902093909355600254815190815290518593600160a060020a038a1693927f3a8e4a77be297b479558947d2e876a8340e0eff8cbf8bdfa2a5b548ef56e5bba929081900390910190a450600280546001908101909155949350505050565b60025481565b600354600160a060020a031681565b600082821115610a7b57fe5b50900390565b604080517f19457468657265756d205369676e6564204d6573736167653a0a333200000000602080830191909152603c80830185905283518084039091018152605c909201928390528151600093918291908401908083835b60208310610af95780518252601f199092019160209182019101610ada565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b600080600080610b3a85610c38565b60408051600080825260208083018085528d905260ff8716838501526060830186905260808301859052925195985093965091945060019360a0808401949293601f19830193908390039091019190865af1158015610b9d573d6000803e3d6000fd5b5050604051601f190151979650505050505050565b81810182811015610bbf57fe5b92915050565b60008181526020818152604080832060038101548154600160a060020a031685526001909352922054610bfd9163ffffffff610bb216565b8154600160a060020a031660009081526001602081905260408220929092556003830181905560048301805490920190915560059091015550565b600080600083516041141515610c4d57600080fd5b50505060208101516040820151604183015160ff169190601b831015610c7457601b830192505b91939092505600a165627a7a7230582066d2994f23712fd362abcefb6acd921e31a3d1dbc4d1e99f429a7fc27f68d5620029`

// DeployMultiPartyEscrow deploys a new Ethereum contract, binding an instance of MultiPartyEscrow to it.
func DeployMultiPartyEscrow(auth *bind.TransactOpts, backend bind.ContractBackend, _token common.Address) (common.Address, *types.Transaction, *MultiPartyEscrow, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiPartyEscrowABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MultiPartyEscrowBin), backend, _token)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MultiPartyEscrow{MultiPartyEscrowCaller: MultiPartyEscrowCaller{contract: contract}, MultiPartyEscrowTransactor: MultiPartyEscrowTransactor{contract: contract}, MultiPartyEscrowFilterer: MultiPartyEscrowFilterer{contract: contract}}, nil
}

// MultiPartyEscrow is an auto generated Go binding around an Ethereum contract.
type MultiPartyEscrow struct {
	MultiPartyEscrowCaller     // Read-only binding to the contract
	MultiPartyEscrowTransactor // Write-only binding to the contract
	MultiPartyEscrowFilterer   // Log filterer for contract events
}

// MultiPartyEscrowCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultiPartyEscrowCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiPartyEscrowTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiPartyEscrowTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiPartyEscrowFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiPartyEscrowFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiPartyEscrowSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiPartyEscrowSession struct {
	Contract     *MultiPartyEscrow // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultiPartyEscrowCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiPartyEscrowCallerSession struct {
	Contract *MultiPartyEscrowCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// MultiPartyEscrowTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiPartyEscrowTransactorSession struct {
	Contract     *MultiPartyEscrowTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// MultiPartyEscrowRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultiPartyEscrowRaw struct {
	Contract *MultiPartyEscrow // Generic contract binding to access the raw methods on
}

// MultiPartyEscrowCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiPartyEscrowCallerRaw struct {
	Contract *MultiPartyEscrowCaller // Generic read-only contract binding to access the raw methods on
}

// MultiPartyEscrowTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiPartyEscrowTransactorRaw struct {
	Contract *MultiPartyEscrowTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiPartyEscrow creates a new instance of MultiPartyEscrow, bound to a specific deployed contract.
func NewMultiPartyEscrow(address common.Address, backend bind.ContractBackend) (*MultiPartyEscrow, error) {
	contract, err := bindMultiPartyEscrow(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrow{MultiPartyEscrowCaller: MultiPartyEscrowCaller{contract: contract}, MultiPartyEscrowTransactor: MultiPartyEscrowTransactor{contract: contract}, MultiPartyEscrowFilterer: MultiPartyEscrowFilterer{contract: contract}}, nil
}

// NewMultiPartyEscrowCaller creates a new read-only instance of MultiPartyEscrow, bound to a specific deployed contract.
func NewMultiPartyEscrowCaller(address common.Address, caller bind.ContractCaller) (*MultiPartyEscrowCaller, error) {
	contract, err := bindMultiPartyEscrow(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowCaller{contract: contract}, nil
}

// NewMultiPartyEscrowTransactor creates a new write-only instance of MultiPartyEscrow, bound to a specific deployed contract.
func NewMultiPartyEscrowTransactor(address common.Address, transactor bind.ContractTransactor) (*MultiPartyEscrowTransactor, error) {
	contract, err := bindMultiPartyEscrow(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowTransactor{contract: contract}, nil
}

// NewMultiPartyEscrowFilterer creates a new log filterer instance of MultiPartyEscrow, bound to a specific deployed contract.
func NewMultiPartyEscrowFilterer(address common.Address, filterer bind.ContractFilterer) (*MultiPartyEscrowFilterer, error) {
	contract, err := bindMultiPartyEscrow(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowFilterer{contract: contract}, nil
}

// bindMultiPartyEscrow binds a generic wrapper to an already deployed contract.
func bindMultiPartyEscrow(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiPartyEscrowABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiPartyEscrow *MultiPartyEscrowRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MultiPartyEscrow.Contract.MultiPartyEscrowCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiPartyEscrow *MultiPartyEscrowRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.MultiPartyEscrowTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiPartyEscrow *MultiPartyEscrowRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.MultiPartyEscrowTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiPartyEscrow *MultiPartyEscrowCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MultiPartyEscrow.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiPartyEscrow *MultiPartyEscrowTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiPartyEscrow *MultiPartyEscrowTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.contract.Transact(opts, method, params...)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances( address) constant returns(uint256)
func (_MultiPartyEscrow *MultiPartyEscrowCaller) Balances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _MultiPartyEscrow.contract.Call(opts, out, "balances", arg0)
	return *ret0, err
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances( address) constant returns(uint256)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _MultiPartyEscrow.Contract.Balances(&_MultiPartyEscrow.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances( address) constant returns(uint256)
func (_MultiPartyEscrow *MultiPartyEscrowCallerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _MultiPartyEscrow.Contract.Balances(&_MultiPartyEscrow.CallOpts, arg0)
}

// Channels is a free data retrieval call binding the contract method 0xe5949b5d.
//
// Solidity: function channels( uint256) constant returns(sender address, recipient address, groupId uint256, value uint256, nonce uint256, expiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowCaller) Channels(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    *big.Int
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
}, error) {
	ret := new(struct {
		Sender     common.Address
		Recipient  common.Address
		GroupId    *big.Int
		Value      *big.Int
		Nonce      *big.Int
		Expiration *big.Int
	})
	out := ret
	err := _MultiPartyEscrow.contract.Call(opts, out, "channels", arg0)
	return *ret, err
}

// Channels is a free data retrieval call binding the contract method 0xe5949b5d.
//
// Solidity: function channels( uint256) constant returns(sender address, recipient address, groupId uint256, value uint256, nonce uint256, expiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Channels(arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    *big.Int
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
}, error) {
	return _MultiPartyEscrow.Contract.Channels(&_MultiPartyEscrow.CallOpts, arg0)
}

// Channels is a free data retrieval call binding the contract method 0xe5949b5d.
//
// Solidity: function channels( uint256) constant returns(sender address, recipient address, groupId uint256, value uint256, nonce uint256, expiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowCallerSession) Channels(arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    *big.Int
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
}, error) {
	return _MultiPartyEscrow.Contract.Channels(&_MultiPartyEscrow.CallOpts, arg0)
}

// NextChannelId is a free data retrieval call binding the contract method 0xf4606f00.
//
// Solidity: function nextChannelId() constant returns(uint256)
func (_MultiPartyEscrow *MultiPartyEscrowCaller) NextChannelId(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _MultiPartyEscrow.contract.Call(opts, out, "nextChannelId")
	return *ret0, err
}

// NextChannelId is a free data retrieval call binding the contract method 0xf4606f00.
//
// Solidity: function nextChannelId() constant returns(uint256)
func (_MultiPartyEscrow *MultiPartyEscrowSession) NextChannelId() (*big.Int, error) {
	return _MultiPartyEscrow.Contract.NextChannelId(&_MultiPartyEscrow.CallOpts)
}

// NextChannelId is a free data retrieval call binding the contract method 0xf4606f00.
//
// Solidity: function nextChannelId() constant returns(uint256)
func (_MultiPartyEscrow *MultiPartyEscrowCallerSession) NextChannelId() (*big.Int, error) {
	return _MultiPartyEscrow.Contract.NextChannelId(&_MultiPartyEscrow.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_MultiPartyEscrow *MultiPartyEscrowCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MultiPartyEscrow.contract.Call(opts, out, "token")
	return *ret0, err
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Token() (common.Address, error) {
	return _MultiPartyEscrow.Contract.Token(&_MultiPartyEscrow.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() constant returns(address)
func (_MultiPartyEscrow *MultiPartyEscrowCallerSession) Token() (common.Address, error) {
	return _MultiPartyEscrow.Contract.Token(&_MultiPartyEscrow.CallOpts)
}

// ChannelAddFunds is a paid mutator transaction binding the contract method 0xda2a5b4f.
//
// Solidity: function channelAddFunds(channelId uint256, amount uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) ChannelAddFunds(opts *bind.TransactOpts, channelId *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "channelAddFunds", channelId, amount)
}

// ChannelAddFunds is a paid mutator transaction binding the contract method 0xda2a5b4f.
//
// Solidity: function channelAddFunds(channelId uint256, amount uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) ChannelAddFunds(channelId *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelAddFunds(&_MultiPartyEscrow.TransactOpts, channelId, amount)
}

// ChannelAddFunds is a paid mutator transaction binding the contract method 0xda2a5b4f.
//
// Solidity: function channelAddFunds(channelId uint256, amount uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) ChannelAddFunds(channelId *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelAddFunds(&_MultiPartyEscrow.TransactOpts, channelId, amount)
}

// ChannelClaim is a paid mutator transaction binding the contract method 0x447fdacb.
//
// Solidity: function channelClaim(channelId uint256, amount uint256, signature bytes, isSendback bool) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) ChannelClaim(opts *bind.TransactOpts, channelId *big.Int, amount *big.Int, signature []byte, isSendback bool) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "channelClaim", channelId, amount, signature, isSendback)
}

// ChannelClaim is a paid mutator transaction binding the contract method 0x447fdacb.
//
// Solidity: function channelClaim(channelId uint256, amount uint256, signature bytes, isSendback bool) returns()
func (_MultiPartyEscrow *MultiPartyEscrowSession) ChannelClaim(channelId *big.Int, amount *big.Int, signature []byte, isSendback bool) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelClaim(&_MultiPartyEscrow.TransactOpts, channelId, amount, signature, isSendback)
}

// ChannelClaim is a paid mutator transaction binding the contract method 0x447fdacb.
//
// Solidity: function channelClaim(channelId uint256, amount uint256, signature bytes, isSendback bool) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) ChannelClaim(channelId *big.Int, amount *big.Int, signature []byte, isSendback bool) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelClaim(&_MultiPartyEscrow.TransactOpts, channelId, amount, signature, isSendback)
}

// ChannelClaimTimeout is a paid mutator transaction binding the contract method 0xbaea65b5.
//
// Solidity: function channelClaimTimeout(channelId uint256) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) ChannelClaimTimeout(opts *bind.TransactOpts, channelId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "channelClaimTimeout", channelId)
}

// ChannelClaimTimeout is a paid mutator transaction binding the contract method 0xbaea65b5.
//
// Solidity: function channelClaimTimeout(channelId uint256) returns()
func (_MultiPartyEscrow *MultiPartyEscrowSession) ChannelClaimTimeout(channelId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelClaimTimeout(&_MultiPartyEscrow.TransactOpts, channelId)
}

// ChannelClaimTimeout is a paid mutator transaction binding the contract method 0xbaea65b5.
//
// Solidity: function channelClaimTimeout(channelId uint256) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) ChannelClaimTimeout(channelId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelClaimTimeout(&_MultiPartyEscrow.TransactOpts, channelId)
}

// ChannelExtend is a paid mutator transaction binding the contract method 0x45059a5d.
//
// Solidity: function channelExtend(channelId uint256, newExpiration uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) ChannelExtend(opts *bind.TransactOpts, channelId *big.Int, newExpiration *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "channelExtend", channelId, newExpiration)
}

// ChannelExtend is a paid mutator transaction binding the contract method 0x45059a5d.
//
// Solidity: function channelExtend(channelId uint256, newExpiration uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) ChannelExtend(channelId *big.Int, newExpiration *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelExtend(&_MultiPartyEscrow.TransactOpts, channelId, newExpiration)
}

// ChannelExtend is a paid mutator transaction binding the contract method 0x45059a5d.
//
// Solidity: function channelExtend(channelId uint256, newExpiration uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) ChannelExtend(channelId *big.Int, newExpiration *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelExtend(&_MultiPartyEscrow.TransactOpts, channelId, newExpiration)
}

// ChannelExtendAndAddFunds is a paid mutator transaction binding the contract method 0x0c19d0ec.
//
// Solidity: function channelExtendAndAddFunds(channelId uint256, newExpiration uint256, amount uint256) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) ChannelExtendAndAddFunds(opts *bind.TransactOpts, channelId *big.Int, newExpiration *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "channelExtendAndAddFunds", channelId, newExpiration, amount)
}

// ChannelExtendAndAddFunds is a paid mutator transaction binding the contract method 0x0c19d0ec.
//
// Solidity: function channelExtendAndAddFunds(channelId uint256, newExpiration uint256, amount uint256) returns()
func (_MultiPartyEscrow *MultiPartyEscrowSession) ChannelExtendAndAddFunds(channelId *big.Int, newExpiration *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelExtendAndAddFunds(&_MultiPartyEscrow.TransactOpts, channelId, newExpiration, amount)
}

// ChannelExtendAndAddFunds is a paid mutator transaction binding the contract method 0x0c19d0ec.
//
// Solidity: function channelExtendAndAddFunds(channelId uint256, newExpiration uint256, amount uint256) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) ChannelExtendAndAddFunds(channelId *big.Int, newExpiration *big.Int, amount *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelExtendAndAddFunds(&_MultiPartyEscrow.TransactOpts, channelId, newExpiration, amount)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) Deposit(opts *bind.TransactOpts, value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "deposit", value)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Deposit(value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.Deposit(&_MultiPartyEscrow.TransactOpts, value)
}

// Deposit is a paid mutator transaction binding the contract method 0xb6b55f25.
//
// Solidity: function deposit(value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) Deposit(value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.Deposit(&_MultiPartyEscrow.TransactOpts, value)
}

// DepositAndOpenChannel is a paid mutator transaction binding the contract method 0x114d104e.
//
// Solidity: function depositAndOpenChannel(recipient address, value uint256, expiration uint256, groupId uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) DepositAndOpenChannel(opts *bind.TransactOpts, recipient common.Address, value *big.Int, expiration *big.Int, groupId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "depositAndOpenChannel", recipient, value, expiration, groupId)
}

// DepositAndOpenChannel is a paid mutator transaction binding the contract method 0x114d104e.
//
// Solidity: function depositAndOpenChannel(recipient address, value uint256, expiration uint256, groupId uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) DepositAndOpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.DepositAndOpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId)
}

// DepositAndOpenChannel is a paid mutator transaction binding the contract method 0x114d104e.
//
// Solidity: function depositAndOpenChannel(recipient address, value uint256, expiration uint256, groupId uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) DepositAndOpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.DepositAndOpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId)
}

// OpenChannel is a paid mutator transaction binding the contract method 0xf36bdb49.
//
// Solidity: function openChannel(recipient address, value uint256, expiration uint256, groupId uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) OpenChannel(opts *bind.TransactOpts, recipient common.Address, value *big.Int, expiration *big.Int, groupId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "openChannel", recipient, value, expiration, groupId)
}

// OpenChannel is a paid mutator transaction binding the contract method 0xf36bdb49.
//
// Solidity: function openChannel(recipient address, value uint256, expiration uint256, groupId uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) OpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.OpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId)
}

// OpenChannel is a paid mutator transaction binding the contract method 0xf36bdb49.
//
// Solidity: function openChannel(recipient address, value uint256, expiration uint256, groupId uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) OpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.OpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) Withdraw(opts *bind.TransactOpts, value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "withdraw", value)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Withdraw(value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.Withdraw(&_MultiPartyEscrow.TransactOpts, value)
}

// Withdraw is a paid mutator transaction binding the contract method 0x2e1a7d4d.
//
// Solidity: function withdraw(value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) Withdraw(value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.Withdraw(&_MultiPartyEscrow.TransactOpts, value)
}

// MultiPartyEscrowEventChannelOpenIterator is returned from FilterEventChannelOpen and is used to iterate over the raw logs and unpacked data for EventChannelOpen events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowEventChannelOpenIterator struct {
	Event *MultiPartyEscrowEventChannelOpen // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MultiPartyEscrowEventChannelOpenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowEventChannelOpen)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MultiPartyEscrowEventChannelOpen)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MultiPartyEscrowEventChannelOpenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowEventChannelOpenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowEventChannelOpen represents a EventChannelOpen event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowEventChannelOpen struct {
	ChannelId *big.Int
	Sender    common.Address
	Recipient common.Address
	GroupId   *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterEventChannelOpen is a free log retrieval operation binding the contract event 0x3a8e4a77be297b479558947d2e876a8340e0eff8cbf8bdfa2a5b548ef56e5bba.
//
// Solidity: e EventChannelOpen(channelId uint256, sender indexed address, recipient indexed address, groupId indexed uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterEventChannelOpen(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, groupId []*big.Int) (*MultiPartyEscrowEventChannelOpenIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var groupIdRule []interface{}
	for _, groupIdItem := range groupId {
		groupIdRule = append(groupIdRule, groupIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "EventChannelOpen", senderRule, recipientRule, groupIdRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowEventChannelOpenIterator{contract: _MultiPartyEscrow.contract, event: "EventChannelOpen", logs: logs, sub: sub}, nil
}

// WatchEventChannelOpen is a free log subscription operation binding the contract event 0x3a8e4a77be297b479558947d2e876a8340e0eff8cbf8bdfa2a5b548ef56e5bba.
//
// Solidity: e EventChannelOpen(channelId uint256, sender indexed address, recipient indexed address, groupId indexed uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchEventChannelOpen(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowEventChannelOpen, sender []common.Address, recipient []common.Address, groupId []*big.Int) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}
	var groupIdRule []interface{}
	for _, groupIdItem := range groupId {
		groupIdRule = append(groupIdRule, groupIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "EventChannelOpen", senderRule, recipientRule, groupIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowEventChannelOpen)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "EventChannelOpen", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
