let Contract = require("truffle-contract");
let TokenJson = require("singularitynet-token-contracts/SingularityNetToken.json");
let Token = Contract(TokenJson);
let AgentFactoryJson = require("singularitynet-alpha-blockchain/AgentFactory.json");
let AgentFactory = Contract(AgentFactoryJson);
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
