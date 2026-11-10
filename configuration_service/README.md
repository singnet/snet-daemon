# Configuration service

The gRPC `configuration_service.ConfigurationService` API provides selected
configuration values and pause/resume commands for the operator UI. It is not a
general-purpose live configuration editor.

See [configuration_service.proto](configuration_service.proto) for the wire
schema and [configuration_service.go](configuration_service.go) for the current
implementation.

## Methods and implementation status

| RPC | Current behavior |
| --- | --- |
| `GetConfiguration` | Authenticates the caller and returns a filtered configuration map plus the UI schema. |
| `StopProcessingRequests` | Authenticates and broadcasts a pause command to subscribers. |
| `StartProcessingRequests` | Authenticates and broadcasts a resume command to subscribers. |
| `UpdateConfiguration` | Authenticates, then returns a “work in progress” error. Does not update the configuration. |
| `IsDaemonProcessingRequests` | Authenticates, then returns a “work in progress” error. |

Pause/resume is handled by subscribers such as the stream rate-limit
interceptor. It does not stop the daemon process, close all APIs, or provide a
graceful drain of in-flight requests. Response status values acknowledge the
command; they are not a count or snapshot of active requests.

`GetConfiguration` only includes keys selected by `config.DisplayKeys`, with
values converted to strings. It is not a complete config-file export. The UI
schema currently covers only a subset of settings and contains historical
defaults; use [config/README.md](../config/README.md) and runtime defaults in
[config.go](../config/config.go) as the configuration reference. The response's
signature field is not populated by this implementation.

## Authorization

Set the top-level `authentication_addresses` array to the Ethereum addresses
allowed to call this service. It defaults to an empty list: the daemon may
start, but no caller is authorized. Invalid address entries are logged and
skipped. This allowlist is separate from payment `allowed_user_addresses` and
`trusted_free_call_signers`.

Each request includes `CallerAuthentication` with a current block and signature.
The signed message is the concatenation of:

1. The exact method prefix: `_GetConfiguration`,
   `_StopProcessingRequests`, `_StartProcessingRequests`,
   `_UpdateConfiguration`, or `_IsDaemonProcessingRequests`.
2. The current block number as a 32-byte, big-endian unsigned integer.

The address is recovered from the signature; it is not appended to this message.
Use the [Ethereum message-signing convention](../escrow/README.md#signature-encoding)
implemented by `utils.GetSignature` / `utils.GetSignerAddressFromMessage`.

When blockchain is enabled, the block number must be within five blocks of the
daemon's latest observed block. Disabling blockchain skips only that freshness
check, not signature verification or the address allowlist. Always supply the
authentication message; there is no anonymous operator mode.


