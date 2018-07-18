"use strict";

let Contract = require("truffle-contract");
let AgentFactoryJson = require("singularitynet-alpha-blockchain/AgentFactory.json");
let AgentFactory = Contract(AgentFactoryJson);
let RegistryJson = require("singularitynet-alpha-blockchain/Registry.json");
let Registry = Contract(RegistryJson);
let fse = require("fs-extra");

module.exports = async (callback) => {
    AgentFactory.setProvider(web3.currentProvider);
    Registry.setProvider(web3.currentProvider);
    let agentFactoryAddress = fse.readJsonSync("build/state/AgentFactoryAddress.json").AgentFactory;
    let registryAddress = fse.readJsonSync("build/state/RegistryAddress.json").Registry;
    let agentFactoryInstance = AgentFactory.at(agentFactoryAddress);
    let registryInstance = Registry.at(registryAddress);
    let createAgentResult = await agentFactoryInstance.createAgent(8, "http://127.0.0.1:5000", {from: web3.eth.accounts[0], gas: 2000000});
    let agentAddress = createAgentResult.logs[0].args.agent;
    writeAddress(agentAddress);
    await registryInstance.createRecord("TestAgent", agentAddress, {from: web3.eth.accounts[0], gas: 2000000});
    callback();
};

let writeAddress = (address) => {
    fse.ensureDirSync("build/state");
    fse.writeJsonSync("build/state/AgentAddress.json", {Agent: address});
};
