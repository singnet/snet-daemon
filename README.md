# snet-daemon

[![CircleCI](https://circleci.com/gh/singnet/snet-daemon.svg?style=svg)](https://circleci.com/gh/singnet/snet-daemon)

SingularityNET Daemon
The daemon is the adapter with which an otherwise SingularityNET-unaware service implementation can be exposed to the SingularityNET platform. It is designed to be deployed as a sidecar proxy alongside the service on a given host.
The daemon abstracts the blockchain components away from the clients.
The SNET Daemon interacts with the Multi Party Escrow to facilitate authorization and payment for services and acts as a passthrough for making API calls to the service.The daemon is the endpoint a client will submit requests to, and they are then passed to the service after validation by the daemon.


## Channel Claim command
Gets the latest channel state of the Channel updated in ETCD by the daemons of the same group and then increments the nonce of the channel.
It then sends and ON-Chain transaction to claim funds.The daemons continue their work independently without any confirmation from the treasurer on the blockchain.


## Development

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

* Build snet-daemon (on Linux amd64 platform)
```bash
$ ./scripts/build linux amd64
```

* Generate default config file  snet-daemon (on Linux amd64 platform)
```bash
$ ./build/snetd-linux-amd64 init 
```
**** Please update the registry address in daemon config based on the test network used 

#### Run Deamon
```bash
$ ../build/snetd-linux-amd64
```




### Main commands


* Start ```snet-daemon```
```bash
$ ./snetd-linux-amd64
```

* Claim funds from the channel
 
  Refer to the link below on an end to end [Example of MPE](https://github.com/singnet/wiki/blob/master/multiPartyEscrowContract/MPE_fronttoback_example1.md)

  At the moment treasurer server is a part of snet-daemon command line interface.

```bash
$ ./snetd-linux-amd64 claim --channel-id 0

```

* Full list of commands, use --help to get more information.
```bash
$ ./build/snetd-linux-amd64 --help
Usage:
  snetd [flags]
  snetd [command]

Available Commands:
  claim       Claim money from payment channel
  help        Help about any command
  init        Write default configuration to file
  list        List channels, claims in progress, etc
  serve       Is the default option which starts the Daemon.

Flags:
  -c, --config string   config file (default "snetd.config.json")
  -h, --help            help for snetd

Use "snetd [command] --help" for more information about a command.
```


* Unit Testing
```bash
$ ./scripts/test
```

### Configuration

Configuration file is a main source of the configuration. Some properties
can be set via environment variables or command line parameters see [table
below](#environment-variables-and-cli-parameters). Use `--config`
parameter with any command to set configuration file name.  By default daemon
use configuration file in JSON format `snetd.config.json` but other formats are
also supported via [Viper](https://github.com/spf13/viper). Use `snet init`
command to save configuration file with default values. Following
configuration properties can be set using configuration file.

#### Main properties

These properties you should usually change before starting daemon for the first
time.

##### DAEMON_ENDPOINT (optional; default: `"127.0.0.1:8080"`)
Network interface and port which daemon listens to. This parameter should be
absolutely equal to the corresponding endpoint in the [service configuration
metadata][service-configuration-metadata]. URI format is recommended:
http://<host>:<port>.

##### ETHEREUM_JSON_RPC_ENDPOINT (optional, default: `"http://127.0.0.1:8545"`)
Endpoint to which daemon sends ethereum JSON-RPC requests; recommend
`"https://kovan.infura.io"` for kovan testnet.

##### IPFS_END_POINT (optional; default `"http://localhost:5002/"`)
Endpoint of IPFS instance to get [service configuration
metadata][service-configuration-metadata]

##### REGISTRY_ADDRESS_KEY (required)
Ethereum address of the Registry contract instance.

##### ORGANIZATION_NAME (required)
Name of the organization to search for [service configuration
metadata][service-configuration-metadata].

##### SERVICE_NAME (required)
Name of the service to search for [service configuration
metadata][service-configuration-metadata].

##### PASSTHROUGH_ENABLED (optional; default: `false`)
When passthrough is disabled, daemon echoes requests back as responses; `false`
reserved mostly for testing purposes.

##### PASSTHROUGH_ENDPOINT (required iff `SERVICE_TYPE` != `executable`)
Endpoint to which requests should be proxied for handling by service.

##### EXECUTABLE_PATH (required iff `SERVICE_TYPE` == `executable`)
Path to executable to expose as a service.

#### Other properties

This options are less frequently needed.

##### AUTO_SSL_DOMAIN (optional; default: `""`) 
Domain name for which the daemon should automatically acquire SSL certs from [Let's Encrypt](https://letsencrypt.org/).

##### AUTO_SSL_CACHE_DIR (optional; only applies if `AUTO_SSL_DOMAIN` is set; default: `".certs"`)
Directory in which to cache the SSL certs issued by Let's Encrypt

##### BLOCKCHAIN_ENABLED (optional; default: `true`)
Enables or disables blockchain features of daemon; `false` reserved mostly for testing purposes

##### HDWALLET_INDEX (optional; default: `0`; only applies if `HDWALLET_MNEMONIC` is set)
Derivation index for key to use within HDWallet specified by mnemonic.

##### HDWALLET_MNEMONIC (optional; default: `""`; this or `PRIVATE_KEY` must be set to use `claim` command)
[bip39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki)
mnemonic corresponding to wallet with which daemon transacts on blockchain.

##### LOG (optional)
See [logger configuration](./logger/README.md)

##### PRIVATE_KEY (optional; default: `""`; this or `HDWALLET_MNEMONIC` must be set to use `claim` command)
Private key with which daemon transacts on blockchain.

##### SSL_CERT (optional; default: `""`)
Path to certificate to use for SSL.

##### SSL_KEY (optional; only applies if `SSL_CERT` is set; default: `""`)
Path to key to use for SSL.

##### PAYMENT_CHANNEL_STORAGE_TYPE (optional; default `"etcd"`)
See [etcd storage type](./etcddb#etcd-storage-type)

##### PAYMENT_CHANNEL_STORAGE_CLIENT (optional)
See [etcd client configuration](./etcddb#etcd-client-configuration)

##### PAYMENT_CHANNEL_STORAGE_SERVIER (optional)
See [etcd server configuration](./etcddb#etcd-server-configuration)

#### Environment variables and CLI parameters

|config file key|environment variable name|flag|
|---|---|---|
|`AUTO_SSL_DOMAIN`|`SNET_AUTO_SSL_DOMAIN`|`--auto-ssl-domain`|
|`AUTO_SSL_CACHE_DIR`|`SNET_AUTO_SSL_CACHE_DIR`|`--auto-ssl-cache`|
|`BLOCKCHAIN_ENABLED`|`SNET_BLOCKCHAIN_ENABLED`|`--blockchain`, `-b`|
|`CONFIG_PATH`|`SNET_CONFIG_PATH`|`--config`, `-c`|
|`ETHEREUM_JSON_RPC_ENDPOINT`|`SNET_ETHEREUM_JSON_RPC_ENDPOINT`|`--ethereum-endpoint`|
|`HDWALLET_INDEX`|`SNET_HDWALLET_INDEX`|`--wallet-index`|
|`HDWALLET_MNEMONIC`|`SNET_HDWALLET_MNEMONIC`|`--mnemonic`|
|`PASSTHROUGH_ENABLED`|`SNET_PASSTHROUGH_ENABLED`|`--passthrough`|
|`SSL_CERT`|`SNET_SSL_CERT`|`--ssl-cert`|
|`SSL_KEY`|`SNET_SSL_KEY`|`--ssl-key`|

[service-configuration-metadata]: https://github.com/singnet/wiki/blob/master/multiPartyEscrowContract/MPEServiceMetadata.md

## Release

Precompiled binaries are published with each [release](https://github.com/singnet/snet-daemon/releases).

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the
[tags on this repository](https://github.com/singnet/snet-daemon/tags). 

## License

This project is licensed under the MIT License - see the
[LICENSE](https://github.com/singnet/snet-daemon/blob/master/LICENSE) file for details.

