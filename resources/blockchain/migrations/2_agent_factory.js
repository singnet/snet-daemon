let Contract = require("truffle-contract");
let TokenAbi = require("singularitynet-token-contracts/abi/SingularityNetToken.json");
let TokenNetworks = require("singularitynet-token-contracts/networks/SingularityNetToken.json");
let TokenBytecode = require("singularitynet-token-contracts/bytecode/SingularityNetToken.json");
let Token = Contract({contractName: "SingularityNetToken", abi: TokenAbi, networks: TokenNetworks,
    bytecode: TokenBytecode});
let AgentFactoryAbi = require("singularitynet-platform-contracts/abi/AgentFactory.json");
let AgentFactoryNetworks = require("singularitynet-platform-contracts/networks/AgentFactory.json");
let AgentFactoryBytecode = require("singularitynet-platform-contracts/bytecode/AgentFactory.json");
let AgentFactory = Contract({contractName: "AgentFactory", abi: AgentFactoryAbi, networks: AgentFactoryNetworks,
    bytecode: AgentFactoryBytecode});
let fse = require("fs-extra");

module.exports = function(deployer, network, accounts) {
    Token.setProvider(web3.currentProvider);
    Token.defaults({from: accounts[0], gas: 4000000});
    AgentFactory.setProvider(web3.currentProvider);
    AgentFactory.defaults({from: accounts[0], gas: 4000000});
    deployer.deploy(Token, {overwrite: false})
        .then(() => Token.deployed())
        .then(() => deployer.deploy(AgentFactory, Token.address, {overwrite: false}))
        .then(() => AgentFactory.deployed().then((instance) => writeAddress(instance)));
};

let writeAddress = (instance) => {
    fse.ensureDirSync("build/state");
    fse.writeJsonSync("build/state/AgentFactoryAddress.json", {"AgentFactory": instance.address});
};
