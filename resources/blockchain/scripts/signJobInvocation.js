"use strict";

let fse = require("fs-extra");

module.exports = async (callback) => {
    let jobAddress = fse.readJsonSync("build/state/JobAddress.json").Job;
    fse.writeJsonSync("build/state/JobInvocation.json", {"Signature": signAddress(jobAddress, web3.eth.accounts[1])});
    callback();
};

let signAddress = (address, account) => {
    let valueHex = "0x" + address.slice(2);
    let h = web3.sha3(valueHex, {encoding: "hex"});
    return web3.eth.sign(account, h);
};
