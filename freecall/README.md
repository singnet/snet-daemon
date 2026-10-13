# Free calls

The `freecall` package owns free-call token validation, user allowances, usage
transactions, storage, and gRPC services. Both `freecall` and `escrow` plug into
`handler.StreamPaymentHandler`; neither package imports the other. Atomic locks
and gob encoding are shared through `storage`, and payment error codes through
`errs`.

The protobuf source is `freecall/state_service.proto`. Its `escrow` namespace,
message fields, and `/escrow.FreeCallStateService/...` method names are retained
for existing clients. Go callers use the generated types and registration
functions from `freecall`. Generate the bindings with `go generate ./...`.

The generation directive uses the module root as the proto include path so the
descriptor is named `freecall/state_service.proto`. Using `-I .` here would name
it `state_service.proto`, conflicting with escrow's descriptor in Go's global
protobuf registry. `go generate` always runs commands in the package's source
directory, regardless of the caller's working directory.

Existing data uses `/free-call-user/storage` and locks use `/freecall/lock`.
Their keys, gob field names, and lock values are unchanged; no storage migration
is required. Compatibility tests exercise both the original protobuf schema and
records encoded with the original `FreeCallUserData` layout.

## Configuration and quotas

See the [configuration reference](../config/README.md#free-calls-and-prepaid-tokens)
for all defaults and precedence rules. Free-call token issuance requires
blockchain-enabled operation and `private_key_for_free_calls` matching the selected
service group's `free_call_signer_address`. An empty key (the default) or signer
mismatch disables the state/token service.

Ordinary wallet token requests require a balance of at least
`min_balance_for_free_call` (default: 10 whole ERC-20 tokens, using 18 decimals).
They cannot supply a nonempty `user_id`. Addresses in `trusted_free_call_signers`
may provide that ID and bypass this balance check. Omitting the list retains the
two built-in trusted addresses; set `[]` explicitly to trust none.

`free_calls_per_address` defaults to `{}`. Positive integer values override the
service-metadata quota; missing entries and `0` fall back to metadata. `-1`,
`"unlimited"`, and `"infinity"` mean unlimited. Address keys are case-insensitive.
For trusted callers, the quota lookup uses the signer's address, while usage is
tracked separately for each address/user-ID/organization/service/group identity.
These are usage allowances, not per-minute rate limits. Unlimited availability
is represented by the current state RPC as `99999999`.

## Tokens and requests

Use [state_service.proto](state_service.proto) for `GetFreeCallToken` and
`GetFreeCallsAvailable`. The signatures use the hash/signing convention described
in [escrow](../escrow/README.md#signature-encoding), with these payloads:

```text
Token request:
"__prefix_free_trial" || AddressText || UserID || OrganizationID || ServiceID || GroupID || CurrentBlock

Availability request or service call:
"__prefix_free_trial" || AddressText || UserID || OrganizationID || ServiceID || GroupID || CurrentBlock || Token
```

Here the address and identifiers are their exact UTF-8 strings, the current
block is a 32-byte big-endian integer, and `Token` is the full raw token byte
sequence. The signature must recover to the supplied address. `GroupID` is the
selected metadata group's ID string, not `daemon_group_name`. Current-block
freshness is checked within five blocks of the daemon's latest observed block.

The daemon-issued token contains a 65-byte signature, an underscore byte, and
the decimal expiration block. Its signature covers organization ID, daemon
group **name**, the caller's 20-byte address, user ID, and the 32-byte expiration
block; see [validation.go](validation.go). Use the returned `token` bytes, or
hex-decode `token_hex` before sending binary metadata.

`token_lifetime_in_blocks` defaults to 172800 blocks. A supplied value at most
172800 is used; larger values currently fall back to that default rather than
returning a validation error. Expiration is calculated from the daemon's current
block, not a fixed number of seconds. `token_expiry_in_minutes` controls separate
prepaid JWTs and does not change free-call token lifetime.

For a service call, send:

| gRPC metadata | Value |
|---------------|-------|
| `snet-payment-type` | `free-call` |
| `snet-free-call-user-address` | Caller/signer address. |
| `snet-free-call-user-id` | Optional ID for trusted callers. |
| `snet-current-block-number` | Current block as a decimal string. |
| `snet-payment-channel-signature-bin` | Caller signature bytes. |
| `snet-free-call-auth-token-bin` | Full raw token bytes, including its expiration suffix. |

Do not send a separate `snet-free-call-token-expiry-block` header: the current
handler extracts expiration from the token. If `allowed_user_flag` is enabled,
the caller must also pass the separate `allowed_user_addresses` check.

## Java SDK migration

This extraction preserves the RPC wire format, but changes the Java source API
when bindings are regenerated. Free-call messages previously nested inside
`io.singularitynet.daemon.escrow.StateService` now belong to `FreeCallState` in
the same Java package. For example, `StateService.FreeCallToken` becomes
`FreeCallState.FreeCallToken`. Payment-channel messages remain in `StateService`;
the free-call gRPC stub remains `FreeCallStateServiceGrpc`.

The explicit outer class name prevents both proto files from generating the
same `StateService.java` output. Removing it does not restore compatibility:
the two files can no longer be generated together without an output collision.

Consumers adopting the new schema must:

- Include `freecall/state_service.proto` as well as the escrow proto files,
  preserving their directory paths because both packages contain a file named
  `state_service.proto`.
- Regenerate message and gRPC bindings and update free-call imports from
  `StateService` to `FreeCallState`.
- Rebuild and test the SDK and its consumers before releasing the new bindings.

When updating a Java SDK, check both its free-call imports and schema download
task: consumers still using `StateService.*` or downloading only the old escrow
schemas need the coordinated changes above. Existing clients do not need new
bindings solely because Java class names changed; wire compatibility and Java
source compatibility are separate concerns.
