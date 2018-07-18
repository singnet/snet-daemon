"use strict";

let Contract = require("truffle-contract");
let AgentJson = require("singularitynet-alpha-blockchain/Agent.json");
let Agent = Contract(AgentJson);
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
