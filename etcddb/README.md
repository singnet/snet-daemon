# etcd storage

The daemon uses etcd as shared atomic storage for payment channel state and other
storage-backed services. Configure `payment_channel_storage_type` in the daemon
configuration:

| Storage type | Use |
|--------------|-----|
| `etcd` (default) | Shared, persistent storage for service replicas using the same cluster. |
| `memory` | In-process storage for tests or a single replica; state is lost when the process exits. |

For replicas sharing payment state, select `etcd` and configure the same cluster
endpoints. Client settings and embedded server settings are separate: using an
external cluster does not require starting an embedded server.

Unknown storage-type values currently fall back to memory with a warning, so a
typo can silently change persistence expectations if the warning is missed.
Use an explicit supported value. The complete parameter list is in the
[configuration reference](../config/README.md#payment-storage).

### Deployment modes

The daemon can connect to an external etcd cluster or run an etcd member in its
own process through the official [etcd embed package](https://etcd.io/docs/v3.7/dev-guide/golang_embed_pkg/).
Embedded mode uses the same etcd server implementation, including persistent
storage and cluster consensus.

| Mode | Operational differences |
|------|-------------------------|
| Embedded | Runs the daemon and an etcd member as one process, reducing the number of separately deployed services. Stopping or restarting that daemon also stops or restarts its etcd member. |
| External | Runs etcd separately, allowing the storage cluster to be maintained and upgraded independently of the daemon replicas. It requires a separate deployment for etcd. |

Choose the mode according to the deployment topology and required configuration
features. The current default `payment_channel_storage_server.enabled: false`
only disables automatic startup of an embedded member; it does not mark embedded
mode as deprecated or prescribe an external cluster for production. The embedded
wrapper's available configuration, including its current TLS limitations, is
described [below](#embedded-server-configuration).

The JSON examples below are configuration fragments to merge into the daemon
configuration. The payment storage setup assumes blockchain is enabled and valid
organization metadata is available.

## Server and client configuration are independent

The daemon configures the embedded server and the storage client separately:

- `payment_channel_storage_server` uses the local daemon configuration and server
  defaults. Organization metadata does not supply embedded server settings.
- `payment_channel_storage_client` starts with endpoints and timeouts from
  organization metadata, then applies local overrides as described
  [below](#client-configuration). This precedence is the same whether the
  embedded server is enabled or disabled.

`payment_channel_storage_server.enabled` only controls whether an embedded member
starts. It does not select that member as the client's destination, change client
configuration precedence, or copy server addresses into client endpoints.
Server startup is also independent of `payment_channel_storage_type`: selecting
`memory` does not disable an explicitly enabled embedded server.
Consequently, an enabled embedded server can coexist with a client configured to
use an external cluster. Disabling the embedded server also does not disable
local client overrides or force the client to use metadata endpoints.

To use the embedded member for storage, configure client endpoints that reach its
client URL. They can come from metadata or a local override. The daemon currently
does not automatically choose free embedded server ports or synchronize client
endpoints with server settings.

## Client configuration

The client starts with settings from the organization metadata group selected by
`daemon_group_name`. The group's `payment.payment_channel_storage_client` supplies
endpoints and timeouts. The local `payment_channel_storage_client` section can
override them, as implemented in [etcddb_conf.go](etcddb_conf.go):

| Field | Meaning | Local default / precedence |
|-------|---------|----------------------------|
| `connection_timeout` | Passed to the etcd client's `DialTimeout`; see [timeouts](#timeouts-and-reconnection). | Omitted or `"0s"`: retain the metadata value. A nonzero local value overrides it independently of `request_timeout`. |
| `request_timeout` | Deadline for each etcd RPC issued by the storage wrapper. | Omitted or `"0s"`: retain the metadata value. A nonzero local value overrides it independently of `connection_timeout`; the effective value must be positive. |
| `endpoints` | Array of etcd client URLs, such as `http://etcd-1:2379`. | Omitted or `[]`: retain metadata endpoints. A nonempty local array replaces the entire metadata list; the lists are not merged. At least one effective endpoint is required. |
| `hot_reload` | Enables the organization-metadata event listener during daemon startup. | `false`. This concerns metadata updates, not transport reconnection or reloading TLS files. |

Use the config file for this nested object. The daemon does not translate
underscore-separated ENV names to nested keys, and endpoints known only through
ENV may be omitted when constructing the client subconfiguration. See
[environment-variable limitations](../config/README.md#environment-variables).

To use the cluster and timeouts from organization metadata:

```json
{
  "payment_channel_storage_type": "etcd",
  "payment_channel_storage_client": {
    "connection_timeout": "0s",
    "request_timeout": "0s",
    "hot_reload": false
  },
  "payment_channel_storage_server": {
    "enabled": false
  }
}
```

To override the client settings locally for a three-member external cluster:

```json
{
  "payment_channel_storage_type": "etcd",
  "payment_channel_storage_client": {
    "connection_timeout": "5s",
    "request_timeout": "3s",
    "endpoints": [
      "http://etcd-1:2379",
      "http://etcd-2:2379",
      "http://etcd-3:2379"
    ],
    "hot_reload": false
  },
  "payment_channel_storage_server": {
    "enabled": false
  }
}
```

Replace the example hostnames with addresses reachable from each daemon replica.
These are client URLs, not the peer URLs used to bootstrap the etcd cluster.

### Timeouts and reconnection

The client is created without waiting for the external cluster to become
available. gRPC normally starts connecting on the first etcd request. Configuration
parsing and loading TLS files still happen during client initialization and can
fail it. An unreachable first endpoint does not prevent using another available
endpoint.

`request_timeout` bounds one etcd RPC, including waiting for connectivity. For
example, with `"request_timeout": "3s"`, a `Get` or `Put` returns an error if it
cannot complete within three seconds. That deadline applies to this request;
later requests get their own deadlines and can use the same client after etcd
becomes available. A multi-step API operation or a transaction retry loop can
issue several RPCs, so this setting is not a deadline for the entire API request.

The daemon does not set a global time limit or attempt count for establishing a
connection. gRPC handles reconnection with increasing backoff. Its current
defaults start at one second, multiply the base delay by 1.6 up to 120 seconds,
and apply random jitter of up to 20%. Connection attempt durations and available
endpoints also affect timing. Recovery can wait until the next connection attempt;
it need not be immediate when etcd starts. An idle client can stop connecting until
another request arrives.

`connection_timeout` is a compatibility name for etcd's `DialTimeout`. In the
currently used etcd client v3.7.2, this field does **not** bound gRPC connection
establishment or the total reconnection period. The library uses it for certain
client operations, including authentication and optional cluster-version checks,
and to derive the initial lease keepalive timeout. This wrapper does not enable
the authentication or version-check options; the lease keepalive use applies when
a mutex session is created. Setting it to `"5s"` does not make the client give up
connecting after five seconds.

See the [etcd client configuration](https://pkg.go.dev/go.etcd.io/etcd/client/v3@v3.7.2#Config),
[gRPC client initialization](https://pkg.go.dev/google.golang.org/grpc@v1.84.0#NewClient),
and [gRPC connection backoff](https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md).

Lease sessions for `NewMutex` are created on demand. The initial lease grant uses
`request_timeout`, while session keepalive follows the client's lifetime. Closing
the client cancels a pending lease grant and stops the TLS certificate monitor.

API startup does not establish that payment storage is ready. While the cluster
is unreachable or lacks quorum, storage operations can fail. The existing
heartbeat is not an etcd readiness check.

## HTTPS client configuration

For HTTPS endpoints, configure these paths at the top level of the daemon
configuration:

```json
{
  "payment_channel_storage_client": {
    "endpoints": ["https://etcd-1:2379"]
  },
  "payment_channel_cert_path": "/etc/singnet/etcd/client.pem",
  "payment_channel_key_path": "/etc/singnet/etcd/client-key.pem",
  "payment_channel_ca_path": "/etc/singnet/etcd/ca.pem"
}
```

All three paths default to empty and are required by the current HTTPS wrapper,
even if the server itself does not require a client certificate. The client
certificate and private key must form a valid pair. The CA file contains
the certificates used to verify the etcd servers. These client settings do not
configure TLS for an embedded server or its peer connections.

### Certificate expiry notifications

For HTTPS etcd connections, the daemon checks its loaded client certificate,
the rest of the client certificate chain, and all certificates in
`payment_channel_ca_path` at client initialization and every hour thereafter.
It logs a `warn` when a certificate has 30 days or less remaining, and an `error`
when it has expired. Repeated alerts for the same certificate and severity are
limited to once every 24 hours per client instance. Expiry is reported immediately
on the next check even if a warning was recently emitted. Initial alerts are also
emitted when creating a new client, including after a restart or reconnect.

To receive these alerts by email or Telegram, configure the existing
[log hooks](../logger/README.md) for both `warn` and `error`, and use a
`log.level` of `warn` or lower. Without a hook the alerts remain in the logs.
The message includes the certificate path, subject, SHA-256 fingerprint and
UTC expiry time; private keys and certificate contents are never included.

Checks do not depend on heartbeat requests and do not change the HTTP/gRPC
heartbeat response or status. They do not perform extra network requests or
change TLS verification. HTTP-only etcd connections do not start a monitor.

The monitor checks the certificates loaded into the running client, not just
the files on disk. After renewing certificates, recreate the etcd client or
restart the daemon to load them. Merely replacing a PEM file does not update
the current client's TLS configuration. This feature does not monitor remote
etcd server or peer certificates, and does not renew certificates automatically.

## Embedded server configuration

`payment_channel_storage_server.enabled` defaults to `false`, so the daemon does
not start an etcd member unless configured to do so. To run a member inside the
daemon process, explicitly set it to `true` and configure
`payment_channel_storage_server`:

| Field | Meaning | Default |
|-------|---------|---------|
| `enabled` | Start an embedded etcd member. | `false` when the server section is absent. |
| `id` | Member name; must match its entry in `cluster`. | `"storage-1"` |
| `scheme` | URL scheme for client and peer URLs. | `"http"` |
| `host` | Host used in both listen and advertised client/peer URLs. | `"127.0.0.1"` |
| `client_port` | Port for etcd client requests. | `2379` |
| `peer_port` | Port for communication between etcd members. | `2380` |
| `token` | Shared bootstrap token; use a distinct token for each separate cluster. | `"unique-token"` |
| `cluster` | Comma-separated member names and full peer URLs. | `"storage-1=http://127.0.0.1:2380"` |
| `startup_timeout` | How long to wait for the embedded member's readiness notification. | `"1m"` |
| `data_dir` | Persistent data directory for this member. | `"storage-data-dir-1.etcd"` |
| `log_level` | Embedded server logging level: `debug`, `info`, `warn`, or `error`. | `"info"` |
| `log_outputs` | Array of log file paths or `"stderr"` / `"stdout"`. | `["./etcd-server.log"]` |

Always set `enabled` explicitly when supplying a server section. The current
configuration loader treats a section present in the configuration file with
`enabled` omitted as enabled.

Relative data/log paths resolve from the daemon's working directory, not the
config-file directory. Keep the member's data directory on persistent writable
storage and do not share one directory between running members. The bootstrap
`token` distinguishes clusters; it is not an authentication credential.

Example for a single embedded member with a local client:

```json
{
  "payment_channel_storage_type": "etcd",
  "payment_channel_storage_client": {
    "connection_timeout": "5s",
    "request_timeout": "3s",
    "endpoints": ["http://127.0.0.1:2379"],
    "hot_reload": false
  },
  "payment_channel_storage_server": {
    "enabled": true,
    "id": "storage-1",
    "scheme": "http",
    "host": "127.0.0.1",
    "client_port": 2379,
    "peer_port": 2380,
    "token": "local-storage-cluster",
    "cluster": "storage-1=http://127.0.0.1:2380",
    "startup_timeout": "1m",
    "data_dir": "storage-data-dir-1.etcd",
    "log_level": "info",
    "log_outputs": ["./etcd-server.log"]
  }
}
```

In this example, the local client endpoint explicitly matches the embedded
member's `scheme`, `host`, and `client_port`. Changing `client_port` requires
updating any client endpoints that refer to this member. Changing `peer_port`
requires updating its peer URL in the bootstrap `cluster` configuration. These
values are not synchronized automatically; client endpoints must use the client
port, not the peer port.

The embedded server starts before the daemon's HTTP/gRPC API and waits for
readiness. A startup error or expiration of `startup_timeout` aborts startup.
Asynchronous initialization of the client does not bypass this server readiness
wait. Once ready, the embedded server and daemon API run concurrently on their
separate endpoints. The default daemon API port is `8080`; etcd client and peer
ports are `2379` and `2380`.

For several embedded members, give each member its own `id`, reachable `host`,
and data directory. Members on the same host also need distinct client and peer
ports. Use the same `cluster` list and `token` on all members, for example:

```text
storage-1=http://127.0.0.1:2380,storage-2=http://127.0.0.1:2480,storage-3=http://127.0.0.1:2580
```

In that example, member peer ports are `2380`, `2480`, and `2580`; client ports
could be `2379`, `2479`, and `2579`. Configure client endpoints separately with
those client URLs. Loopback addresses are suitable for members on one host;
use reachable host addresses for a cluster spanning several machines.

Start the members together so a quorum can form while each daemon waits for its
embedded server. The wrapper bootstraps with the etcd `new` cluster state and
does not expose server or peer TLS certificate settings. Changing `scheme` to
`https` alone does not configure server-side TLS. For deployment and cluster
membership management, see the [etcd clustering guide](https://etcd.io/docs/v3.7/op-guide/clustering/).

## Tests

Run commands from the repository root. The tests start local embedded etcd
servers; an external etcd cluster is not required.

Run the complete package tests, optionally with the race detector:

```sh
go test ./etcddb -count=1 -timeout 3m
go test -race ./etcddb -count=1 -timeout 3m
```

The three-member quorum test verifies writes and reads with all members running,
continued access after stopping the leader, and failed writes and linearizable
reads after stopping a second member. It uses dynamically allocated local ports
and temporary data directories:

```sh
go test ./etcddb -run '^TestEtcdClusterQuorum$' -count=1 -timeout 1m
```

The asynchronous client tests verify recovery when etcd starts after the client
and cancellation of a pending lease request when the client closes:

```sh
go test ./etcddb -run '^TestEtcdClient(ConnectsAfterServerStarts|CloseCancelsPendingSession)$' -count=1 -timeout 1m
```

These tests also run as part of `go test ./...`. Some existing single-member
suite tests use ports `2379` and `2380`, so keep those ports free when running
the complete package suite.
