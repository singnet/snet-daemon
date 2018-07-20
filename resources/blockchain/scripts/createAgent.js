"use strict";

let Contract = require("truffle-contract");
let AgentFactoryAbi = require("singularitynet-platform-contracts/abi/AgentFactory.json");
let AgentFactory = Contract({contractName: "AgentFactory", abi: AgentFactoryAbi});
let fse = require("fs-extra");

module.exports = async (callback) => {
    AgentFactory.setProvider(web3.currentProvider);
    let agentFactoryAddress = fse.readJsonSync("build/state/AgentFactoryAddress.json").AgentFactory;
    let agentFactoryInstance = AgentFactory.at(agentFactoryAddress);
    let createAgentResult = await agentFactoryInstance.createAgent(8, "http://127.0.0.1:5000", "", {from: web3.eth.accounts[0], gas: 2000000});
    let agentAddress = createAgentResult.logs[0].args.agent;
    writeAddress(agentAddress);
    callback();
};

let writeAddress = (address) => {
    fse.ensureDirSync("build/state");
    fse.writeJsonSync("build/state/AgentAddress.json", {Agent: address});
};
