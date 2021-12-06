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

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// SingularityNetTokenABI is the input ABI used to generate the binding from.
const SingularityNetTokenABI = "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"MINTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"PAUSER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burnFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// SingularityNetTokenBin is the compiled bytecode used for deploying new contracts.
const SingularityNetTokenBin = `"0x60806040523480156200001157600080fd5b5060405162001d9e38038062001d9e833981810160405260408110156200003757600080fd5b81019080805160405193929190846401000000008211156200005857600080fd5b9083019060208201858111156200006e57600080fd5b82516401000000008111828201881017156200008957600080fd5b82525081516020918201929091019080838360005b83811015620000b85781810151838201526020016200009e565b50505050905090810190601f168015620000e65780820380516001836020036101000a031916815260200191505b50604052602001805160405193929190846401000000008211156200010a57600080fd5b9083019060208201858111156200012057600080fd5b82516401000000008111828201881017156200013b57600080fd5b82525081516020918201929091019080838360005b838110156200016a57818101518382015260200162000150565b50505050905090810190601f168015620001985780820380516001836020036101000a031916815260200191505b50604052505082518391508290620001b8906004906020850190620003e9565b508051620001ce906005906020840190620003e9565b50506006805461ff001960ff19909116601217169055506200020d6000620001fe6001600160e01b036200029a16565b6001600160e01b036200029f16565b604080516a4d494e5445525f524f4c4560a81b8152905190819003600b0190206200024590620001fe6001600160e01b036200029a16565b604080516a5041555345525f524f4c4560a81b8152905190819003600b0190206200027d90620001fe6001600160e01b036200029a16565b6200029260086001600160e01b03620002b816565b50506200048b565b335b90565b620002b482826001600160e01b03620002ce16565b5050565b6006805460ff191660ff92909216919091179055565b600082815260208181526040909120620002f39183906200139a62000350821b17901c565b15620002b4576200030c6001600160e01b036200029a16565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b600062000370836001600160a01b0384166001600160e01b036200037916565b90505b92915050565b60006200039083836001600160e01b03620003d116565b620003c85750815460018181018455600084815260208082209093018490558454848252828601909352604090209190915562000373565b50600062000373565b60009081526001919091016020526040902054151590565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106200042c57805160ff19168380011785556200045c565b828001600101855582156200045c579182015b828111156200045c5782518255916020019190600101906200043f565b506200046a9291506200046e565b5090565b6200029c91905b808211156200046a576000815560010162000475565b611903806200049b6000396000f3fe608060405234801561001057600080fd5b50600436106101a95760003560e01c806370a08231116100f9578063a457c2d711610097578063d539139311610071578063d53913931461051f578063d547741f14610527578063dd62ed3e14610553578063e63ab1e914610581576101a9565b8063a457c2d7146104aa578063a9059cbb146104d6578063ca15c87314610502576101a9565b80639010d07c116100d35780639010d07c1461042f57806391d148541461046e57806395d89b411461049a578063a217fddf146104a2576101a9565b806370a08231146103d557806379cc6790146103fb5780638456cb5914610427576101a9565b8063313ce567116101665780633f4ba83a116101405780633f4ba83a1461037c57806340c10f191461038457806342966c68146103b05780635c975abb146103cd576101a9565b8063313ce5671461030657806336568abe146103245780633950935114610350576101a9565b806306fdde03146101ae578063095ea7b31461022b57806318160ddd1461026b57806323b872dd14610285578063248a9ca3146102bb5780632f2ff15d146102d8575b600080fd5b6101b6610589565b6040805160208082528351818301528351919283929083019185019080838360005b838110156101f05781810151838201526020016101d8565b50505050905090810190601f16801561021d5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b6102576004803603604081101561024157600080fd5b506001600160a01b03813516906020013561061f565b604080519115158252519081900360200190f35b61027361063d565b60408051918252519081900360200190f35b6102576004803603606081101561029b57600080fd5b506001600160a01b03813581169160208101359091169060400135610643565b610273600480360360208110156102d157600080fd5b50356106d0565b610304600480360360408110156102ee57600080fd5b50803590602001356001600160a01b03166106e5565b005b61030e610751565b6040805160ff9092168252519081900360200190f35b6103046004803603604081101561033a57600080fd5b50803590602001356001600160a01b031661075a565b6102576004803603604081101561036657600080fd5b506001600160a01b0381351690602001356107bb565b61030461080f565b6103046004803603604081101561039a57600080fd5b506001600160a01b038135169060200135610880565b610304600480360360208110156103c657600080fd5b50356108f1565b610257610905565b610273600480360360208110156103eb57600080fd5b50356001600160a01b0316610913565b6103046004803603604081101561041157600080fd5b506001600160a01b03813516906020013561092e565b61030461098e565b6104526004803603604081101561044557600080fd5b50803590602001356109fd565b604080516001600160a01b039092168252519081900360200190f35b6102576004803603604081101561048457600080fd5b50803590602001356001600160a01b0316610a22565b6101b6610a40565b610273610aa1565b610257600480360360408110156104c057600080fd5b506001600160a01b038135169060200135610aa6565b610257600480360360408110156104ec57600080fd5b506001600160a01b038135169060200135610b14565b6102736004803603602081101561051857600080fd5b5035610b28565b610273610b3f565b6103046004803603604081101561053d57600080fd5b50803590602001356001600160a01b0316610b62565b6102736004803603604081101561056957600080fd5b506001600160a01b0381358116916020013516610bbb565b610273610be6565b60048054604080516020601f60026000196101006001881615020190951694909404938401819004810282018101909252828152606093909290918301828280156106155780601f106105ea57610100808354040283529160200191610615565b820191906000526020600020905b8154815290600101906020018083116105f857829003601f168201915b5050505050905090565b600061063361062c610c09565b8484610c0d565b5060015b92915050565b60035490565b6000610650848484610cf9565b6106c68461065c610c09565b6106c18560405180606001604052806028815260200161172d602891396001600160a01b038a1660009081526002602052604081209061069a610c09565b6001600160a01b03168152602081019190915260400160002054919063ffffffff610e6216565b610c0d565b5060019392505050565b60009081526020819052604090206002015490565b60008281526020819052604090206002015461070890610703610c09565b610a22565b6107435760405162461bcd60e51b815260040180806020018281038252602f81526020018061162b602f913960400191505060405180910390fd5b61074d8282610ef9565b5050565b60065460ff1690565b610762610c09565b6001600160a01b0316816001600160a01b0316146107b15760405162461bcd60e51b815260040180806020018281038252602f815260200180611875602f913960400191505060405180910390fd5b61074d8282610f68565b60006106336107c8610c09565b846106c185600260006107d9610c09565b6001600160a01b03908116825260208083019390935260409182016000908120918c16815292529020549063ffffffff610fd716565b604080516a5041555345525f524f4c4560a81b8152905190819003600b01902061083b90610703610c09565b6108765760405162461bcd60e51b815260040180806020018281038252603981526020018061167c6039913960400191505060405180910390fd5b61087e611031565b565b604080516a4d494e5445525f524f4c4560a81b8152905190819003600b0190206108ac90610703610c09565b6108e75760405162461bcd60e51b81526004018080602001828103825260368152602001806117556036913960400191505060405180910390fd5b61074d82826110d5565b6109026108fc610c09565b826111d3565b50565b600654610100900460ff1690565b6001600160a01b031660009081526001602052604090205490565b600061096b8260405180606001604052806024815260200161178b6024913961095e86610959610c09565b610bbb565b919063ffffffff610e6216565b905061097f83610979610c09565b83610c0d565b61098983836111d3565b505050565b604080516a5041555345525f524f4c4560a81b8152905190819003600b0190206109ba90610703610c09565b6109f55760405162461bcd60e51b81526004018080602001828103825260378152602001806118196037913960400191505060405180910390fd5b61087e6112db565b6000828152602081905260408120610a1b908363ffffffff61136316565b9392505050565b6000828152602081905260408120610a1b908363ffffffff61136f16565b60058054604080516020601f60026000196101006001881615020190951694909404938401819004810282018101909252828152606093909290918301828280156106155780601f106105ea57610100808354040283529160200191610615565b600081565b6000610633610ab3610c09565b846106c1856040518060600160405280602581526020016118506025913960026000610add610c09565b6001600160a01b03908116825260208083019390935260409182016000908120918d1681529252902054919063ffffffff610e6216565b6000610633610b21610c09565b8484610cf9565b600081815260208190526040812061063790611384565b604080516a4d494e5445525f524f4c4560a81b8152905190819003600b01902081565b600082815260208190526040902060020154610b8090610703610c09565b6107b15760405162461bcd60e51b81526004018080602001828103825260308152602001806116fd6030913960400191505060405180910390fd5b6001600160a01b03918216600090815260026020908152604080832093909416825291909152205490565b604080516a5041555345525f524f4c4560a81b8152905190819003600b01902081565b3390565b6001600160a01b038316610c525760405162461bcd60e51b81526004018080602001828103825260248152602001806117f56024913960400191505060405180910390fd5b6001600160a01b038216610c975760405162461bcd60e51b81526004018080602001828103825260228152602001806116b56022913960400191505060405180910390fd5b6001600160a01b03808416600081815260026020908152604080832094871680845294825291829020859055815185815291517f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9259281900390910190a3505050565b6001600160a01b038316610d3e5760405162461bcd60e51b81526004018080602001828103825260258152602001806117d06025913960400191505060405180910390fd5b6001600160a01b038216610d835760405162461bcd60e51b81526004018080602001828103825260238152602001806116086023913960400191505060405180910390fd5b610d8e83838361138f565b610dd1816040518060600160405280602681526020016116d7602691396001600160a01b038616600090815260016020526040902054919063ffffffff610e6216565b6001600160a01b038085166000908152600160205260408082209390935590841681522054610e06908263ffffffff610fd716565b6001600160a01b0380841660008181526001602090815260409182902094909455805185815290519193928716927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef92918290030190a3505050565b60008184841115610ef15760405162461bcd60e51b81526004018080602001828103825283818151815260200191508051906020019080838360005b83811015610eb6578181015183820152602001610e9e565b50505050905090810190601f168015610ee35780820380516001836020036101000a031916815260200191505b509250505060405180910390fd5b505050900390565b6000828152602081905260409020610f17908263ffffffff61139a16565b1561074d57610f24610c09565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b6000828152602081905260409020610f86908263ffffffff6113af16565b1561074d57610f93610c09565b6001600160a01b0316816001600160a01b0316837ff6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b60405160405180910390a45050565b600082820183811015610a1b576040805162461bcd60e51b815260206004820152601b60248201527f536166654d6174683a206164646974696f6e206f766572666c6f770000000000604482015290519081900360640190fd5b600654610100900460ff16611084576040805162461bcd60e51b815260206004820152601460248201527314185d5cd8589b194e881b9bdd081c185d5cd95960621b604482015290519081900360640190fd5b6006805461ff00191690557f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa6110b8610c09565b604080516001600160a01b039092168252519081900360200190a1565b6001600160a01b038216611130576040805162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015290519081900360640190fd5b61113c6000838361138f565b60035461114f908263ffffffff610fd716565b6003556001600160a01b03821660009081526001602052604090205461117b908263ffffffff610fd716565b6001600160a01b03831660008181526001602090815260408083209490945583518581529351929391927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9281900390910190a35050565b6001600160a01b0382166112185760405162461bcd60e51b81526004018080602001828103825260218152602001806117af6021913960400191505060405180910390fd5b6112248260008361138f565b6112678160405180606001604052806022815260200161165a602291396001600160a01b038516600090815260016020526040902054919063ffffffff610e6216565b6001600160a01b038316600090815260016020526040902055600354611293908263ffffffff6113c416565b6003556040805182815290516000916001600160a01b038516917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9181900360200190a35050565b600654610100900460ff161561132b576040805162461bcd60e51b815260206004820152601060248201526f14185d5cd8589b194e881c185d5cd95960821b604482015290519081900360640190fd5b6006805461ff0019166101001790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a2586110b8610c09565b6000610a1b8383611406565b6000610a1b836001600160a01b03841661146a565b600061063782611482565b610989838383611486565b6000610a1b836001600160a01b0384166114d5565b6000610a1b836001600160a01b03841661151f565b6000610a1b83836040518060400160405280601e81526020017f536166654d6174683a207375627472616374696f6e206f766572666c6f770000815250610e62565b815460009082106114485760405162461bcd60e51b81526004018080602001828103825260228152602001806115e66022913960400191505060405180910390fd5b82600001828154811061145757fe5b9060005260206000200154905092915050565b60009081526001919091016020526040902054151590565b5490565b611491838383610989565b611499610905565b156109895760405162461bcd60e51b815260040180806020018281038252602a8152602001806118a4602a913960400191505060405180910390fd5b60006114e1838361146a565b61151757508154600181810184556000848152602080822090930184905584548482528286019093526040902091909155610637565b506000610637565b600081815260018301602052604081205480156115db578354600019808301919081019060009087908390811061155257fe5b906000526020600020015490508087600001848154811061156f57fe5b60009182526020808320909101929092558281526001898101909252604090209084019055865487908061159f57fe5b60019003818190600052602060002001600090559055866001016000878152602001908152602001600020600090556001945050505050610637565b600091505061063756fe456e756d657261626c655365743a20696e646578206f7574206f6620626f756e647345524332303a207472616e7366657220746f20746865207a65726f2061646472657373416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e2061646d696e20746f206772616e7445524332303a206275726e20616d6f756e7420657863656564732062616c616e636545524332305072657365744d696e7465725061757365723a206d75737420686176652070617573657220726f6c6520746f20756e706175736545524332303a20617070726f766520746f20746865207a65726f206164647265737345524332303a207472616e7366657220616d6f756e7420657863656564732062616c616e6365416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e2061646d696e20746f207265766f6b6545524332303a207472616e7366657220616d6f756e74206578636565647320616c6c6f77616e636545524332305072657365744d696e7465725061757365723a206d7573742068617665206d696e74657220726f6c6520746f206d696e7445524332303a206275726e20616d6f756e74206578636565647320616c6c6f77616e636545524332303a206275726e2066726f6d20746865207a65726f206164647265737345524332303a207472616e736665722066726f6d20746865207a65726f206164647265737345524332303a20617070726f76652066726f6d20746865207a65726f206164647265737345524332305072657365744d696e7465725061757365723a206d75737420686176652070617573657220726f6c6520746f20706175736545524332303a2064656372656173656420616c6c6f77616e63652062656c6f77207a65726f416363657373436f6e74726f6c3a2063616e206f6e6c792072656e6f756e636520726f6c657320666f722073656c6645524332305061757361626c653a20746f6b656e207472616e73666572207768696c6520706175736564a26469706673582212206e15dde2dbd9513414f58305bc1fa8bee122abc8b78c5ff4bbe6dc2555054d4f64736f6c63430006020033"`

// DeploySingularityNetToken deploys a new Ethereum contract, binding an instance of SingularityNetToken to it.
func DeploySingularityNetToken(auth *bind.TransactOpts, backend bind.ContractBackend, name string, symbol string) (common.Address, *types.Transaction, *SingularityNetToken, error) {
	parsed, err := abi.JSON(strings.NewReader(SingularityNetTokenABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SingularityNetTokenBin), backend, name, symbol)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SingularityNetToken{SingularityNetTokenCaller: SingularityNetTokenCaller{contract: contract}, SingularityNetTokenTransactor: SingularityNetTokenTransactor{contract: contract}, SingularityNetTokenFilterer: SingularityNetTokenFilterer{contract: contract}}, nil
}

// SingularityNetToken is an auto generated Go binding around an Ethereum contract.
type SingularityNetToken struct {
	SingularityNetTokenCaller     // Read-only binding to the contract
	SingularityNetTokenTransactor // Write-only binding to the contract
	SingularityNetTokenFilterer   // Log filterer for contract events
}

// SingularityNetTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type SingularityNetTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingularityNetTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SingularityNetTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingularityNetTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SingularityNetTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SingularityNetTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SingularityNetTokenSession struct {
	Contract     *SingularityNetToken // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// SingularityNetTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SingularityNetTokenCallerSession struct {
	Contract *SingularityNetTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// SingularityNetTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SingularityNetTokenTransactorSession struct {
	Contract     *SingularityNetTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// SingularityNetTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type SingularityNetTokenRaw struct {
	Contract *SingularityNetToken // Generic contract binding to access the raw methods on
}

// SingularityNetTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SingularityNetTokenCallerRaw struct {
	Contract *SingularityNetTokenCaller // Generic read-only contract binding to access the raw methods on
}

// SingularityNetTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SingularityNetTokenTransactorRaw struct {
	Contract *SingularityNetTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSingularityNetToken creates a new instance of SingularityNetToken, bound to a specific deployed contract.
func NewSingularityNetToken(address common.Address, backend bind.ContractBackend) (*SingularityNetToken, error) {
	contract, err := bindSingularityNetToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SingularityNetToken{SingularityNetTokenCaller: SingularityNetTokenCaller{contract: contract}, SingularityNetTokenTransactor: SingularityNetTokenTransactor{contract: contract}, SingularityNetTokenFilterer: SingularityNetTokenFilterer{contract: contract}}, nil
}

// NewSingularityNetTokenCaller creates a new read-only instance of SingularityNetToken, bound to a specific deployed contract.
func NewSingularityNetTokenCaller(address common.Address, caller bind.ContractCaller) (*SingularityNetTokenCaller, error) {
	contract, err := bindSingularityNetToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenCaller{contract: contract}, nil
}

// NewSingularityNetTokenTransactor creates a new write-only instance of SingularityNetToken, bound to a specific deployed contract.
func NewSingularityNetTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*SingularityNetTokenTransactor, error) {
	contract, err := bindSingularityNetToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenTransactor{contract: contract}, nil
}

// NewSingularityNetTokenFilterer creates a new log filterer instance of SingularityNetToken, bound to a specific deployed contract.
func NewSingularityNetTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*SingularityNetTokenFilterer, error) {
	contract, err := bindSingularityNetToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenFilterer{contract: contract}, nil
}

// bindSingularityNetToken binds a generic wrapper to an already deployed contract.
func bindSingularityNetToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SingularityNetTokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SingularityNetToken *SingularityNetTokenRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SingularityNetToken.Contract.SingularityNetTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SingularityNetToken *SingularityNetTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.SingularityNetTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SingularityNetToken *SingularityNetTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.SingularityNetTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SingularityNetToken *SingularityNetTokenCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SingularityNetToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SingularityNetToken *SingularityNetTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SingularityNetToken *SingularityNetTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "DEFAULT_ADMIN_ROLE")
	return *ret0, err
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _SingularityNetToken.Contract.DEFAULTADMINROLE(&_SingularityNetToken.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _SingularityNetToken.Contract.DEFAULTADMINROLE(&_SingularityNetToken.CallOpts)
}

// MINTERROLE is a free data retrieval call binding the contract method 0xd5391393.
//
// Solidity: function MINTER_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCaller) MINTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "MINTER_ROLE")
	return *ret0, err
}

// MINTERROLE is a free data retrieval call binding the contract method 0xd5391393.
//
// Solidity: function MINTER_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenSession) MINTERROLE() ([32]byte, error) {
	return _SingularityNetToken.Contract.MINTERROLE(&_SingularityNetToken.CallOpts)
}

// MINTERROLE is a free data retrieval call binding the contract method 0xd5391393.
//
// Solidity: function MINTER_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCallerSession) MINTERROLE() ([32]byte, error) {
	return _SingularityNetToken.Contract.MINTERROLE(&_SingularityNetToken.CallOpts)
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCaller) PAUSERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "PAUSER_ROLE")
	return *ret0, err
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenSession) PAUSERROLE() ([32]byte, error) {
	return _SingularityNetToken.Contract.PAUSERROLE(&_SingularityNetToken.CallOpts)
}

// PAUSERROLE is a free data retrieval call binding the contract method 0xe63ab1e9.
//
// Solidity: function PAUSER_ROLE() constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCallerSession) PAUSERROLE() ([32]byte, error) {
	return _SingularityNetToken.Contract.PAUSERROLE(&_SingularityNetToken.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "allowance", owner, spender)
	return *ret0, err
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _SingularityNetToken.Contract.Allowance(&_SingularityNetToken.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _SingularityNetToken.Contract.Allowance(&_SingularityNetToken.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "balanceOf", account)
	return *ret0, err
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _SingularityNetToken.Contract.BalanceOf(&_SingularityNetToken.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _SingularityNetToken.Contract.BalanceOf(&_SingularityNetToken.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() constant returns(uint8)
func (_SingularityNetToken *SingularityNetTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var (
		ret0 = new(uint8)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "decimals")
	return *ret0, err
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() constant returns(uint8)
func (_SingularityNetToken *SingularityNetTokenSession) Decimals() (uint8, error) {
	return _SingularityNetToken.Contract.Decimals(&_SingularityNetToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() constant returns(uint8)
func (_SingularityNetToken *SingularityNetTokenCallerSession) Decimals() (uint8, error) {
	return _SingularityNetToken.Contract.Decimals(&_SingularityNetToken.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "getRoleAdmin", role)
	return *ret0, err
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _SingularityNetToken.Contract.GetRoleAdmin(&_SingularityNetToken.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) constant returns(bytes32)
func (_SingularityNetToken *SingularityNetTokenCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _SingularityNetToken.Contract.GetRoleAdmin(&_SingularityNetToken.CallOpts, role)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) constant returns(address)
func (_SingularityNetToken *SingularityNetTokenCaller) GetRoleMember(opts *bind.CallOpts, role [32]byte, index *big.Int) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "getRoleMember", role, index)
	return *ret0, err
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) constant returns(address)
func (_SingularityNetToken *SingularityNetTokenSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _SingularityNetToken.Contract.GetRoleMember(&_SingularityNetToken.CallOpts, role, index)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) constant returns(address)
func (_SingularityNetToken *SingularityNetTokenCallerSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _SingularityNetToken.Contract.GetRoleMember(&_SingularityNetToken.CallOpts, role, index)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCaller) GetRoleMemberCount(opts *bind.CallOpts, role [32]byte) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "getRoleMemberCount", role)
	return *ret0, err
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _SingularityNetToken.Contract.GetRoleMemberCount(&_SingularityNetToken.CallOpts, role)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCallerSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _SingularityNetToken.Contract.GetRoleMemberCount(&_SingularityNetToken.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) constant returns(bool)
func (_SingularityNetToken *SingularityNetTokenCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "hasRole", role, account)
	return *ret0, err
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) constant returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _SingularityNetToken.Contract.HasRole(&_SingularityNetToken.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) constant returns(bool)
func (_SingularityNetToken *SingularityNetTokenCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _SingularityNetToken.Contract.HasRole(&_SingularityNetToken.CallOpts, role, account)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() constant returns(string)
func (_SingularityNetToken *SingularityNetTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "name")
	return *ret0, err
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() constant returns(string)
func (_SingularityNetToken *SingularityNetTokenSession) Name() (string, error) {
	return _SingularityNetToken.Contract.Name(&_SingularityNetToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() constant returns(string)
func (_SingularityNetToken *SingularityNetTokenCallerSession) Name() (string, error) {
	return _SingularityNetToken.Contract.Name(&_SingularityNetToken.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() constant returns(bool)
func (_SingularityNetToken *SingularityNetTokenCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "paused")
	return *ret0, err
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() constant returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) Paused() (bool, error) {
	return _SingularityNetToken.Contract.Paused(&_SingularityNetToken.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() constant returns(bool)
func (_SingularityNetToken *SingularityNetTokenCallerSession) Paused() (bool, error) {
	return _SingularityNetToken.Contract.Paused(&_SingularityNetToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() constant returns(string)
func (_SingularityNetToken *SingularityNetTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "symbol")
	return *ret0, err
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() constant returns(string)
func (_SingularityNetToken *SingularityNetTokenSession) Symbol() (string, error) {
	return _SingularityNetToken.Contract.Symbol(&_SingularityNetToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() constant returns(string)
func (_SingularityNetToken *SingularityNetTokenCallerSession) Symbol() (string, error) {
	return _SingularityNetToken.Contract.Symbol(&_SingularityNetToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SingularityNetToken.contract.Call(opts, out, "totalSupply")
	return *ret0, err
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenSession) TotalSupply() (*big.Int, error) {
	return _SingularityNetToken.Contract.TotalSupply(&_SingularityNetToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() constant returns(uint256)
func (_SingularityNetToken *SingularityNetTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _SingularityNetToken.Contract.TotalSupply(&_SingularityNetToken.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Approve(&_SingularityNetToken.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Approve(&_SingularityNetToken.TransactOpts, spender, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) Burn(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "burn", amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenSession) Burn(amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Burn(&_SingularityNetToken.TransactOpts, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) Burn(amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Burn(&_SingularityNetToken.TransactOpts, amount)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) BurnFrom(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "burnFrom", account, amount)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenSession) BurnFrom(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.BurnFrom(&_SingularityNetToken.TransactOpts, account, amount)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) BurnFrom(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.BurnFrom(&_SingularityNetToken.TransactOpts, account, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.DecreaseAllowance(&_SingularityNetToken.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.DecreaseAllowance(&_SingularityNetToken.TransactOpts, spender, subtractedValue)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.GrantRole(&_SingularityNetToken.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.GrantRole(&_SingularityNetToken.TransactOpts, role, account)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.IncreaseAllowance(&_SingularityNetToken.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.IncreaseAllowance(&_SingularityNetToken.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) Mint(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "mint", to, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenSession) Mint(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Mint(&_SingularityNetToken.TransactOpts, to, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) Mint(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Mint(&_SingularityNetToken.TransactOpts, to, amount)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_SingularityNetToken *SingularityNetTokenSession) Pause() (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Pause(&_SingularityNetToken.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) Pause() (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Pause(&_SingularityNetToken.TransactOpts)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.RenounceRole(&_SingularityNetToken.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.RenounceRole(&_SingularityNetToken.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.RevokeRole(&_SingularityNetToken.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.RevokeRole(&_SingularityNetToken.TransactOpts, role, account)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactor) Transfer(opts *bind.TransactOpts, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "transfer", recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Transfer(&_SingularityNetToken.TransactOpts, recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactorSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Transfer(&_SingularityNetToken.TransactOpts, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactor) TransferFrom(opts *bind.TransactOpts, sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "transferFrom", sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.TransferFrom(&_SingularityNetToken.TransactOpts, sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_SingularityNetToken *SingularityNetTokenTransactorSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _SingularityNetToken.Contract.TransferFrom(&_SingularityNetToken.TransactOpts, sender, recipient, amount)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SingularityNetToken *SingularityNetTokenTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SingularityNetToken.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SingularityNetToken *SingularityNetTokenSession) Unpause() (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Unpause(&_SingularityNetToken.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_SingularityNetToken *SingularityNetTokenTransactorSession) Unpause() (*types.Transaction, error) {
	return _SingularityNetToken.Contract.Unpause(&_SingularityNetToken.TransactOpts)
}

// SingularityNetTokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the SingularityNetToken contract.
type SingularityNetTokenApprovalIterator struct {
	Event *SingularityNetTokenApproval // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenApproval)
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
		it.Event = new(SingularityNetTokenApproval)
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
func (it *SingularityNetTokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenApproval represents a Approval event raised by the SingularityNetToken contract.
type SingularityNetTokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*SingularityNetTokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenApprovalIterator{contract: _SingularityNetToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenApproval)
				if err := _SingularityNetToken.contract.UnpackLog(event, "Approval", log); err != nil {
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

// SingularityNetTokenPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the SingularityNetToken contract.
type SingularityNetTokenPausedIterator struct {
	Event *SingularityNetTokenPaused // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenPaused)
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
		it.Event = new(SingularityNetTokenPaused)
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
func (it *SingularityNetTokenPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenPaused represents a Paused event raised by the SingularityNetToken contract.
type SingularityNetTokenPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterPaused(opts *bind.FilterOpts) (*SingularityNetTokenPausedIterator, error) {

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenPausedIterator{contract: _SingularityNetToken.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenPaused) (event.Subscription, error) {

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenPaused)
				if err := _SingularityNetToken.contract.UnpackLog(event, "Paused", log); err != nil {
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

// SingularityNetTokenRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the SingularityNetToken contract.
type SingularityNetTokenRoleAdminChangedIterator struct {
	Event *SingularityNetTokenRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenRoleAdminChanged)
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
		it.Event = new(SingularityNetTokenRoleAdminChanged)
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
func (it *SingularityNetTokenRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenRoleAdminChanged represents a RoleAdminChanged event raised by the SingularityNetToken contract.
type SingularityNetTokenRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*SingularityNetTokenRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenRoleAdminChangedIterator{contract: _SingularityNetToken.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenRoleAdminChanged)
				if err := _SingularityNetToken.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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

// SingularityNetTokenRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the SingularityNetToken contract.
type SingularityNetTokenRoleGrantedIterator struct {
	Event *SingularityNetTokenRoleGranted // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenRoleGranted)
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
		it.Event = new(SingularityNetTokenRoleGranted)
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
func (it *SingularityNetTokenRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenRoleGranted represents a RoleGranted event raised by the SingularityNetToken contract.
type SingularityNetTokenRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*SingularityNetTokenRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenRoleGrantedIterator{contract: _SingularityNetToken.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenRoleGranted)
				if err := _SingularityNetToken.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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

// SingularityNetTokenRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the SingularityNetToken contract.
type SingularityNetTokenRoleRevokedIterator struct {
	Event *SingularityNetTokenRoleRevoked // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenRoleRevoked)
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
		it.Event = new(SingularityNetTokenRoleRevoked)
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
func (it *SingularityNetTokenRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenRoleRevoked represents a RoleRevoked event raised by the SingularityNetToken contract.
type SingularityNetTokenRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*SingularityNetTokenRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenRoleRevokedIterator{contract: _SingularityNetToken.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenRoleRevoked)
				if err := _SingularityNetToken.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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

// SingularityNetTokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the SingularityNetToken contract.
type SingularityNetTokenTransferIterator struct {
	Event *SingularityNetTokenTransfer // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenTransfer)
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
		it.Event = new(SingularityNetTokenTransfer)
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
func (it *SingularityNetTokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenTransfer represents a Transfer event raised by the SingularityNetToken contract.
type SingularityNetTokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*SingularityNetTokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenTransferIterator{contract: _SingularityNetToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenTransfer)
				if err := _SingularityNetToken.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// SingularityNetTokenUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the SingularityNetToken contract.
type SingularityNetTokenUnpausedIterator struct {
	Event *SingularityNetTokenUnpaused // Event containing the contract specifics and raw log

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
func (it *SingularityNetTokenUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SingularityNetTokenUnpaused)
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
		it.Event = new(SingularityNetTokenUnpaused)
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
func (it *SingularityNetTokenUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SingularityNetTokenUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SingularityNetTokenUnpaused represents a Unpaused event raised by the SingularityNetToken contract.
type SingularityNetTokenUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_SingularityNetToken *SingularityNetTokenFilterer) FilterUnpaused(opts *bind.FilterOpts) (*SingularityNetTokenUnpausedIterator, error) {

	logs, sub, err := _SingularityNetToken.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &SingularityNetTokenUnpausedIterator{contract: _SingularityNetToken.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_SingularityNetToken *SingularityNetTokenFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *SingularityNetTokenUnpaused) (event.Subscription, error) {

	logs, sub, err := _SingularityNetToken.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SingularityNetTokenUnpaused)
				if err := _SingularityNetToken.contract.UnpackLog(event, "Unpaused", log); err != nil {
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
