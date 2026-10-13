# SingularityNET Daemon

![ci/cd build](https://github.com/singnet/snet-daemon/actions/workflows/build.yml/badge.svg)
![tests](https://github.com/singnet/snet-daemon/actions/workflows/test.yml/badge.svg)
[![coverage](https://coveralls.io/repos/github/singnet/snet-daemon/badge.svg?branch=dev)](https://coveralls.io/github/singnet/snet-daemon?branch=dev)
[![Releases](https://img.shields.io/github/release/singnet/snet-daemon/all.svg?style=flat-square)](https://github.com/singnet/snet-daemon/releases)
[![LICENSE](https://img.shields.io/github/license/singnet/snet-daemon.svg?style=flat-square)](LICENSE)

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
curl -LJO https://github.com/singnet/snet-daemon/releases/download/v6.2.3/snetd-linux-amd64-v6.2.3
chmod +x snetd-linux-amd64-v6.2.3 
```

#### Generate basic config file

For most users, a simple config is enough:

```bash
./snetd-linux-amd64-v6.2.3 init 
```

This command will generate a file `snetd.config.json` in which you will need to
change [some parameters](#main_properties).

#### Generate default full config file

```bash
./snetd-linux-amd64-v6.2.3 init-full
```

#### Run Daemon

```bash
./snetd-linux-amd64-v6.2.3
```

Specifying the path to the config using the '-c' argument:

```bash
./snetd-linux-amd64-v6.2.3 -c name_of_config.json
```

## Configuration <a name="configuration"></a>

The daemon loads `snetd.config.json` from the working directory when it exists.
Use `--config` / `-c` to select another file; an explicitly selected missing file
is an error. Without a file, built-in defaults are used. Other file formats are
supported through [Viper](https://github.com/spf13/viper).
Use `init` for a basic template or `init-full` to write the currently available
configuration defaults; neither command produces a ready-to-deploy configuration.
Some settings also support [environment variables and CLI flags](#table_conf).
The complete parameter reference, precedence rules, and limitations are in
[config/README.md](config/README.md).

### Main properties <a name="main_properties"></a>

These properties you should usually change before starting daemon for the first
time.

* **blockchain_network_selected** (optional, default `"sepolia"`) —
  Selects an entry in [the network configuration](resources/blockchain_network_config.json):
  `sepolia`, `main`, `goerli`, or `local`. An entry's presence does not guarantee
  that its public RPC service or contracts are still available. RPC endpoints and
  the Registry address are derived from the selected network unless overridden.

* **daemon_endpoint** (optional, default `"127.0.0.1:8080"`) —
  Defines the ip and the port on which the daemon listens to.
  format is :`<host>:<port>`.

* **ethereum_json_rpc_http_endpoint** (optional, network-derived) —
  Ethereum JSON-RPC HTTP endpoint. A nonempty local value overrides the selected
  network's endpoint. For Infura, use a complete URL including your project ID;
  `https://sepolia.infura.io/v3` alone is not the shipped default URL.

* **blockchain_provider_api_key** (optional, default: `""`) —
  Optional RPC authentication credential. The HTTP client uses Basic Auth for
  an Infura secret and Bearer Auth for a JWT or other providers. The WebSocket
  client currently uses Basic Auth. This does not supply the project ID in an
  Infura endpoint URL.

* **organization_id** (required) — Id of the organization to search for [service configuration
  metadata][service-configuration-metadata].

* **service_id** (required) —
  Id of the service to search for [service configuration
  metadata][service-configuration-metadata].

* **passthrough_enabled** (optional; default: `true`) —
  when passthrough is disabled, daemon echoes requests back as responses; `false`
  reserved mostly for testing purposes.

* **service_endpoint** (default: `"http://localhost:5000"`) —
  Backend endpoint to which requests are forwarded. Startup validates this
  address even with passthrough disabled or a process backend. It must identify
  a host and must not point back to the daemon's own listening address.

* **executable_path** (required for a `process` backend; default: `""`) —
  Path to the executable to expose as a service. The backend `service_type`
  (`grpc`, `http`, `jsonrpc`, or `process`) comes from service metadata, not a
  local daemon configuration key.

* **ipfs_endpoint** (optional; default `"https://ipfs.singularitynet.io:443"`) —
  endpoint of IPFS instance to get [service configuration
  metadata][service-configuration-metadata]

### Free calls properties <a name="free_calls_properties"></a>

These parameters configure free-call token issuance and quotas. See the
[free-call guide](freecall/README.md) for the RPC and signature formats.

* **private_key_for_free_calls** (required for token issuance; default: `""`) —
  The private key is used to sign authorization tokens for free calls. This address is NOT required to have any tokens.

  ⚠️ This private key must correspond to the public address specified in your service_metadata.json under the
  selected group's `groups[].free_call_signer_address` field. A missing key or
  mismatched address disables the free-call state/token service.


* **trusted_free_call_signers** (optional, default SingularityNET addresses) —
  Trusted caller addresses that may request daemon-signed tokens for a `user_id`
  without the ordinary wallet balance check (e.g. backends or marketplaces).

  An explicit empty array disables trusted-signer token issuance (including the
  Marketplace flow); direct wallet clients can still request tokens if the other
  free-call requirements are met. Omitting the key retains the built-in signers.

  ```json
  {
    "trusted_free_call_signers": [
      "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F",
      "0x3Bb9b2499c283cec176e7C707Ecb495B7a961ebf"
    ]
  }
  ```

* **min_balance_for_free_call** (optional, default 10) — The minimum token (FET) balance (in whole tokens) required on
  the user's wallet to be eligible for free calls (Web3-based flow).
  ```json
  { "min_balance_for_free_call": "1" }
  ```


* **free_calls_per_address** (optional, default empty)

  Override the quota for a signing address. A missing entry or `0` uses the
  service metadata quota; `"unlimited"`, `"infinity"`, or `-1` allows unlimited
  calls. Address matching is case-insensitive. Trusted-signer flows use the
  signer's address for this lookup, while usage records also distinguish user IDs.

  ```json
  {
    "free_calls_per_address": {
      "0x0000000000000000000000000000000000000001": 500,
      "0x0000000000000000000000000000000000000002": 100
    }
  }
  ```

### Other properties <a name="other_properties"></a>

This options are less frequently needed.

* **service_credentials** (optional, for metadata `"service_type":"http"` only):
  Array of credentials, example:

  ```json
  {
    "service_credentials": [
      {"key": "example_body_param", "value": 12345,"location": "body"},
      {"key": "X-API-Key", "value": "<BACKEND_API_KEY>", "location": "header"}
    ]
  }
  ```
  Location can be: query, header or body. Query and header values must be string.

* **service_timeout** (optional; default: `"100s"`) —
  Fallback backend request deadline in the gRPC proxy when the caller has not
  supplied a deadline. Use a duration string such as `"30s"`; this is not a
  universal timeout for heartbeat, metering, or the simple HTTP daemon mode.

* **allowed_user_flag** (optional; default:`false`) — You may need to protect the service provider 's service in test
  environment from being called by anyone, only Authorized users can make calls , when this flag is defined in the
  config, you can enforce this behaviour.You cannot set this flag to true
  in mainnet.  
  This config is applicable only when you have the value to true.
  In that case, `allowed_user_addresses` must contain at least one valid address.

* **allowed_user_addresses** (optional) — List of selected user addresses who can make requests to Daemon
  Is Applicable only when you have `allowed_user_flag` set to true.

* **authentication_addresses** (optional, default empty) —
  Addresses authorized to use the configuration-service RPCs. An empty list
  prevents access. Reading configuration and pause/resume operations are
  implemented; remote updates are not. See [configuration service](configuration_service/README.md).

* **auto_ssl_domain** (optional; default: `""`) —  
  domain name for which the daemon should automatically acquire SSL certs
  from [Let's Encrypt](https://letsencrypt.org/).
  Requires inbound access to port 80 for HTTP-01 challenges and a writable cache.

* **auto_ssl_cache_dir** (optional; only applies if `auto_ssl_domain` is set; default: `".certs"`) —
  directory in which to cache the SSL certs issued by Let's Encrypt

* **blockchain_enabled** (optional; default: `true`) —
  enables or disables blockchain features of daemon; `false` reserved mostly for testing purposes

* **burst_size** (optional; effective default: `2147483647`) —
  see [rate limiting configuration](./ratelimit/README.md)

* **daemon_group_name** (optional, default: `"default_group"`) —
  This parameter defines the group the daemon belongs to.
  The group helps determine the recipient address for payments.
  [service configuration
  metadata][service-configuration-metadata].

* **log** (optional) —
  see [logger configuration](./logger/README.md)

* **max_message_size_in_mb** (optional; default: `4`) —
  Message size limit in MiB (`1024 * 1024` bytes), despite the historical key
  name. Startup accepts integers from `1` to `2048`. This configures the daemon's
  gRPC receive limit and supported backend send/receive limits. Prefer streaming
  chunks to increasing the limit for large payloads.

* **metering_enabled** (optional, default: `false`) —
  This is used to define if metering needs to be enabled or not .You will need to define a valid ` metering_endpoint`
  when this flag is enabled

* **metering_endpoint** (required when `metering_enabled` is true; default: `""`) —
  Base URL for the metering service. Startup verification uses `/metering/verify`
  and usage reporting uses `/metering/usage`.

* **private_key_for_metering** (default: `""`) —
  Signing key required for metering registration and signed usage publication.
  The heartbeat-notification publication path also uses this key; it is not an
  unsigned webhook. See [metrics configuration](metrics/README.md).

* **ssl_cert** (optional; default: `""`) —
  API server certificate PEM; must be configured together with `ssl_key`.

* **ssl_key** (optional; default: `""`) —
  Matching API server private-key PEM; must be configured together with `ssl_cert`.

* **payment_channel_storage_type** (optional; default `"etcd"`) —
  see [storage types](etcddb/README.md#etcd-storage). Use only `etcd` or `memory`;
  unknown values currently fall back to nonpersistent memory storage.

* **payment_channel_storage_client** (optional) —
  see [etcd client configuration](etcddb/README.md#client-configuration)

* **payment_channel_storage_server** (optional) —
  see [embedded etcd server configuration](etcddb/README.md#embedded-server-configuration)

* **rate_limit_per_minute** (optional; default: `Infinity`) —
  see [rate limiting configuration](./ratelimit/README.md)

* **registry_address_key** (Optional) —
  Ethereum address of the Registry contract instance.This is auto determined if not specified based on the
  blockchain_network_selected
  If a value is specified, it will be used and no attempt will be made to auto determine the registry address.

* **alerts_email** (optional; default: `""`) — It must be a valid email. if it is empty, then it is considered as alerts
  disabled. see [daemon alerts/notifications configuration](./metrics/README.md)

* **service_heartbeat_type** (optional; default: `""`) —
  Accepts `""`, `none`, `tcp`, `grpc`, `http`, or `https`. The first three perform
  a TCP connection check against `service_endpoint`; they do not disable health
  checking. See [daemon heartbeats](metrics/README.md#heartbeat).

* **heartbeat_endpoint** (default: `""`) —
  Configure the service health endpoint when selecting `grpc`, `http`, or `https`.
  TCP mode uses `service_endpoint` instead. Leaving this empty does not guarantee
  a `SERVING` result. See [daemon heartbeats](metrics/README.md#heartbeat).

* **ipfs_timeout** (optional; default: `30`) —
  IPFS HTTP client timeout in whole seconds, not a duration string. It does not
  apply to Lighthouse gateway requests. `0` disables this client's timeout.

* **token_expiry_in_minutes** (optional; default: `1440` minutes ~24hrs) — This is the default expiry time for a JWT
  token issued.

* **token_secret_key** —
  Secret used to sign prepaid JWTs. With blockchain enabled, the configured
  value must contain at least 32 bytes. A public test value is built in and
  passes that length check: **replace it with your own random secret for production**.
  Daemon replicas validating the same prepaid tokens must share the secret.

* **notification_endpoint** (optional; default: `""`) — It must be a valid URL. if it is empty, then it is
  considered as alerts disabled. see [daemon alerts/notifications configuration](./metrics/README.md)

#### Environment variables and CLI parameters <a name="table_conf"></a>

The flags below other than `--config` belong to `snetd serve`.
Explicitly supplied, bound flags override environment variables, which override
the config file and defaults. `--config` directly selects the file; it is not a
Viper setting and has no working `SNET_CONFIG_PATH` equivalent.

| config file key                   | environment variable name              | flag                  |
|-----------------------------------|----------------------------------------|-----------------------|
| `auto_ssl_domain`                 | `SNET_AUTO_SSL_DOMAIN`                 | `--auto-ssl-domain`   |
| `auto_ssl_cache_dir`              | `SNET_AUTO_SSL_CACHE_DIR`              | `--auto-ssl-cache`    |
| `blockchain_enabled`              | `SNET_BLOCKCHAIN_ENABLED`              | `--blockchain`, `-b`  |
| File selection (not a config key) | —                                     | `--config`, `-c`      |
| `daemon_endpoint`                | `SNET_DAEMON_ENDPOINT`                 | —                    |
| `daemon_type`                    | `SNET_DAEMON_TYPE`                     | `--type`, `-t`        |
| `ethereum_json_rpc_http_endpoint` | `SNET_ETHEREUM_JSON_RPC_HTTP_ENDPOINT` | —                    |
| `passthrough_enabled`             | `SNET_PASSTHROUGH_ENABLED`             | `--passthrough`       |
| `ssl_cert`                        | `SNET_SSL_CERT`                        | `--ssl-cert`          |
| `ssl_key`                         | `SNET_SSL_KEY`                         | `--ssl-key`           |

Use the config file for nested objects and arrays. Underscores are not translated
into nested-key separators: for example, `SNET_LOG_LEVEL` does not set `log.level`.
See [configuration loading](config/README.md#loading-and-precedence) for details.

[service-configuration-metadata]: https://github.com/singnet/wiki/blob/master/multiPartyEscrowContract/MPEServiceMetadata.md

## Channel Claim and other commands <a name="commands"></a>

The [ProviderControlService](escrow/control_service.proto) prepares claims from
the shared payment state. The separate SNET CLI (`snet treasurer`) submits the
corresponding on-chain transaction; the daemon CLI (`snetd`) does not contain a
`claim` command. A compatible provider client can also implement this workflow.

**Claim funds from the channel**

Use the [SNET CLI treasurer commands](https://dev.singularitynet.io/docs/products/DecentralizedAIPlatform/CLI/Manual/Treasurer/):

```bash
# Inspect payments available to claim (no blockchain transaction)
snet treasurer print-unclaimed --endpoint <DAEMON_ENDPOINT>

# Claim payments from a specific channel
snet treasurer claim --endpoint <DAEMON_ENDPOINT> <CHANNEL_ID>

# Claim payments from all available channels
snet treasurer claim-all --endpoint <DAEMON_ENDPOINT>
```

Replace the placeholders with the daemon's reachable API endpoint (not an etcd
endpoint) and a channel ID. Configure the CLI for the same blockchain network
and use the provider identity authorized by the selected group's payment
recipient address. The claim commands submit blockchain transactions and require
funds for gas; the examples retain interactive transaction confirmation.

The daemon prepares claim data through `StartClaim` /
`StartClaimForMultipleChannels`, while the provider client signs and submits the
transaction. See the [payment and claim protocol](escrow/README.md). Local daemon
commands can list channels and claims or unlock state; they do not submit claim
transactions.

**Full list of commands, use --help to get more information:**

```bash
./snetd-linux-amd64-v6.2.3 --help

Usage:
  snetd [flags]
  snetd [command]

Available Commands:
  channel      Manage operations on payment channels
  completion   Generate the autocompletion script for the specified shell
  freecall     Manage operations on free call users
  generate-key Generate new pair of keys
  help         Help about any command
  init         Write default configuration to file
  init-full    Write full default configuration to file
  list         List channels, claims in progress, etc
  serve        Is the default option which starts the Daemon.
  version      List the current version of the Daemon.

Flags:
  -c, --config string   config file (default "snetd.config.json")
  -h, --help            help for snetd

Use "snetd [command] --help" for more information about a command.
```

## Build & Development <a name="build"></a>

These instructions are intended to facilitate the development and testing of SingularityNET Daemon.

### Prerequisites and dependencies

Toolchain versions:

* [Go](https://go.dev/dl/): at least the version in [go.mod](go.mod), currently **1.27.2**; CI uses that version.
* [Protoc](https://github.com/protocolbuffers/protobuf/releases): CI uses **36.2**.

Put `go`, `protoc`, and the Go-installed code generators (`GOBIN`, or `GOPATH/bin`
when unset) on `PATH`.

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

**Build snet-daemon**. Using ldflags, the supplied version, Git revision, and build time are set as
part of the build. You need to pass the version as shown in the example below:

```bash
./scripts/build <linux/windows/darwin> <amd64/arm/arm64> <version>
```

**Examples**

Bash:

```bash
./scripts/build linux amd64 v6.2.3
```

Powershell:

```powershell
./scripts/powershell/build.ps1 linux amd64 v6.2.3
```

The final binaries will be in the repository's `build/` folder.

#### Docker

1. Build the docker image:

```
docker build -f docker/Dockerfile -t snet-daemon:v6.2.3 --build-arg VERSION=v6.2.3 .
```

2. Prepare config and certificates:

The source Dockerfile expects generated bindings in the build context: run the
dependency-generation step above first. See [Docker instructions](docker/README.md)
for release-binary images, supported build arguments, and runtime permissions.

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
Use `"daemon_endpoint": "0.0.0.0:8080"` to make it reachable through the mapping.
For a minimal container configuration, use `"log": {"output": {"type": ["stdout"]}}`.
File logs, embedded-etcd data/logs, and ACME cache need separate writable locations
owned by UID 10001; the default `/app` working directory is not writable by that user.

```bash
docker run -d --rm --name snetd -v "$(pwd)/snet-config:/etc/singnet:ro" -p 8080:8080 snet-daemon:v6.2.3
```

powershell:

```powershell
docker run -d --rm --name snetd -v "${PWD}\snet-config:/etc/singnet:ro" -p 8080:8080 snet-daemon:v6.2.3
```

#### Multi-compiling

If you want to build daemon for several platforms, run `./scripts/build_all_platforms <version>` instead
of `./scripts/build`.

You can edit the script to choose the specific platforms, but by default it will build for Linux, Darwin (OSX), and
Windows.

#### Unit Testing

```bash
go test ./...
```

Generate bindings first. The legacy `scripts/test` helper is not equivalent: it
modifies dependencies, skips nested packages, and expects a Coveralls upload
environment. CI uses `go test ./...` and a separate
[golangci-lint workflow](.github/workflows/lint.yml).

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
