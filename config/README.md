# Daemon configuration reference

This reference describes the current source tree. Defaults come from
[config.go](config.go), with network-derived settings from
[blockchain_network_config.go](blockchain_network_config.go) and
[the network resource](../resources/blockchain_network_config.json).
An omitted setting is not always equivalent to an empty string, `0`, `false`, or
an empty array; those differences are called out below.

The operator-UI schema in `configuration_schema.go` is incomplete and contains
historical values. It is not the source of runtime defaults or a complete
validation schema. `init-full` writes settings known to Viper, not every
optional parameter that the daemon supports.

## Loading and precedence

Run commands from a directory containing `snetd.config.json`, or select a file:

```sh
snetd serve --config /etc/singnet/snetd.config.json
```

- The default file name is `snetd.config.json`, relative to the working directory.
  If it does not exist and `--config` was not supplied, built-in defaults are used.
  An explicitly selected missing/unreadable file is an error.
- `snetd init -c example.json` writes the basic template; `init-full` writes the
  currently available default settings. Both refuse to overwrite an existing file.
  Replace organization/service placeholders and supply deployment-specific values.
- JSON, YAML, and other formats recognized by the pinned Viper version can be
  selected by file extension. Examples here use JSON.
- For bound scalar settings, priority is explicit programmatic `Set`, then a
  changed CLI flag, environment variable, config file, and built-in default.
  A flag's help/default value is not necessarily the effective runtime default.
- Deprecated-name migration uses `Set`: an old key can override the new key even
  when both are present. Remove old names instead of setting both.
- RPC endpoints and the Registry address use a nonempty local override before
  network-derived values. Etcd client settings have their own metadata/local
  precedence described below; embedded-server settings are independent.
- Relative filesystem paths resolve from the **process working directory**, not
  from the directory containing the config file.
- There is no general config-file hot reload. Restart after changing settings
  unless a specific subsystem documents a reload mechanism.

### Environment variables

Simple top-level settings read through Viper getters use the `SNET_` prefix and
uppercase key, for example `SNET_DAEMON_ENDPOINT` or `SNET_METERING_ENABLED`.
An empty environment variable is normally treated as unset, not as an instruction
to clear a file/default value.

Use the config file for arrays, maps, and nested objects. The daemon does not
configure a dot-to-underscore environment-key replacer: `SNET_LOG_LEVEL` does
not set `log.level`. Although direct getters can read a dotted name such as
`SNET_LOG.LEVEL`, this is not a portable shell interface, and settings discovered
only through ENV may be omitted by `Unmarshal` / `SubWithDefault`. In particular,
ENV-only etcd endpoints are not a reliable configuration path.

`--config` is read directly by the CLI, not through Viper. Neither `config_path`
nor `SNET_CONFIG_PATH` selects a configuration file.

### Supported CLI overrides

Except for the global `--config`, these flags belong to `snetd serve`:

| Flag | Config key | Effective default without overrides |
|------|------------|-------------------------------------|
| `--config`, `-c` | File selection, not a config key | `snetd.config.json` |
| `--type`, `-t` | `daemon_type` | `"grpc"` |
| `--blockchain`, `-b` | `blockchain_enabled` | `true` |
| `--passthrough` | `passthrough_enabled` | `true`; use `--passthrough=false` to disable |
| `--auto-ssl-domain` | `auto_ssl_domain` | `""` |
| `--auto-ssl-cache` | `auto_ssl_cache_dir` | `".certs"` |
| `--ssl-cert` | `ssl_cert` | `""` |
| `--ssl-key` | `ssl_key` | `""` |

The current help also exposes legacy flags that have no active configuration
consumer: `--port`, `--db-path`, `--mnemonic`, `--wallet-index`, `--service-type`,
`--wire-encoding`, and `--poll-sleep`. Do not use them to configure a deployment.
Set `daemon_endpoint` to choose the listener; backend type and encoding come
from service metadata. There is no `--ethereum-endpoint` flag.

## Daemon and backend

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `daemon_endpoint` | string | `"127.0.0.1:8080"` | Listener in `host:port` form. Use a reachable interface such as `0.0.0.0` for container port publishing. |
| `daemon_type` | string | `"grpc"` | `grpc` enables the gRPC proxy, gRPC-Web and auxiliary HTTP routes. `http` selects a simple HTTP proxy with a reduced feature set; startup warns that it is not for production. |
| `organization_id` | string | `"YOUR_ORG_ID"` | Replace with the registered organization ID when blockchain is enabled. |
| `service_id` | string | `"YOUR_SERVICE_ID"` | Replace with the registered service ID when blockchain is enabled. |
| `daemon_group_name` | string | `"default_group"` | Selects the group in organization and service metadata, including payment and free-call settings. |
| `passthrough_enabled` | boolean | `true` | Forward service calls. `false` selects a test loopback handler; it does not independently disable payment validation. |
| `service_endpoint` | string | `"http://localhost:5000"` | Backend address, also used by TCP heartbeat. Validation requires a host and rejects a direct loop back to the daemon listener, even when passthrough is off or the backend is a process. Prefer an explicit URL scheme. |
| `service_timeout` | duration string | `"100s"` | Default request deadline for the gRPC proxy's backend calls when the caller has no deadline. See timeout notes below. |
| `max_message_size_in_mb` | integer | `4` | Actually MiB (`1024 * 1024` bytes). Accepted range: `1`–`2048`. Applied to gRPC receives and supported backend send/receive calls, not a universal HTTP body limit. |
| `executable_path` | string | `""` | Required for metadata `service_type: process`. The executable receives the RPC method as an argument and request bytes on stdin. |
| `service_credentials` | array of objects | absent / empty | For metadata `service_type: http`. Entries have `key`, `value`, and `location` (`header`, `query`, or `body`). Header/query values must be strings. Treat credentials as secrets. |

`service_type` and wire `encoding` are service metadata fields, not local config
keys. Supported proxy backend types are `grpc`, `http`, `jsonrpc`, and `process`.
For a gRPC backend, an `https://` endpoint enables TLS; other schemes use the
current insecure transport path.

Use valid, positive duration strings such as `"30s"` or `"2m"` for
`service_timeout`. An existing caller deadline is preserved, not capped by this
setting. A nonpositive parsed duration does not add a context deadline. The HTTP
backend client also has a separate timeout of `service_timeout + 1s`.
The simple `daemon_type: http` handler does not use this setting. Invalid input
is not a supported way to request a default: validation currently logs a fallback
without writing the normalized value back to Viper.

## Blockchain and access control

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `blockchain_enabled` | boolean | `true` | Enables blockchain-backed metadata, payment and related services. `false` is a test mode that bypasses payment validation; it is not a production access-control mechanism. |
| `blockchain_network_selected` | string | `"sepolia"` | Selects `sepolia`, `main`, `goerli`, or `local` from the shipped network resource. Availability of external RPCs/contracts is not guaranteed by an entry's presence. |
| `ethereum_json_rpc_http_endpoint` | string | network-derived | Nonempty local override for the Ethereum HTTP RPC URL. Empty/omitted retains the network value. |
| `ethereum_json_rpc_ws_endpoint` | string | network-derived | Nonempty local override for the Ethereum WebSocket RPC URL. Empty/omitted retains the network value. |
| `registry_address_key` | address string | network/contract-derived | Nonempty local override for the Registry contract. Otherwise use the network resource or contract deployment data for its network ID. |
| `blockchain_provider_api_key` | string | `""` | Optional RPC credential: the HTTP path selects Infura Basic Auth or Bearer Auth for JWT/other providers. The WebSocket path currently uses Basic Auth. |
| `allowed_user_flag` | boolean | `false` | Restrict supported payment/free-call validation to listed addresses. Forbidden when the selected network is `main`. |
| `allowed_user_addresses` | address array | absent / empty | Required and nonempty when `allowed_user_flag` is true; each entry must be a valid Ethereum address. |
| `authentication_addresses` | address array | absent / empty | Authorized configuration-service callers. Empty means no authorized operators. Invalid entries are logged and skipped. See [configuration service](../configuration_service/README.md). |

`network_id` is read from the selected entry in the network resource, not as an
independent local daemon override. The build scripts embed that resource in the
binary; changing the JSON beside an already-built binary does not replace the
embedded mapping. Nonempty local endpoint/Registry overrides still apply.

## Metadata retrieval

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `ipfs_endpoint` | string | `"https://ipfs.singularitynet.io:443"` | IPFS API endpoint for metadata and model/proto retrieval. |
| `lighthouse_endpoint` | string | `"https://gateway.lighthouse.storage/ipfs/"` | Gateway prefix for `filecoin://` metadata references. The CID is appended directly; include the required trailing slash. |
| `ipfs_timeout` | integer seconds | `30` | IPFS HTTP client timeout. `0` disables that timeout. Does not control Lighthouse requests, which currently use the default HTTP client without an explicit timeout. |

## API TLS

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `ssl_cert` | path string | `""` | API server certificate PEM. Must be paired with `ssl_key`. |
| `ssl_key` | path string | `""` | API server private-key PEM. Must be paired with `ssl_cert`. |
| `auto_ssl_domain` | string | `""` | Enables automatic ACME certificates for this domain. Requires externally reachable HTTP-01 on port 80 and permission to bind that port. |
| `auto_ssl_cache_dir` | path string | `".certs"` | Writable cache for ACME certificates. Relevant when `auto_ssl_domain` is set. |

In the normal listener mode, automatic TLS takes precedence over a manual pair;
a supplied manual pair is still loaded at startup. Manual API certificate files
are polled every three seconds. This is distinct from etcd-client TLS, which is
loaded once per client. Experimental traffic splitting uses separate plaintext
listeners and expects TLS termination at the upstream proxy.

## Payment storage

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `payment_channel_storage_type` | string | `"etcd"` | Use `etcd` for persistent shared state or `memory` for in-process state. Unknown values currently fall back to memory; check spelling carefully. |
| `payment_channel_cert_path` | path string | `""` | Etcd HTTPS client certificate PEM. |
| `payment_channel_key_path` | path string | `""` | Matching etcd HTTPS client private-key PEM. |
| `payment_channel_ca_path` | path string | `""` | CA bundle used to verify etcd HTTPS servers. |

For HTTPS etcd endpoints the current wrapper requires all three TLS files, even
if the remote server would accept a client without a certificate. They configure
the **client**, not the embedded server or its peer connections.

### Etcd client

`payment_channel_storage_client` is a nested object. Its metadata base is the
selected organization group's `payment.payment_channel_storage_client`.

| Full key | Type | Local default | Meaning / precedence |
|----------|------|---------------|----------------------|
| `payment_channel_storage_client.endpoints` | URL array | absent | Nonempty local list replaces metadata endpoints. Omitted/`[]` retains metadata. At least one effective endpoint is required. |
| `payment_channel_storage_client.connection_timeout` | duration string | `"0s"` | Zero retains metadata; nonzero overrides. Passed as etcd `DialTimeout`, not a global connectivity deadline. |
| `payment_channel_storage_client.request_timeout` | duration string | `"0s"` | Zero retains metadata; nonzero overrides independently. Effective timeout must be positive; bounds individual storage RPCs, not a whole multi-step API call. |
| `payment_channel_storage_client.hot_reload` | boolean | `false` | Enables the organization-metadata event listener when an etcd client is present at startup. Not transport reconnection or TLS-file reload. |

The client is created without waiting for cluster availability. Reconnection is
handled by gRPC; there is no daemon-wide attempt count or retry deadline. See
[timeouts and reconnection](../etcddb/README.md#timeouts-and-reconnection).
Local overrides do not remove the startup requirement for valid organization
metadata when blockchain is enabled.

### Embedded etcd server

`payment_channel_storage_server` is a separate local object. No server settings
are taken from organization metadata, and enabling the server does not select it
as the client's destination.

| Full key | Type | Default | Meaning |
|----------|------|---------|---------|
| `payment_channel_storage_server.enabled` | boolean | `false` | Start a member in the daemon process. **Explicitly set this when supplying the section:** a section present in the file without `enabled` currently enables the server. |
| `payment_channel_storage_server.id` | string | `"storage-1"` | Member name, matching its entry in `cluster`. |
| `payment_channel_storage_server.scheme` | string | `"http"` | Client/peer URL scheme. `https` alone is insufficient: the wrapper does not expose server/peer TLS settings. |
| `payment_channel_storage_server.host` | string | `"127.0.0.1"` | Used for both listen and advertised URLs. |
| `payment_channel_storage_server.client_port` | integer | `2379` | Client port; not automatically allocated or copied into client endpoints. |
| `payment_channel_storage_server.peer_port` | integer | `2380` | Peer port; keep `cluster` URLs consistent. |
| `payment_channel_storage_server.token` | string | `"unique-token"` | Bootstrap cluster token, not an authentication password. Use a distinct token per independent cluster. |
| `payment_channel_storage_server.cluster` | string | `"storage-1=http://127.0.0.1:2380"` | Comma-separated `member=peer-URL` bootstrap list. |
| `payment_channel_storage_server.startup_timeout` | duration string | `"1m"` | Wait for server readiness before starting the daemon API. Use a positive duration. |
| `payment_channel_storage_server.data_dir` | path string | `"storage-data-dir-1.etcd"` | Writable persistent directory for this member. |
| `payment_channel_storage_server.log_level` | string | `"info"` | `debug`, `info`, `warn`, or `error`. |
| `payment_channel_storage_server.log_outputs` | string array | `["./etcd-server.log"]` | Log paths or `stdout` / `stderr`. |

See [etcd deployment examples](../etcddb/README.md). `enabled` controls member
startup independently of the chosen storage backend; selecting `memory` does not
automatically disable an explicitly enabled embedded server.

## Free calls and prepaid tokens

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `private_key_for_free_calls` | hex private-key string | `""` | Required for free-call token issuance. Must match the selected service group's `free_call_signer_address`. |
| `trusted_free_call_signers` | address array | two addresses below | Trusted token-request signers allowed to supply `user_id`, without the ordinary wallet balance check. An explicit `[]` clears the list; omission retains defaults. |
| `min_balance_for_free_call` | whole-token integer or integer string | `"10"` | Minimum ERC-20 balance for ordinary wallet token requests, converted to base units using 18 decimals. Use a nonnegative whole number. |
| `free_calls_per_address` | object keyed by address | `{}` | Address-specific quota. Positive integers override metadata; missing/`0` uses metadata; `-1`, `"unlimited"`, or `"infinity"` means unlimited. Keys are case-insensitive. |
| `token_expiry_in_minutes` | integer minutes | `1440` | Prepaid JWT lifetime. Use a positive value; not the lifetime of free-call tokens. |
| `token_secret_key` | string | public test value below | Prepaid JWT signing secret. Blockchain-enabled validation requires at least 32 bytes. Replace the public test default for production and share the intended secret between replicas validating the same tokens. |

The built-in trusted addresses are
`0x3Bb9b2499c283cec176e7C707Ecb495B7a961ebf` and
`0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F`.
The built-in JWT test value is `test-secret-key-at-least-32-bytes-long`.
It passes the length check but is public, not a safe production credential.
The daemon does not automatically generate a replacement secret.

For `free_calls_per_address`, a trusted signer's address is the lookup key even
when requests carry different user IDs; usage records distinguish those IDs.
See [free calls](../freecall/README.md) for token signatures and block-based expiry,
and [escrow](../escrow/README.md) for prepaid JWTs.

## Rate limiting

| Key | Type | Effective default | Meaning |
|-----|------|-------------------|---------|
| `rate_limit_per_minute` | number or numeric string | unlimited | Token refill rate per minute; fractional rates are supported. Missing, `0`, or unparsable input currently disables limiting. Use positive finite values for a limit. |
| `burst_size` | integer | `2147483647` | Bucket capacity. Missing/`0` becomes `math.MaxInt32`; ignored for unlimited rate. Set a positive capacity together with a finite rate. |

The bucket starts full and excess calls are rejected rather than queued. This
is not a per-wallet quota. See [rate limiting](../ratelimit/README.md).

## Metering, heartbeat, and notifications

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `metering_enabled` | boolean | `false` | Enable usage reporting. Startup registration also requires blockchain to be enabled. |
| `metering_endpoint` | URL string | `""` | Base URL; required and validated when metering is enabled. Startup uses `/metering/verify`, reporting uses `/metering/usage`. |
| `private_key_for_metering` | hex private-key string | `""` | Required by metering registration; signs usage requests. Also used by the signed notification publication path. |
| `service_heartbeat_type` | string | `""` | `""`, `none`, and `tcp` check `service_endpoint` over TCP. `grpc`, `http`, and `https` use `heartbeat_endpoint`. |
| `heartbeat_endpoint` | string | `""` | Endpoint for explicit health checks. Validation requires an HTTP(S) URL with a host; the gRPC path currently has a URL/target mismatch described in the metrics guide. Empty does not force a successful heartbeat. |
| `alerts_email` | string | `""` | Notification recipient. Empty disables notification configuration; it does not disable logger email hooks. |
| `notification_endpoint` | URL string | `""` | Notification URL. Both this and a valid `alerts_email` are needed to enable notification configuration. Not required merely because metering is enabled. |

Heartbeat is a reachability/health probe, not an etcd readiness check. TCP checks
have a ten-second connection timeout; gRPC health calls have a one-second RPC
deadline; the HTTP health path currently uses no explicit client timeout.
Notification publication uses the metering signing path; it is not a generic
unsigned webhook. See [metrics and monitoring](../metrics/README.md).

## Logging

`log` is a nested object. Full leaf keys are listed here; configuration examples
and operational details are in [logger/README.md](../logger/README.md).

| Full key | Type | Default | Meaning |
|----------|------|---------|---------|
| `log.level` | string | `"info"` | `debug`, `info`, `warn` (`warning` alias), `error`, `panic`, or `fatal`. Panic/fatal records bypass this threshold. |
| `log.timezone` | string | `"UTC"` | Go time-zone name for log-entry timestamps, not a control for all file-name timestamps. |
| `log.formatter.type` | string | `"text"` | `text` or `json`. |
| `log.formatter.timestamp_format` | string | `"2006-01-02T15:04:05.999Z07:00"` | Go time layout. Explicit `""` selects RFC3339. |
| `log.output.type` | string array | `["file", "stdout"]` | `file`, `stdout`, and/or `stderr`; must not be empty. Panic/fatal also go to stderr if not already selected. |
| `log.output.file_pattern` | string | `"./snet-daemon.%Y%m%d.log"` | Name computed at logger initialization. Only `%Y`, `%m`, `%d`, `%H`, `%M`, `%S` are supported. |
| `log.output.current_link` | path string | `"./snet-daemon.log"` | Symlink to the current file; explicit `""` disables link creation. Requires filesystem permissions for symlinks. |
| `log.output.max_size_in_mb` | integer | `10` | Size-based rotation threshold. `0` selects Lumberjack's 100 MiB fallback, not unlimited size. |
| `log.output.max_age_in_days` | integer | `7` | Backup retention based on encoded rotation time; `0` removes the age limit. |
| `log.output.rotation_count` | integer | `0` | Maximum retained backup count; `0` removes the count limit, not rotation or age cleanup. |
| `log.hooks` | string array | `[]` | Names of hook objects under `log`, such as `log.mail`. |

Hook fields use `log.<name>.<field>`. There are no built-in hook definitions.

| Field | Type | Requirement / effective default |
|-------|------|---------------------------------|
| `type` | string | Required: `email` or `telegram_bot`. |
| `levels` | string array | Required, nonempty list of exact log levels; not a minimum severity threshold. |
| `host` | string | Required for email. SMTP hostname. |
| `port` | integer | Required, nonzero for email. |
| `from` | string | Required for email. |
| `to` | string | Required for email. |
| `username` | string | For email; omitted/empty falls back to `from`. |
| `password` | string | Required, nonempty for email. Treat as a secret. |
| `application_name` | string | Email authentication identity; default `""`. |
| `telegram_api_key` | string | Required, nonempty for Telegram. Treat as a secret. |
| `telegram_chat_id` | integer | Required, nonzero for Telegram; negative group IDs are allowed. |
| `telegram_disable_notifications` | boolean | Default `false`; enables silent Telegram delivery when true. |

`disable_notification` is the Telegram API payload field, **not** a daemon config
key. Invalid optional hooks are reported and skipped. Hook levels do not bypass
the logger threshold for ordinary records. File backups are gzip-compressed by
the wrapper; there is no configurable compression flag.

## Training and dynamic pricing

| Key | Type | Default | Meaning / constraints |
|-----|------|---------|-----------------------|
| `model_training_enabled` | boolean | `false` | Enables training when blockchain and the required service-proto training metadata are also available. |
| `model_maintenance_endpoint` | string | `""` | Training-provider endpoint; omitted/empty falls back to `service_endpoint`. |
| `enable_dynamic_pricing` | boolean | `false` | Uses method-specific dynamic pricing when its mapping exists in service proto metadata; otherwise normal metadata pricing is used. |

See [training](../training/README.md). Method options such as dataset limits and
maximum models belong in the service protobuf schema, not in this daemon JSON.

## Experimental listeners

The entire `experimental` object is absent by default. These settings affect the
normal `grpc` daemon mode and should be tested with the deployment's proxy/TLS
topology before use.

| Full key | Type | Default | Meaning |
|----------|------|---------|---------|
| `experimental.split_grpc_web` | boolean | `false` | Separate gRPC-Web matching inside the shared-port multiplexer; does not allocate a new TCP port. |
| `experimental.use_original_cmux` | boolean | `false` | Use upstream cmux instead of the fork. |
| `experimental.traffic_split.grpc` | unsigned integer | no listener override | gRPC listener port when `traffic_split` is present. |
| `experimental.traffic_split.http` | unsigned integer | no listener override | HTTP/gRPC-Web listener port when `traffic_split` is present. |

With `traffic_split` present, configure **both** ports explicitly; omitted values
become `0` (OS-selected ports), not the daemon's usual port. The host comes from
`daemon_endpoint`, but its port is replaced. This path bypasses cmux and the normal
API TLS setup; terminate TLS at your upstream proxy. The other experimental mux
flags do not control this path.

## Deprecated and unsupported names

The following old names are migrated during validation. Prefer only the new name:

| Deprecated key | Replacement |
|----------------|-------------|
| `daemon_end_point` | `daemon_endpoint` |
| `ipfs_end_point` | `ipfs_endpoint` |
| `passthrough_endpoint` | `service_endpoint` |
| `metering_end_point` | `metering_endpoint` |
| `heartbeat_svc_end_point` | `heartbeat_endpoint` |
| `notification_svc_end_point` | `notification_endpoint` |
| `pvt_key_for_metering` | `private_key_for_metering` |
| `pvt_key_for_free_calls` | `private_key_for_free_calls` |

`monitoring_enabled`, `monitoring_svc_end_point`, `is_curation_in_progress`,
`curation_address_for_validation`, and `allowed_users` are not supported aliases.
Use the documented metering and access-control keys. The loader does not reject
every unknown key, so successful parsing alone does not prove a setting is used.

## Examples and maintenance

Start with `snetd init`, then apply the relevant examples from
[etcd](../etcddb/README.md), [logging](../logger/README.md),
[metrics](../metrics/README.md), and [Docker](../docker/README.md).
These examples are fragments or deployment templates, not universally valid
production configurations. Never commit real private keys, JWT secrets, SMTP
passwords, or bot tokens.

When adding a setting, update this reference, its consumer's guide, and tests for
its default, validation, and override behavior. Keep examples valid JSON and
distinguish runtime defaults from network/metadata-derived values.
