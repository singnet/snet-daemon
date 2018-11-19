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
const MultiPartyEscrowABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"channels\",\"outputs\":[{\"name\":\"sender\",\"type\":\"address\"},{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"groupId\",\"type\":\"bytes32\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\"},{\"name\":\"expiration\",\"type\":\"uint256\"},{\"name\":\"signer\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"nextChannelId\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"channelId\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"groupId\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"signer\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"expiration\",\"type\":\"uint256\"}],\"name\":\"ChannelOpen\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"channelId\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"claimAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"sendBackAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"keepAmpount\",\"type\":\"uint256\"}],\"name\":\"ChannelClaim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"channelId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"claimAmount\",\"type\":\"uint256\"}],\"name\":\"ChannelSenderClaim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"channelId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"newExpiration\",\"type\":\"uint256\"}],\"name\":\"ChannelExtend\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"channelId\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"additionalFunds\",\"type\":\"uint256\"}],\"name\":\"ChannelAddFunds\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"receiver\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"TransferFunds\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"expiration\",\"type\":\"uint256\"},{\"name\":\"groupId\",\"type\":\"bytes32\"},{\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"openChannel\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"expiration\",\"type\":\"uint256\"},{\"name\":\"groupId\",\"type\":\"bytes32\"},{\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"depositAndOpenChannel\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelIds\",\"type\":\"uint256[]\"},{\"name\":\"amounts\",\"type\":\"uint256[]\"},{\"name\":\"isSendbacks\",\"type\":\"bool[]\"},{\"name\":\"v\",\"type\":\"uint8[]\"},{\"name\":\"r\",\"type\":\"bytes32[]\"},{\"name\":\"s\",\"type\":\"bytes32[]\"}],\"name\":\"multiChannelClaim\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"v\",\"type\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\"},{\"name\":\"isSendback\",\"type\":\"bool\"}],\"name\":\"channelClaim\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"newExpiration\",\"type\":\"uint256\"}],\"name\":\"channelExtend\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"channelAddFunds\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"},{\"name\":\"newExpiration\",\"type\":\"uint256\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"channelExtendAndAddFunds\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"channelId\",\"type\":\"uint256\"}],\"name\":\"channelClaimTimeout\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// MultiPartyEscrowBin is the compiled bytecode used for deploying new contracts.
const MultiPartyEscrowBin = `0x608060405234801561001057600080fd5b50604051602080611160833981016040525160038054600160a060020a031916600160a060020a03909216919091179055611110806100506000396000f3006080604052600436106100da5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630c19d0ec81146100df57806327e235e3146100ff5780632911dce7146101325780632e1a7d4d1461017957806345059a5d14610191578063a9059cbb146101ac578063b6b55f25146101d0578063baea65b5146101e8578063c94fd86714610200578063d52ea81d1461022c578063da2a5b4f1461039e578063e5949b5d146103b9578063eedcc3801461041a578063f4606f001461044d578063fc0c546a14610462575b600080fd5b3480156100eb57600080fd5b506100fd600435602435604435610493565b005b34801561010b57600080fd5b50610120600160a060020a03600435166104c2565b60408051918252519081900360200190f35b34801561013e57600080fd5b50610165600160a060020a03600435811690602435906044359060643590608435166104d4565b604080519115158252519081900360200190f35b34801561018557600080fd5b5061016560043561050e565b34801561019d57600080fd5b5061016560043560243561060a565b3480156101b857600080fd5b50610165600160a060020a0360043516602435610691565b3480156101dc57600080fd5b5061016560043561075d565b3480156101f457600080fd5b506100fd6004356108ae565b34801561020c57600080fd5b506100fd60043560243560ff6044351660643560843560a4351515610941565b34801561023857600080fd5b50604080516020600480358082013583810280860185019096528085526100fd95369593946024949385019291829185019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750506040805187358901803560208181028481018201909552818452989b9a998901989297509082019550935083925085019084908082843750949750610bfb9650505050505050565b3480156103aa57600080fd5b50610165600435602435610cf5565b3480156103c557600080fd5b506103d1600435610db3565b60408051600160a060020a0398891681529688166020880152868101959095526060860193909352608085019190915260a084015290921660c082015290519081900360e00190f35b34801561042657600080fd5b50610165600160a060020a0360043581169060243590604435906064359060843516610dfd565b34801561045957600080fd5b50610120610f8d565b34801561046e57600080fd5b50610477610f93565b60408051600160a060020a039092168252519081900360200190f35b61049d838361060a565b15156104a857600080fd5b6104b28382610cf5565b15156104bd57600080fd5b505050565b60016020526000908152604090205481565b60006104df8561075d565b15156104ea57600080fd5b6104f78686868686610dfd565b151561050257600080fd5b50600195945050505050565b3360009081526001602052604081205482111561052a57600080fd5b600354604080517fa9059cbb000000000000000000000000000000000000000000000000000000008152336004820152602481018590529051600160a060020a039092169163a9059cbb916044808201926020929091908290030181600087803b15801561059757600080fd5b505af11580156105ab573d6000803e3d6000fd5b505050506040513d60208110156105c157600080fd5b505115156105ce57600080fd5b336000908152600160205260409020546105ee908363ffffffff610fa216565b3360009081526001602081905260409091209190915592915050565b60008281526020819052604081208054600160a060020a0316331461062e57600080fd5b600581015483101561063f57600080fd5b600084815260208181526040918290206005018590558151858152915186927ff8d4e64f6b2b3db6aaf38b319e259285a48ecd0c5bc0115c9928aba297c7342092908290030190a25060019392505050565b336000908152600160205260408120548211156106ad57600080fd5b336000908152600160205260409020546106cd908363ffffffff610fa216565b3360009081526001602052604080822092909255600160a060020a038516815220546106ff908363ffffffff610fb416565b600160a060020a0384166000818152600160209081526040918290209390935580518581529051919233927f5a0155838afb0f859197785e575b9ad1afeb456c6e522b6f632ee8465941315e9281900390910190a350600192915050565b600354604080517f23b872dd000000000000000000000000000000000000000000000000000000008152336004820152306024820152604481018490529051600092600160a060020a0316916323b872dd91606480830192602092919082900301818787803b1580156107cf57600080fd5b505af11580156107e3573d6000803e3d6000fd5b505050506040513d60208110156107f957600080fd5b5051151561088e57604080517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602860248201527f556e61626c6520746f207472616e7366657220746f6b656e20746f207468652060448201527f636f6e7472616374000000000000000000000000000000000000000000000000606482015290519081900360840190fd5b336000908152600160205260409020546105ee908363ffffffff610fb416565b600081815260208190526040902054600160a060020a031633146108d157600080fd5b6000818152602081905260409020600501544310156108ef57600080fd5b6108f881610fc7565b60008181526020818152604091829020600301548251908152915183927f10522b5c8770b85fe85ef3f007840ca9fc1bfa80b980dddc847e766a303f8cda92908290030190a250565b60008681526020819052604081206003810154909190819088111561096557600080fd5b6001830154600160a060020a0316331461097e57600080fd5b610a3f308a85600401548b6040516020018085600160a060020a0316600160a060020a03166c010000000000000000000000000281526014018481526020018381526020018281526020019450505050506040516020818303038152906040526040518082805190602001908083835b60208310610a0d5780518252601f1990920191602091820191016109ee565b6001836020036101000a038019825116818451168082178552505050505050905001915050604051809103902061103a565b604080516000808252602080830180855285905260ff8c1683850152606083018b9052608083018a9052925193955060019360a08084019493601f19830193908390039091019190865af1158015610a9b573d6000803e3d6000fd5b5050604051601f1901516006850154909250600160a060020a038084169116149050610ac657600080fd5b600089815260208190526040902060030154610ae8908963ffffffff610fa216565b60008a81526020818152604080832060030193909355338252600190522054610b17908963ffffffff610fb416565b336000908152600160205260409020558315610b8e57610b3689610fc7565b6000898152602081815260408083206003015481518c81529283015281810192909252905133918b917fbe89dadc951b7d901eb74681dc1a36e63ab3f366404a072e8eede90e5615b83f9181900360600190a3610bf0565b60008981526020818152604080832060048101805460010190556003015481518c81529283019390935281810192909252905133918b917fbe89dadc951b7d901eb74681dc1a36e63ab3f366404a072e8eede90e5615b83f9181900360600190a35b505050505050505050565b8551855160009082148015610c105750818651145b8015610c1c5750818551145b8015610c285750818451145b8015610c345750818351145b1515610c3f57600080fd5b5060005b81811015610ceb57610ce38882815181101515610c5c57fe5b906020019060200201518883815181101515610c7457fe5b906020019060200201518784815181101515610c8c57fe5b906020019060200201518785815181101515610ca457fe5b906020019060200201518786815181101515610cbc57fe5b906020019060200201518b87815181101515610cd457fe5b90602001906020020151610941565b600101610c43565b5050505050505050565b33600090815260016020526040812054821115610d1157600080fd5b33600090815260016020526040902054610d31908363ffffffff610fa216565b336000908152600160209081526040808320939093558582528190522060030154610d62908363ffffffff610fb416565b60008481526020818152604091829020600301929092558051848152905185927fb0e2286f86435d8f98d9cf1c908b693792eb905dd03cd40d2b1d23a3e5311a40928290030190a250600192915050565b6000602081905290815260409020805460018201546002830154600384015460048501546005860154600690960154600160a060020a039586169694861695939492939192911687565b33600090815260016020526040812054851115610e1957600080fd5b600160a060020a0382161515610e2e57600080fd5b6040805160e08101825233808252600160a060020a038981166020808501918252848601898152606086018c815260006080880181815260a089018e81528c881660c08b019081526002805485528488528c85209b518c54908b1673ffffffffffffffffffffffffffffffffffffffff19918216178d5598516001808e018054928d16928c16929092179091559651908c0155935160038b0155905160048a0155516005890155905160069097018054979095169690931695909517909255918252919091522054610f06908663ffffffff610fa216565b33600081815260016020908152604091829020939093556002548151908152600160a060020a038681169482019490945280820189905260608101889052905186938a1692917f747506b844327a7a28a59a7c306bafc2b7b6d832d40dc3340152617dd174a372919081900360800190a45060028054600190810190915595945050505050565b60025481565b600354600160a060020a031681565b600082821115610fae57fe5b50900390565b81810182811015610fc157fe5b92915050565b60008181526020818152604080832060038101548154600160a060020a031685526001909352922054610fff9163ffffffff610fb416565b8154600160a060020a031660009081526001602081905260408220929092556003830181905560048301805490920190915560059091015550565b604080517f19457468657265756d205369676e6564204d6573736167653a0a333200000000602080830191909152603c80830185905283518084039091018152605c909201928390528151600093918291908401908083835b602083106110b25780518252601f199092019160209182019101611093565b5181516020939093036101000a60001901801990911692169190911790526040519201829003909120959450505050505600a165627a7a72305820a249ee7440cbc59e8be40ec44c946d73209e660fefa2f0081a3d0a578868d35f0029`

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
// Solidity: function channels( uint256) constant returns(sender address, recipient address, groupId bytes32, value uint256, nonce uint256, expiration uint256, signer address)
func (_MultiPartyEscrow *MultiPartyEscrowCaller) Channels(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    [32]byte
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
	Signer     common.Address
}, error) {
	ret := new(struct {
		Sender     common.Address
		Recipient  common.Address
		GroupId    [32]byte
		Value      *big.Int
		Nonce      *big.Int
		Expiration *big.Int
		Signer     common.Address
	})
	out := ret
	err := _MultiPartyEscrow.contract.Call(opts, out, "channels", arg0)
	return *ret, err
}

// Channels is a free data retrieval call binding the contract method 0xe5949b5d.
//
// Solidity: function channels( uint256) constant returns(sender address, recipient address, groupId bytes32, value uint256, nonce uint256, expiration uint256, signer address)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Channels(arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    [32]byte
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
	Signer     common.Address
}, error) {
	return _MultiPartyEscrow.Contract.Channels(&_MultiPartyEscrow.CallOpts, arg0)
}

// Channels is a free data retrieval call binding the contract method 0xe5949b5d.
//
// Solidity: function channels( uint256) constant returns(sender address, recipient address, groupId bytes32, value uint256, nonce uint256, expiration uint256, signer address)
func (_MultiPartyEscrow *MultiPartyEscrowCallerSession) Channels(arg0 *big.Int) (struct {
	Sender     common.Address
	Recipient  common.Address
	GroupId    [32]byte
	Value      *big.Int
	Nonce      *big.Int
	Expiration *big.Int
	Signer     common.Address
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

// ChannelClaim is a paid mutator transaction binding the contract method 0xc94fd867.
//
// Solidity: function channelClaim(channelId uint256, amount uint256, v uint8, r bytes32, s bytes32, isSendback bool) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) ChannelClaim(opts *bind.TransactOpts, channelId *big.Int, amount *big.Int, v uint8, r [32]byte, s [32]byte, isSendback bool) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "channelClaim", channelId, amount, v, r, s, isSendback)
}

// ChannelClaim is a paid mutator transaction binding the contract method 0xc94fd867.
//
// Solidity: function channelClaim(channelId uint256, amount uint256, v uint8, r bytes32, s bytes32, isSendback bool) returns()
func (_MultiPartyEscrow *MultiPartyEscrowSession) ChannelClaim(channelId *big.Int, amount *big.Int, v uint8, r [32]byte, s [32]byte, isSendback bool) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelClaim(&_MultiPartyEscrow.TransactOpts, channelId, amount, v, r, s, isSendback)
}

// ChannelClaim is a paid mutator transaction binding the contract method 0xc94fd867.
//
// Solidity: function channelClaim(channelId uint256, amount uint256, v uint8, r bytes32, s bytes32, isSendback bool) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) ChannelClaim(channelId *big.Int, amount *big.Int, v uint8, r [32]byte, s [32]byte, isSendback bool) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.ChannelClaim(&_MultiPartyEscrow.TransactOpts, channelId, amount, v, r, s, isSendback)
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

// DepositAndOpenChannel is a paid mutator transaction binding the contract method 0x2911dce7.
//
// Solidity: function depositAndOpenChannel(recipient address, value uint256, expiration uint256, groupId bytes32, signer address) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) DepositAndOpenChannel(opts *bind.TransactOpts, recipient common.Address, value *big.Int, expiration *big.Int, groupId [32]byte, signer common.Address) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "depositAndOpenChannel", recipient, value, expiration, groupId, signer)
}

// DepositAndOpenChannel is a paid mutator transaction binding the contract method 0x2911dce7.
//
// Solidity: function depositAndOpenChannel(recipient address, value uint256, expiration uint256, groupId bytes32, signer address) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) DepositAndOpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId [32]byte, signer common.Address) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.DepositAndOpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId, signer)
}

// DepositAndOpenChannel is a paid mutator transaction binding the contract method 0x2911dce7.
//
// Solidity: function depositAndOpenChannel(recipient address, value uint256, expiration uint256, groupId bytes32, signer address) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) DepositAndOpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId [32]byte, signer common.Address) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.DepositAndOpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId, signer)
}

// MultiChannelClaim is a paid mutator transaction binding the contract method 0xd52ea81d.
//
// Solidity: function multiChannelClaim(channelIds uint256[], amounts uint256[], isSendbacks bool[], v uint8[], r bytes32[], s bytes32[]) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) MultiChannelClaim(opts *bind.TransactOpts, channelIds []*big.Int, amounts []*big.Int, isSendbacks []bool, v []uint8, r [][32]byte, s [][32]byte) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "multiChannelClaim", channelIds, amounts, isSendbacks, v, r, s)
}

// MultiChannelClaim is a paid mutator transaction binding the contract method 0xd52ea81d.
//
// Solidity: function multiChannelClaim(channelIds uint256[], amounts uint256[], isSendbacks bool[], v uint8[], r bytes32[], s bytes32[]) returns()
func (_MultiPartyEscrow *MultiPartyEscrowSession) MultiChannelClaim(channelIds []*big.Int, amounts []*big.Int, isSendbacks []bool, v []uint8, r [][32]byte, s [][32]byte) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.MultiChannelClaim(&_MultiPartyEscrow.TransactOpts, channelIds, amounts, isSendbacks, v, r, s)
}

// MultiChannelClaim is a paid mutator transaction binding the contract method 0xd52ea81d.
//
// Solidity: function multiChannelClaim(channelIds uint256[], amounts uint256[], isSendbacks bool[], v uint8[], r bytes32[], s bytes32[]) returns()
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) MultiChannelClaim(channelIds []*big.Int, amounts []*big.Int, isSendbacks []bool, v []uint8, r [][32]byte, s [][32]byte) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.MultiChannelClaim(&_MultiPartyEscrow.TransactOpts, channelIds, amounts, isSendbacks, v, r, s)
}

// OpenChannel is a paid mutator transaction binding the contract method 0xeedcc380.
//
// Solidity: function openChannel(recipient address, value uint256, expiration uint256, groupId bytes32, signer address) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) OpenChannel(opts *bind.TransactOpts, recipient common.Address, value *big.Int, expiration *big.Int, groupId [32]byte, signer common.Address) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "openChannel", recipient, value, expiration, groupId, signer)
}

// OpenChannel is a paid mutator transaction binding the contract method 0xeedcc380.
//
// Solidity: function openChannel(recipient address, value uint256, expiration uint256, groupId bytes32, signer address) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) OpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId [32]byte, signer common.Address) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.OpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId, signer)
}

// OpenChannel is a paid mutator transaction binding the contract method 0xeedcc380.
//
// Solidity: function openChannel(recipient address, value uint256, expiration uint256, groupId bytes32, signer address) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) OpenChannel(recipient common.Address, value *big.Int, expiration *big.Int, groupId [32]byte, signer common.Address) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.OpenChannel(&_MultiPartyEscrow.TransactOpts, recipient, value, expiration, groupId, signer)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(receiver address, value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactor) Transfer(opts *bind.TransactOpts, receiver common.Address, value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.contract.Transact(opts, "transfer", receiver, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(receiver address, value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowSession) Transfer(receiver common.Address, value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.Transfer(&_MultiPartyEscrow.TransactOpts, receiver, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(receiver address, value uint256) returns(bool)
func (_MultiPartyEscrow *MultiPartyEscrowTransactorSession) Transfer(receiver common.Address, value *big.Int) (*types.Transaction, error) {
	return _MultiPartyEscrow.Contract.Transfer(&_MultiPartyEscrow.TransactOpts, receiver, value)
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

// MultiPartyEscrowChannelAddFundsIterator is returned from FilterChannelAddFunds and is used to iterate over the raw logs and unpacked data for ChannelAddFunds events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelAddFundsIterator struct {
	Event *MultiPartyEscrowChannelAddFunds // Event containing the contract specifics and raw log

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
func (it *MultiPartyEscrowChannelAddFundsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowChannelAddFunds)
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
		it.Event = new(MultiPartyEscrowChannelAddFunds)
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
func (it *MultiPartyEscrowChannelAddFundsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowChannelAddFundsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowChannelAddFunds represents a ChannelAddFunds event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelAddFunds struct {
	ChannelId       *big.Int
	AdditionalFunds *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterChannelAddFunds is a free log retrieval operation binding the contract event 0xb0e2286f86435d8f98d9cf1c908b693792eb905dd03cd40d2b1d23a3e5311a40.
//
// Solidity: e ChannelAddFunds(channelId indexed uint256, additionalFunds uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterChannelAddFunds(opts *bind.FilterOpts, channelId []*big.Int) (*MultiPartyEscrowChannelAddFundsIterator, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "ChannelAddFunds", channelIdRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowChannelAddFundsIterator{contract: _MultiPartyEscrow.contract, event: "ChannelAddFunds", logs: logs, sub: sub}, nil
}

// WatchChannelAddFunds is a free log subscription operation binding the contract event 0xb0e2286f86435d8f98d9cf1c908b693792eb905dd03cd40d2b1d23a3e5311a40.
//
// Solidity: e ChannelAddFunds(channelId indexed uint256, additionalFunds uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchChannelAddFunds(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowChannelAddFunds, channelId []*big.Int) (event.Subscription, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "ChannelAddFunds", channelIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowChannelAddFunds)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "ChannelAddFunds", log); err != nil {
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

// MultiPartyEscrowChannelClaimIterator is returned from FilterChannelClaim and is used to iterate over the raw logs and unpacked data for ChannelClaim events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelClaimIterator struct {
	Event *MultiPartyEscrowChannelClaim // Event containing the contract specifics and raw log

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
func (it *MultiPartyEscrowChannelClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowChannelClaim)
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
		it.Event = new(MultiPartyEscrowChannelClaim)
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
func (it *MultiPartyEscrowChannelClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowChannelClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowChannelClaim represents a ChannelClaim event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelClaim struct {
	ChannelId      *big.Int
	Recipient      common.Address
	ClaimAmount    *big.Int
	SendBackAmount *big.Int
	KeepAmpount    *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChannelClaim is a free log retrieval operation binding the contract event 0xbe89dadc951b7d901eb74681dc1a36e63ab3f366404a072e8eede90e5615b83f.
//
// Solidity: e ChannelClaim(channelId indexed uint256, recipient indexed address, claimAmount uint256, sendBackAmount uint256, keepAmpount uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterChannelClaim(opts *bind.FilterOpts, channelId []*big.Int, recipient []common.Address) (*MultiPartyEscrowChannelClaimIterator, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "ChannelClaim", channelIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowChannelClaimIterator{contract: _MultiPartyEscrow.contract, event: "ChannelClaim", logs: logs, sub: sub}, nil
}

// WatchChannelClaim is a free log subscription operation binding the contract event 0xbe89dadc951b7d901eb74681dc1a36e63ab3f366404a072e8eede90e5615b83f.
//
// Solidity: e ChannelClaim(channelId indexed uint256, recipient indexed address, claimAmount uint256, sendBackAmount uint256, keepAmpount uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchChannelClaim(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowChannelClaim, channelId []*big.Int, recipient []common.Address) (event.Subscription, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "ChannelClaim", channelIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowChannelClaim)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "ChannelClaim", log); err != nil {
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

// MultiPartyEscrowChannelExtendIterator is returned from FilterChannelExtend and is used to iterate over the raw logs and unpacked data for ChannelExtend events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelExtendIterator struct {
	Event *MultiPartyEscrowChannelExtend // Event containing the contract specifics and raw log

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
func (it *MultiPartyEscrowChannelExtendIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowChannelExtend)
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
		it.Event = new(MultiPartyEscrowChannelExtend)
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
func (it *MultiPartyEscrowChannelExtendIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowChannelExtendIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowChannelExtend represents a ChannelExtend event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelExtend struct {
	ChannelId     *big.Int
	NewExpiration *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterChannelExtend is a free log retrieval operation binding the contract event 0xf8d4e64f6b2b3db6aaf38b319e259285a48ecd0c5bc0115c9928aba297c73420.
//
// Solidity: e ChannelExtend(channelId indexed uint256, newExpiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterChannelExtend(opts *bind.FilterOpts, channelId []*big.Int) (*MultiPartyEscrowChannelExtendIterator, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "ChannelExtend", channelIdRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowChannelExtendIterator{contract: _MultiPartyEscrow.contract, event: "ChannelExtend", logs: logs, sub: sub}, nil
}

// WatchChannelExtend is a free log subscription operation binding the contract event 0xf8d4e64f6b2b3db6aaf38b319e259285a48ecd0c5bc0115c9928aba297c73420.
//
// Solidity: e ChannelExtend(channelId indexed uint256, newExpiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchChannelExtend(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowChannelExtend, channelId []*big.Int) (event.Subscription, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "ChannelExtend", channelIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowChannelExtend)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "ChannelExtend", log); err != nil {
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

// MultiPartyEscrowChannelOpenIterator is returned from FilterChannelOpen and is used to iterate over the raw logs and unpacked data for ChannelOpen events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelOpenIterator struct {
	Event *MultiPartyEscrowChannelOpen // Event containing the contract specifics and raw log

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
func (it *MultiPartyEscrowChannelOpenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowChannelOpen)
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
		it.Event = new(MultiPartyEscrowChannelOpen)
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
func (it *MultiPartyEscrowChannelOpenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowChannelOpenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowChannelOpen represents a ChannelOpen event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelOpen struct {
	ChannelId  *big.Int
	Sender     common.Address
	Recipient  common.Address
	GroupId    [32]byte
	Signer     common.Address
	Amount     *big.Int
	Expiration *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterChannelOpen is a free log retrieval operation binding the contract event 0x747506b844327a7a28a59a7c306bafc2b7b6d832d40dc3340152617dd174a372.
//
// Solidity: e ChannelOpen(channelId uint256, sender indexed address, recipient indexed address, groupId indexed bytes32, signer address, amount uint256, expiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterChannelOpen(opts *bind.FilterOpts, sender []common.Address, recipient []common.Address, groupId [][32]byte) (*MultiPartyEscrowChannelOpenIterator, error) {

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

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "ChannelOpen", senderRule, recipientRule, groupIdRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowChannelOpenIterator{contract: _MultiPartyEscrow.contract, event: "ChannelOpen", logs: logs, sub: sub}, nil
}

// WatchChannelOpen is a free log subscription operation binding the contract event 0x747506b844327a7a28a59a7c306bafc2b7b6d832d40dc3340152617dd174a372.
//
// Solidity: e ChannelOpen(channelId uint256, sender indexed address, recipient indexed address, groupId indexed bytes32, signer address, amount uint256, expiration uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchChannelOpen(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowChannelOpen, sender []common.Address, recipient []common.Address, groupId [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "ChannelOpen", senderRule, recipientRule, groupIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowChannelOpen)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "ChannelOpen", log); err != nil {
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

// MultiPartyEscrowChannelSenderClaimIterator is returned from FilterChannelSenderClaim and is used to iterate over the raw logs and unpacked data for ChannelSenderClaim events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelSenderClaimIterator struct {
	Event *MultiPartyEscrowChannelSenderClaim // Event containing the contract specifics and raw log

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
func (it *MultiPartyEscrowChannelSenderClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowChannelSenderClaim)
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
		it.Event = new(MultiPartyEscrowChannelSenderClaim)
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
func (it *MultiPartyEscrowChannelSenderClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowChannelSenderClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowChannelSenderClaim represents a ChannelSenderClaim event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowChannelSenderClaim struct {
	ChannelId   *big.Int
	ClaimAmount *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterChannelSenderClaim is a free log retrieval operation binding the contract event 0x10522b5c8770b85fe85ef3f007840ca9fc1bfa80b980dddc847e766a303f8cda.
//
// Solidity: e ChannelSenderClaim(channelId indexed uint256, claimAmount uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterChannelSenderClaim(opts *bind.FilterOpts, channelId []*big.Int) (*MultiPartyEscrowChannelSenderClaimIterator, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "ChannelSenderClaim", channelIdRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowChannelSenderClaimIterator{contract: _MultiPartyEscrow.contract, event: "ChannelSenderClaim", logs: logs, sub: sub}, nil
}

// WatchChannelSenderClaim is a free log subscription operation binding the contract event 0x10522b5c8770b85fe85ef3f007840ca9fc1bfa80b980dddc847e766a303f8cda.
//
// Solidity: e ChannelSenderClaim(channelId indexed uint256, claimAmount uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchChannelSenderClaim(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowChannelSenderClaim, channelId []*big.Int) (event.Subscription, error) {

	var channelIdRule []interface{}
	for _, channelIdItem := range channelId {
		channelIdRule = append(channelIdRule, channelIdItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "ChannelSenderClaim", channelIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowChannelSenderClaim)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "ChannelSenderClaim", log); err != nil {
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

// MultiPartyEscrowTransferFundsIterator is returned from FilterTransferFunds and is used to iterate over the raw logs and unpacked data for TransferFunds events raised by the MultiPartyEscrow contract.
type MultiPartyEscrowTransferFundsIterator struct {
	Event *MultiPartyEscrowTransferFunds // Event containing the contract specifics and raw log

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
func (it *MultiPartyEscrowTransferFundsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiPartyEscrowTransferFunds)
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
		it.Event = new(MultiPartyEscrowTransferFunds)
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
func (it *MultiPartyEscrowTransferFundsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiPartyEscrowTransferFundsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiPartyEscrowTransferFunds represents a TransferFunds event raised by the MultiPartyEscrow contract.
type MultiPartyEscrowTransferFunds struct {
	Sender   common.Address
	Receiver common.Address
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterTransferFunds is a free log retrieval operation binding the contract event 0x5a0155838afb0f859197785e575b9ad1afeb456c6e522b6f632ee8465941315e.
//
// Solidity: e TransferFunds(sender indexed address, receiver indexed address, amount uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) FilterTransferFunds(opts *bind.FilterOpts, sender []common.Address, receiver []common.Address) (*MultiPartyEscrowTransferFundsIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.FilterLogs(opts, "TransferFunds", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return &MultiPartyEscrowTransferFundsIterator{contract: _MultiPartyEscrow.contract, event: "TransferFunds", logs: logs, sub: sub}, nil
}

// WatchTransferFunds is a free log subscription operation binding the contract event 0x5a0155838afb0f859197785e575b9ad1afeb456c6e522b6f632ee8465941315e.
//
// Solidity: e TransferFunds(sender indexed address, receiver indexed address, amount uint256)
func (_MultiPartyEscrow *MultiPartyEscrowFilterer) WatchTransferFunds(opts *bind.WatchOpts, sink chan<- *MultiPartyEscrowTransferFunds, sender []common.Address, receiver []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _MultiPartyEscrow.contract.WatchLogs(opts, "TransferFunds", senderRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiPartyEscrowTransferFunds)
				if err := _MultiPartyEscrow.contract.UnpackLog(event, "TransferFunds", log); err != nil {
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
