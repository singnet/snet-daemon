"use strict";

let bip = require("bip39");
let fse = require("fs-extra");

let writeMnemonic = (mnemonic) => {
    fse.ensureDirSync("build/state");
    fse.writeJsonSync("build/state/Mnemonic.json", {Mnemonic: mnemonic});
};

let mnemonic = bip.generateMnemonic(128);
writeMnemonic(mnemonic);
