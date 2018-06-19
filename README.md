# snet-daemon

SingularityNET Daemon

## Getting Started

These instructions are intended to facilitate the development and testing of SingularityNET Daemon. Users interested in
deploying SingularityNET services using SingularityNET Daemon should install the appropriate binary as
[released](#release).

### Prerequisites

* [Go 1.10+](https://golang.org/dl/)
* [Dep 0.4.1+](https://github.com/golang/dep#installation)
* [Node 8+ w/npm](https://nodejs.org/en/download/)

### Installing

* Clone the git repository
```bash
$ git clone git@github.com:singnet/snet-daemon.git
$ cd snet-daemon
```

* Install development/test dependencies
```bash
$ ./scripts/install
```

### Testing

A simple test script has been setup that does the following
1. Generates a [bip39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki) mnemonic
2. Runs a [ganache-cli](https://github.com/trufflesuite/ganache-cli) test RPC with the generated mnemonic
3. Deploys the required network singleton contracts (SingularityNetToken, AgentFactory, Registry) and
creates an Agent contract instance
4. Writes a daemon configuration file with the Agent contract address, generated mnemonic, and test RPC endpoint
5. Runs an instance of snetd
6. Creates and funds a Job contract instance
7. Signs the job invocation
8. Calls the daemon using the predetermined job address and job signature
9. Cleans up

* Invoke the test script
```bash
$ ./scripts/test
```

## Release

Precompiled binaries are published with each [release](https://github.com/singnet/snet-daemon/releases).

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the
[tags on this repository](https://github.com/singnet/snet-daemon/tags). 

## License

This project is licensed under the MIT License - see the
[LICENSE](https://github.com/singnet/snet-daemon/blob/master/LICENSE) file for details.
