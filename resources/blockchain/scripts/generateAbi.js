// Deprecated, use resources/generate-smart-binds/main.go

// Script converts published contract binaries to the form which can be used by
// abigen to generate golang Ethereum contract stubs.

// parse command line
const optionDefinitions = [
    {name: 'contract-package', type: String},
    {name: 'contract-name', type: String},
    {name: 'go-package', type: String},
    {name: 'output-file', type: String}
];
let commandLineArgs = require('command-line-args');
let options = commandLineArgs(optionDefinitions, {camelCase: true});

let contractPackage = options.contractPackage
let contractName = options.contractName

// load contract from package
let prefix = '../resources/blockchain/node_modules/';
let bytecode = prefix + contractPackage + '/bytecode/' + contractName + '.json';
let abi = prefix + contractPackage + '/abi/' + contractName + '.json';

// call abigen with generated combined JSON
let execFile = require('child_process').execFile;
let child = execFile('abigen',
    ['-abi', abi, '-bin', bytecode, '-pkg', options.goPackage, '-out', options.outputFile, '-type', contractName],
    function (err, stdout, stderr) {
        if (err) {
            console.log(err);
        }
        if (stdout !== '') {
            console.log(stdout);
        }
        if (stderr !== '') {
            console.log(stderr);
        }
    });
