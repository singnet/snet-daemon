"use strict";

let Contract = require("truffle-contract");
let AgentAbi = require("singularitynet-platform-contracts/abi/Agent.json");
let Agent = Contract({contractName: "Agent", abi: AgentAbi});
let fse = require("fs-extra");

module.exports = async (callback) => {
    Agent.setProvider(web3.currentProvider);
    let agentAddress = fse.readJsonSync("build/state/AgentAddress.json").Agent;
    let agentInstance = Agent.at(agentAddress);
    let createJobResult = await agentInstance.createJob({from: web3.eth.accounts[1], gas: 2000000});
    let jobAddress = createJobResult.logs[0].args.job;
    writeAddress(jobAddress);
    callback();
};

let writeAddress = (address) => {
    fse.ensureDirSync("build/state");
    fse.writeJsonSync("build/state/JobAddress.json", {Job: address});
};
