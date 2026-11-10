# Metrics, heartbeat, and notifications

These are separate mechanisms:

- **Heartbeat:** pull-based health/reachability checks exposed by the gRPC daemon.
- **Metering:** signed usage reporting to a configured service; disabled by default.
- **Heartbeat notifications:** signed publication of heartbeat failures to a
  configured notification service.
- **Logger hooks:** independent email/Telegram delivery of selected log levels;
  see [logger](../logger/README.md).

For all keys and defaults, see [configuration](../config/README.md#metering-heartbeat-and-notifications).

## Heartbeat

In the normal `daemon_type: grpc` mode, the daemon exposes HTTP
`GET /heartbeat` and gRPC `grpc.health.v1.Health/Check`. These share the daemon
listener with the service API unless experimental traffic splitting is enabled.
The simple `daemon_type: http` proxy does not register these auxiliary routes.

The daemon probes its backend when asked for a heartbeat; it does not periodically
push heartbeats to a monitoring server. This is not an etcd quorum/readiness check.

| `service_heartbeat_type` | Probe target | Behavior |
|--------------------------|--------------|----------|
| Omitted, `""`, `none`, or `tcp` | `service_endpoint` | Attempt a TCP connection, with a ten-second connection timeout. `none` does not disable the probe. |
| `grpc` | `heartbeat_endpoint` | gRPC health `Check`, with `service_id` as the service name and a one-second RPC deadline. The current client uses insecure transport. |
| `http` or `https` | `heartbeat_endpoint` | HTTP GET; requires status 200 and a nonempty response body. Uses the default HTTP client without an explicit timeout. |

The config default is `service_heartbeat_type: ""` and `heartbeat_endpoint: ""`.
An empty health endpoint is appropriate for TCP mode. Explicit gRPC/HTTP(S) modes
need a usable endpoint: although validation permits an empty value, the request
path does not turn it into an unconditional success or a TCP fallback.

There is currently a mismatch for gRPC endpoints: startup validation accepts
only HTTP(S) URLs with a host, but the gRPC health client passes the value directly
to `grpc.NewClient` without converting it to a gRPC target. A normal
`dns:///host:port` target fails that validation, while an `http://host:port` URL
is not normalized for dialing. Prefer TCP or HTTP(S) health checks until that
path is corrected. For HTTP(S), use a complete URL.

### Interpreting the response

HTTP heartbeat returns JSON containing `daemonID`, `timestamp`, `status`,
`serviceheartbeat`, `daemonVersion`, `blockchainEnabled`, `blockchainNetwork`,
and `storageClientCertDetails`; training metadata can also be included.
`serviceheartbeat` is itself a JSON-encoded string containing service ID/status.

The outer result is `Online` when the probe returns no error and `Offline` on
a probe error. Important current limitations:

- TCP success proves an open port, not application readiness.
- HTTP success checks transport/status/body presence, not a health status field
  inside the response body.
- The gRPC path currently maps a successful RPC to `Online` even when the
  backend's returned health enum is not `SERVING`.
- The HTTP handler does not use its HTTP status code as a readiness signal.
- Missing payment-storage certificate files can produce log errors while
  collecting certificate details; this does not itself make storage ready or
  determine backend health.

Consumers must not interpret this endpoint as a comprehensive payment/storage
readiness check. The current gRPC health `List` method is unimplemented and
`Watch` does not stream health updates.

### HTTP backend health example

Merge this fragment into the daemon config and implement the endpoint in the backend:

```json
{
  "service_heartbeat_type": "http",
  "heartbeat_endpoint": "http://127.0.0.1:5000/health"
}
```

For the default TCP probe, no dedicated health RPC/URL is required:

```json
{
  "service_heartbeat_type": "",
  "heartbeat_endpoint": ""
}
```

## Metering

`metering_enabled` defaults to `false`. The old names `monitoring_enabled` and
`monitoring_svc_end_point` are not supported aliases.

To enable reporting, supply:

- `metering_enabled: true`, with blockchain enabled.
- `metering_endpoint`: the metering service base URL.
- `private_key_for_metering`: the daemon's signing key, registered/recognized by
  the metering service.

Startup verifies the daemon using `<metering_endpoint>/metering/verify`. Usage
reporting posts to `<metering_endpoint>/metering/usage`.
Registration failure can prevent startup; metering is not enabled simply by
providing a URL.

Daemon ID is currently the lowercase hex SHA-256 of the concatenation of
organization ID, service ID, group ID string, and Registry contract address.
It does not include `daemon_endpoint`, so replicas of the same service/group/network
can have the same ID. See [register.go](register.go).

Usage publication sends `X-Daemonid`, `X-Token`, `X-username`,
`X-Organizationid`, `X-Groupid`, `X-Serviceid`, `X-Currentblocknumber`, and
`X-Signature`. The signature covers the `_usage` prefix, user name, organization,
service, group, and the current block. See [utils.go](utils.go) and
[response_stats.go](response_stats.go) for the payload and encoding.
Do not reuse the older registration-only `Access-Token` example for usage requests.

## Heartbeat notifications

Both `alerts_email` and `notification_endpoint` default to empty, disabling
notification configuration. If both are supplied, startup validates their formats.
They are not mandatory merely because metering is enabled.

A failed heartbeat can publish a notification containing recipient, message,
details, component ID, timestamp, type, and level. Publication uses the same
signed request path as metering, including `private_key_for_metering` and the
current block. This is not an arbitrary unsigned webhook interface; configure
a compatible receiver and signing setup. There is no general promise that every
daemon warning/error is sent to this endpoint.

For general operational alerts, configure logger hooks instead. In particular,
etcd certificate-expiry alerts are emitted as `warn`/`error` logs and can be
delivered by those hooks; see [etcd certificate monitoring](../etcddb/README.md#certificate-expiry-notifications).

## Operational notes

`service_timeout` controls the gRPC proxy's backend request deadline, not these
monitoring clients. The HTTP heartbeat and usage/notification clients currently
have no explicit client timeout. Configure network reachability and external
monitoring timeouts accordingly; do not assume the backend request timeout also
bounds monitoring work.

Tests use local servers and mocks:

```sh
go test ./metrics -count=1
```
