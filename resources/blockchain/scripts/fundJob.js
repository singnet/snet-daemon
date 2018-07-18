"use strict";

let Contract = require("truffle-contract");
let JobJson = require("singularitynet-alpha-blockchain/Job.json");
let Job = Contract(JobJson);
let TokenJson = require("singularitynet-token-contracts/SingularityNetToken.json");
let Token = Contract(TokenJson);
let fse = require("fs-extra");

module.exports = async (callback) => {
    Job.setProvider(web3.currentProvider);
    Token.setProvider(web3.currentProvider);
    let jobAddress = fse.readJsonSync("build/state/JobAddress.json").Job;
    let jobInstance = Job.at(jobAddress);
    let tokenAddress = await jobInstance.token.call();
    let tokenInstance = Token.at(tokenAddress);

    tokenInstance.transfer(web3.eth.accounts[1], 8, {from: web3.eth.accounts[0]});
    tokenInstance.approve(jobInstance.address, 8, {from: web3.eth.accounts[1]});
    jobInstance.fundJob({from: web3.eth.accounts[1]});
    callback();
};
