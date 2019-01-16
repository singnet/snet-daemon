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

### Dependencies

* install [Protoc 3.0+](https://askubuntu.com/questions/1072683/how-can-i-install-protoc-on-ubuntu-16-04) 

* install [protoc-gen-go] 
``` bash
$ go get -u github.com/golang/protobuf/protoc-gen-go
```

* install [grpc]
```bash
$ go get -u google.golang.org/grpc
```

* install [golint]
```bash
$ sudo apt-get install golint
```

### Installing

* Clone the git repository to the following path $GOPATH/src/github.com/singnet/
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


* **blockchain_network_selected** (required; default: `"local"`) - 
Name of the network to be used for Daemon possible values are one of (kovan,ropsten,main,local or rinkeby).
Daemon will automatically read the Registry address associated with this network
For local network ( you can also specify the registry address manually),see the blockchain_network_config.json

* **daemon_endpoint** (optional; default: `"127.0.0.1:8080"`) - 
network interface and port which daemon listens to. This parameter should be
absolutely equal to the corresponding endpoint in the [service configuration
metadata][service-configuration-metadata]. URI format is recommended:
http://<host>:<port>.

* **ipfs_end_point** (optional; default `"http://localhost:5002/"`) - 
endpoint of IPFS instance to get [service configuration
metadata][service-configuration-metadata]

* **organization_id** (required) - 
Id of the organization to search for [service configuration
metadata][service-configuration-metadata].

* **service_id** (required) - 
Id of the service to search for [service configuration
metadata][service-configuration-metadata].

* **passthrough_enabled** (optional; default: `false`) - 
when passthrough is disabled, daemon echoes requests back as responses; `false`
reserved mostly for testing purposes.

* **passthrough_endpoint** (required iff `service_type` != `executable`) - 
endpoint to which requests should be proxied for handling by service.

* **executable_path** (required iff `service_type` == `executable`) - 
path to executable to expose as a service.

#### Other properties

This options are less frequently needed.

* **auto_ssl_domain** (optional; default: `""`) -  
domain name for which the daemon should automatically acquire SSL certs from [Let's Encrypt](https://letsencrypt.org/).

* **auto_ssl_cache_dir** (optional; only applies if `auto_ssl_domain` is set; default: `".certs"`) - 
directory in which to cache the SSL certs issued by Let's Encrypt

* **blockchain_enabled** (optional; default: `true`) - 
enables or disables blockchain features of daemon; `false` reserved mostly for testing purposes

* **burst_size** (optional; default: Infinite) - 
see [rate limiting configuration](./ratelimit/README.md)

* **hdwallet_index** (optional; default: `0`; only applies if `hdwallet_mnemonic` is set) - 
derivation index for key to use within HDWallet specified by mnemonic.

* **hdwallet_mnemonic** (optional; default: `""`; this or `private_key` must be set to use `claim` command) - 
[bip39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki)
mnemonic corresponding to wallet with which daemon transacts on blockchain.

* **private_key** (optional; default: `""`; this or `hdwallet_mnemonic` must be set to use `claim` command) - 
private key with which daemon transacts on blockchain.

* **log** (optional) - 
see [logger configuration](./logger/README.md)

* **monitoring_enabled** (optional; default: `true`) - 
Enable or Disable monitoring of Requests arrived and response sent back

* **monitoring_svc_end_point** (optional;only applies if `monitoring_enabled` is set to true) - 
Needs to be a vaild url where the request and response stats are published as part of monitoring

* **ssl_cert** (optional; default: `""`) - 
path to certificate to use for SSL.

* **ssl_key** (optional; only applies if `ssl_cert` is set; default: `""`) - 
path to key to use for SSL.

* **payment_channel_storage_type** (optional; default `"etcd"`) - 
see [etcd storage type](./etcddb#etcd-storage-type)

* **payment_channel_storage_client** (optional) - 
see [etcd client configuration](./etcddb#etcd-client-configuration)

* **payment_channel_storage_server** (optional) - 
see [etcd server configuration](./etcddb#etcd-server-configuration)

* **rate_limit_per_minute** (optional; default: `Infinity`) - 
see [rate limiting configuration](./ratelimit/README.md)
 
* **alerts_email** (optional; default: `""`) - It must be a valid email. if it is empty, then it is considered as alerts disabled. see [daemon alerts/notifications configuration](./metrics/README.md)

* **notification_svc_end_point** (optional; default: `""`) - It must be a valid URL. if it is empty, then it is considered as alerts disabled. see [daemon alerts/notifications configuration](./metrics/README.md)

* **service_heartbeat_type** (optional; default: `grpc`) - possible type configurations are ```none | grpc | http```. If it is left empty, then it is considered as none type. see [daemon heartbeats configuration](./metrics/README.md)

* **heartbeat_svc_end_point** (optional; default: `""`) - It must be a valid URL. if it is empty, then service state always assumed as SERVING, and same will be wrapped in Daemon Heartbeat. see [daemon heartbeats configuration](./metrics/README.md)

* **ipfs_timeout** (optional; default: `30`) - All IPFS read/writes timeout if the operations doesnt complete in 30 sec or set duration in this config entry.

#### Environment variables and CLI parameters

|config file key|environment variable name|flag|
|---|---|---|
|`auto_ssl_domain`|`SNET_AUTO_SSL_DOMAIN`|`--auto-ssl-domain`|
|`auto_ssl_cache_dir`|`SNET_AUTO_SSL_CACHE_DIR`|`--auto-ssl-cache`|
|`blockchain_enabled`|`SNET_BLOCKCHAIN_ENABLED`|`--blockchain`, `-b`|
|`config_path`|`SNET_CONFIG_PATH`|`--config`, `-c`|
|`ethereum_json_rpc_endpoint`|`SNET_ETHEREUM_JSON_RPC_ENDPOINT`|`--ethereum-endpoint`|
|`hdwallet_index`|`SNET_HDWALLET_INDEX`|`--wallet-index`|
|`hdwallet_mnemonic`|`SNET_HDWALLET_MNEMONIC`|`--mnemonic`|
|`passthrough_enabled`|`SNET_PASSTHROUGH_ENABLED`|`--passthrough`|
|`ssl_cert`|`SNET_SSL_CERT`|`--ssl-cert`|
|`ssl_key`|`SNET_SSL_KEY`|`--ssl-key`|

[service-configuration-metadata]: https://github.com/singnet/wiki/blob/master/multiPartyEscrowContract/MPEServiceMetadata.md

## Release

Precompiled binaries are published with each [release](https://github.com/singnet/snet-daemon/releases).

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the
[tags on this repository](https://github.com/singnet/snet-daemon/tags). 

## License

This project is licensed under the MIT License - see the
[LICENSE](https://github.com/singnet/snet-daemon/blob/master/LICENSE) file for details.

