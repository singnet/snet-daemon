let Contract = require("truffle-contract");
let RegistryJson = require("singularitynet-alpha-blockchain/Registry.json");
let Registry = Contract(RegistryJson);
let fse = require("fs-extra");

module.exports = function(deployer, network, accounts) {
    Registry.setProvider(web3.currentProvider);
    Registry.defaults({from: accounts[0], gas: 4000000});
    deployer.deploy(Registry, {overwrite: false})
        .then(() => Registry.deployed().then((instance) => writeAddress(instance)));
};

let writeAddress = (instance) => {
    fse.ensureDirSync("build/state");
    fse.writeJsonSync("build/state/RegistryAddress.json", {"Registry": instance.address});
};
