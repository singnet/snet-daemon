# SingularityNET Daemon

![ci/cd build]( https://github.com/singnet/snet-daemon/actions/workflows/build.yml/badge.svg)
![ci/cd build]( https://github.com/singnet/snet-daemon/actions/workflows/test.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/singnet/snet-daemon/v6)](https://goreportcard.com/report/github.com/singnet/snet-daemon/v6)
[![Coverage](https://coveralls.io/repos/github/singnet/snet-daemon/badge.svg?branch=master)](https://coveralls.io/github/singnet/snet-daemon?branch=master)
[![Releases](https://img.shields.io/github/release/singnet/snet-daemon/all.svg?style=flat-square)](https://github.com/singnet/snet-daemon/releases)
[![LICENSE](https://img.shields.io/github/license/singnet/snet-daemon.svg?style=flat-square)](https://github.com/singnet/snet-daemon/blob/main/LICENSE)

Users interested in deploying SingularityNET services should use this daemon

The daemon is the adapter with which an otherwise SingularityNET-unaware service implementation can be exposed to the
SingularityNET platform. It is designed to be deployed as a sidecar proxy alongside the service on a given host.
The daemon abstracts the blockchain components away from the clients.
The SNET Daemon interacts with the Multi Party Escrow to facilitate authorization and payment for services and acts as a
passthrough for making API calls to the service. The daemon is the endpoint a client will submit requests to, and they
are then passed to the service after validation by the daemon.

# Table of contents

1. [Install and use](#use)
2. [Configuration](#configuration)
    1. [Main properties](#main_properties)
    2. [Additional properties](#other_properties)
3. [Channel Claim and other commands](#commands)
4. [Build and development](#build)

## Install and use <a name="use"></a>

Precompiled binaries are published with each release,
download [from releases page](https://github.com/singnet/snet-daemon/releases) or use terminal:

```bash
curl -LJO https://github.com/singnet/snet-daemon/releases/download/v6.1.0/snetd-linux-amd64-v6.1.0
chmod +x snetd-linux-amd64-v6.1.0 
```

#### Generate basic config file

For most users, a simple config is enough:

```bash
./snetd-linux-amd64-v6.1.0 init 
```

This command will generate a file `snetd.config.json` in which you will need to
change [some parameters](#main_properties).

#### Generate default full config file

```bash
./snetd-linux-amd64-v6.1.0 init-full
```

#### Run Daemon

```bash
./snetd-linux-amd64-v6.1.0
```

Specifying the path to the config using the '-c' argument:

```bash
./snetd-linux-amd64-v6.1.0 -c name_of_config.json
```

## Configuration <a name="configuration"></a>

Configuration file is a main source of the configuration. Some properties
can be set via environment variables or command line parameters see [table
below](#table_conf). Use `--config` parameter with any command to set configuration file name. By default daemon
use configuration file in JSON format `snetd.config.json` but other formats are
also supported via [Viper](https://github.com/spf13/viper). Use `init-full` or `init`
command to save configuration file with default values. The following configuration properties can be set using a
configuration file.

### Main properties <a name="main_properties"></a>

These properties you should usually change before starting daemon for the first
time.

* **blockchain_network_selected** (required, default `"sepolia"`)
  Name of the network to be used for Daemon possible values are one of (sepolia, main, goerli, local).
  Daemon will automatically read the Registry address associated with this network For local network ( you can also
  specify the registry address manually), see the blockchain_network_config.json

* **daemon_endpoint** (required, default `"127.0.0.1:8080"`) —
  Defines the ip and the port on which the daemon listens to.
  format is :`<host>:<port>`.

* **ethereum_json_rpc_http_endpoint** (optional, default: `"https://sepolia.infura.io/v3"`) —
  endpoint to which daemon sends ethereum JSON-RPC requests;
  Based on the network selected blockchain_network_selected the end point is auto determined
  Example `"https://sepolia.infura.io/v3"` for sepolia testnet.

* **blockchain_provider_api_key** (optional, default: `""`) — basic header authorization key for blockchain providers.
  Tested with
  [infura api key secret](https://docs.infura.io/dashboard/secure-an-api/api-key-secret).

* **organization_id** (required) — Id of the organization to search for [service configuration
  metadata][service-configuration-metadata].

* **service_id** (required) —
  Id of the service to search for [service configuration
  metadata][service-configuration-metadata].

* **passthrough_enabled** (optional; default: `true`) —
  when passthrough is disabled, daemon echoes requests back as responses; `false`
  reserved mostly for testing purposes.

* **service_endpoint** (required except service_type `executable`; default: `http://localhost:5000`) —
  endpoint to which requests should be proxied for handling by service.
  This config is mandatory when `passthrough_enabled` is set to true.
  and needs to be a valid url

* **executable_path** (required if `service_type` is `executable`) —
  path to executable to expose as a service.

* **ipfs_endpoint** (optional; default `"https://ipfs.singularitynet.io:443"`) —
  endpoint of IPFS instance to get [service configuration
  metadata][service-configuration-metadata]

### Free calls properties <a name="free_calls_properties"></a>

These parameters are needed if you specified free_calls in the service metadata.

* **private_key_for_free_calls** (required) —
  The private key is used to sign authorization tokens for free calls. This address is NOT required to have any tokens.

  ⚠️ This private key must correspond to the public address specified in your service_metadata.json under the
  free_calls.public_key field.


* **trusted_free_call_signers** (optional, default SingularityNET addresses) — A list of trusted public addresses that
  are allowed to issue free call tokens on your behalf (e.g. trusted centralized backends or marketplaces).

  ⚠️ If this field is left empty, free calls will not work via the SingularityNET Marketplace, but will still be
  available for clients using the Web3 approach (e.g., via SDKs).

  ```json
  "trusted_free_call_signers": [
  "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F",
  "0x3Bb9b2499c283cec176e7C707Ecb495B7a961ebf" ]
  ```

* **min_balance_for_free_call** (optional, default 10) — The minimum token (FET) balance (in whole tokens) required on
  the user's wallet to be eligible for free calls (Web3-based flow).
  ```json
  "min_balance_for_free_call": "1"
  ```


* **free_calls_per_address** (optional, default empty)

  You can set a separate number of allowed free calls for
  certain addresses (not for web2 approach).

  ```json
  "free_calls_per_address": {
      "0x03607652d3ee4dad68ecea78f266906b421508d5": 500,
      "0x0709e9B78756B740ab0C64427f43f8305fD6D1A7": 100
  }
  ```

### Other properties <a name="other_properties"></a>

This options are less frequently needed.

* **service_credentials** (optional, for `"service_type":"http"` only):
  Array of credentials, example:

  ```
  "service_credentials": [
      {"key": "example_body_param", "value": 12345,"location": "body"},
      {"key": "X-API-Key", "value": "546bd7d4-d3e1-46ba-b752-bc45e4dc5b39", "location": "header"}
    ],
  ```
  Location can be: query, header or body. Query and header values must be string.

* **allowed_user_flag** (optional; default:`false`) — You may need to protect the service provider 's service in test
  environment from being called by anyone, only Authorized users can make calls , when this flag is defined in the
  config, you can enforce this behaviour.You cannot set this flag to true
  in mainnet.  
  This config is applicable only when you have the value to true.
  In which case it becomes mandatory to define the configuration `allowed_users`,

* **allowed_user_addresses** (optional) — List of selected user addresses who can make requests to Daemon
  Is Applicable only when you have `allowed_user_flag` set to true.

* **authentication_addresses** (required if `You need to update Daemon configurations remotely`)
  Contains the Authentication addresses
  that will be used to validate all requests to update Daemon configuration remotely
  through a user interface (Operator UI)

* **auto_ssl_domain** (optional; default: `""`) —  
  domain name for which the daemon should automatically acquire SSL certs
  from [Let's Encrypt](https://letsencrypt.org/).

* **auto_ssl_cache_dir** (optional; only applies if `auto_ssl_domain` is set; default: `".certs"`) —
  directory in which to cache the SSL certs issued by Let's Encrypt

* **blockchain_enabled** (optional; default: `true`) —
  enables or disables blockchain features of daemon; `false` reserved mostly for testing purposes

* **burst_size** (optional; default: Infinite) —
  see [rate limiting configuration](./ratelimit/README.md)

* **daemon_group_name** (optional, default: `"default_group"`) —
  This parameter defines the group the daemon belongs to.
  The group helps determine the recipient address for payments.
  [service configuration
  metadata][service-configuration-metadata].

* **log** (optional) —
  see [logger configuration](./logger/README.md)

* **max_message_size_in_mb** (optional; default: `4`) —
  The default value set is to 4 (units are in MB), this is used to configure the max size in MB of the message received
  by the Daemon.
  In case of Large messages, it is recommended to use streaming than setting a very high value on this configuration.
  It is not recommended to set the value more than 4GB
  `Please make sure your grpc version > 1.25`

* **metering_enabled** (optional, default: `false`) —
  This is used to define if metering needs to be enabled or not .You will need to define a valid ` metering_endpoint`
  when this flag is enabled

* **metering_endpoint** (optional; only applies if `metering_enabled` is set to true) —
  Needs to be a valid url where the request and response stats are published as part of Metering

* **private_key_for_metering** (optional; only applies if `metering_enabled` is set to true)
  This is used for authentication between daemon and the metering service in the context publishing stats, Even the
  latest Channel Status is published, this way the offline channel state balance can also be tracked.
  Daemon will send a signature signed by this private key, metering service will already have the public key
  corresponding
  to this Daemon, metering service will ensure that the signer it receives matches the public key configured at its end.
  This is mandatory only when metering is enabled.

* **ssl_cert** (optional; default: `""`) —
  path to certificate to use for SSL.

* **ssl_key** (optional; only applies if `ssl_cert` is set; default: `""`) —
  path to key to use for SSL.

* **payment_channel_storage_type** (optional; default `"etcd"`) —
  see [etcd storage type](./etcddb#etcd-storage-type)

* **payment_channel_storage_client** (optional) —
  see [etcd client configuration](./etcddb#etcd-client-configuration)

* **payment_channel_storage_server** (optional) —
  see [etcd server configuration](./etcddb#etcd-server-configuration)

* **rate_limit_per_minute** (optional; default: `Infinity`) —
  see [rate limiting configuration](./ratelimit/README.md)

* **registry_address_key** (Optional) —
  Ethereum address of the Registry contract instance.This is auto determined if not specified based on the
  blockchain_network_selected
  If a value is specified, it will be used and no attempt will be made to auto determine the registry address.

* **alerts_email** (optional; default: `""`) — It must be a valid email. if it is empty, then it is considered as alerts
  disabled. see [daemon alerts/notifications configuration](./metrics/README.md)

* **service_heartbeat_type** (optional; default: `""`) — possible type configurations are ```none | grpc | http```. If
  it is left empty, then it is considered as none type. see [daemon heartbeats configuration](./metrics/README.md)

* **heartbeat_endpoint** (optional; default: `""`) — It must be a valid URL. if it is empty, then service state
  always assumed as SERVING, and same will be wrapped in Daemon Heartbeat.
  see [daemon heartbeats configuration](./metrics/README.md)

* **ipfs_timeout** (optional; default: `30`) — All IPFS read/writes timeout if the operations dont complete in 30 sec
  or set duration in this config entry.

* **is_curation_in_progress** (optional; default: `false`) — You may need to protect the service provider's service in
  test environment from being called by anyone, only Authorized users can make calls, when this flag is set to true,
  you can enforce this behaviour. Also see `curation_address_for_validation`

* **token_expiry_in_minutes** (optional; default: `1440` minutes ~24hrs) — This is the default expiry time for a JWT
  token issued.

* **token_secret_key** (optional;) — This is the secret key used to sign a JWT token, please do add this in your
  configuration to make your tokens a lot more secure.

* **notification_endpoint** (optional; default: `""`) — It must be a valid URL. if it is empty, then it is
  considered as alerts disabled. see [daemon alerts/notifications configuration](./metrics/README.md)

#### Environment variables and CLI parameters <a name="table_conf"></a>

| config file key                   | environment variable name              | flag                  |
|-----------------------------------|----------------------------------------|-----------------------|
| `auto_ssl_domain`                 | `SNET_AUTO_SSL_DOMAIN`                 | `--auto-ssl-domain`   |
| `auto_ssl_cache_dir`              | `SNET_AUTO_SSL_CACHE_DIR`              | `--auto-ssl-cache`    |
| `blockchain_enabled`              | `SNET_BLOCKCHAIN_ENABLED`              | `--blockchain`, `-b`  |
| `config_path`                     | `SNET_CONFIG_PATH`                     | `--config`, `-c`      |
| `ethereum_json_rpc_http_endpoint` | `SNET_ETHEREUM_JSON_RPC_HTTP_ENDPOINT` | `--ethereum-endpoint` |
| `hdwallet_index`                  | `SNET_HDWALLET_INDEX`                  | `--wallet-index`      |
| `hdwallet_mnemonic`               | `SNET_HDWALLET_MNEMONIC`               | `--mnemonic`          |
| `passthrough_enabled`             | `SNET_PASSTHROUGH_ENABLED`             | `--passthrough`       |
| `ssl_cert`                        | `SNET_SSL_CERT`                        | `--ssl-cert`          |
| `ssl_key`                         | `SNET_SSL_KEY`                         | `--ssl-key`           |

[service-configuration-metadata]: https://github.com/singnet/wiki/blob/master/multiPartyEscrowContract/MPEServiceMetadata.md

## Channel Claim and other commands <a name="commands"></a>

Gets the latest channel state of the Channel updated in ETCD by the daemons of the same group and then increments the
nonce of the channel.
It then sends and ON-Chain transaction to claim funds. The daemons continue their work independently without any
confirmation from the treasurer on the blockchain.

**Claim funds from the channel**

Refer to the link below on an end to
end [Example of MPE](https://github.com/singnet/wiki/tree/master/multiPartyEscrowContract/front-to-back-examples)
At the moment treasurer server is a part of snet-daemon command line interface.

```bash
./snetd-linux-amd64-v6.1.0 claim --channel-id 0
```

**Full list of commands, use --help to get more information:**

```bash
./snetd-linux-amd64-v6.1.0 --help

Usage:
  snetd [flags]
  snetd [command]

Available Commands:
  channel     Manage operations on payment channels
  freecall    Manage operations on free call users
  help        Help about any command
  init        Write basic configuration to file
  init-full   Write full default configuration to file
  list        List channels, claims in progress, etc
  serve       Is the default option which starts the Daemon.
  version     List the current version of the Daemon.

Flags:
  -c, --config string   config file (default "snetd.config.json")
  -h, --help            help for snetd

Use "snetd [command] --help" for more information about a command.
```

## Build & Development <a name="build"></a>

These instructions are intended to facilitate the development and testing of SingularityNET Daemon.

### Prerequisites and dependencies

Recommended versions:

* [Go 1.24+](https://golang.org/dl/)
* [Protoc 31.1+](https://github.com/protocolbuffers/protobuf/releases)

**Protoc (libprotoc), golang and $GOPATH/bin are recommended to be in environment variables.**

### Installing

* Clone the git repository (for example, $GOPATH/src/github.com/singnet/)

```bash
git clone git@github.com:singnet/snet-daemon.git
cd snet-daemon
```

**Install dependencies and generate bindings**

Bash

```bash
./scripts/install_deps
```

PowerShell

```powershell
./scripts/powershell/install_deps.ps1
```

**Build snet-daemon**. Please note using ldflags, the latest tagged version, sha1 revision and the build time are set as
part of the build. You need to pass the version as shown in the example below:

```bash
./scripts/build <linux/windows/darwin> <amd64/arm/arm64> <version>
```

**Examples**

Bash:

```bash
./scripts/build linux amd64 v6.1.0
```

Powershell:

```powershell
./scripts/powershell/build.ps1 linux amd64 v6.1.0
```

The final binaries will be in the `/build` folder.

#### Docker

1. Build the docker image:

```
docker build -t snet-daemon:v6.1.0 --build-arg VERSION=v6.1.0 . 
```

2. Prepare config and certificates:

```
mkdir ./snet-config/
```

snet-config/

```
  snetd.config.json
  ca.pem
  client.pem
  client-key.pem
  localhost.crt
  localhost.key
  ...anything else your setup needs
```

3. Run the docker image:

Port mapping must match daemon_endpoint.
The daemon listens on the port specified in snetd.config.json → daemon_endpoint (e.g., 0.0.0.0:8080).
When you run the container, the right-hand side of -p HOST:CONTAINER must equal that port.

```bash
docker run -d --rm --name snetd -v "$(pwd)/snet-config:/etc/singnet:ro" -p 8080:8080 snet-daemon:v6.1.0
```

powershell:

```powershell
docker run -d --rm --name snetd -v "$( PWD )\snet-config:/etc/singnet:ro" -p 8080:8080 snet-daemon:v6.1.0
```

#### Multi-compiling

If you want to build daemon for several platforms, run `./scripts/build-all <version>` instead
of `./scripts/build`.

You can edit the script to choose the specific platforms, but by default it will build for Linux, Darwin (OSX), and
Windows.

#### Unit Testing

```bash
./scripts/test
``` 

or

```bash
go test ./...
```

### Fixing errors

If daemon panic with `panic: proto: file "?.proto" is already registered`
you should set environment var `GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn`

### Blockchain network config

You can edit `ethereum_json_rpc_http_endpoint` in `resources/blockchain_network_config.json` before ./scripts/build.

### Signatures in Daemon

* [Payment Signatures](/escrow/README.md).

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the
[tags on this repository](https://github.com/singnet/snet-daemon/tags).

## License

This project is licensed under the MIT License - see the
[LICENSE](https://github.com/singnet/snet-daemon/blob/master/LICENSE) file for details.

