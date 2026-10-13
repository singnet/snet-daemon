# Payment channels and authorization

The `escrow` package owns payment-channel state, paid-call validation,
claim preparation, and prepaid JWT tokens. Free-call logic and its protobuf
schema live in [freecall](../freecall/README.md).

The protocols below describe blockchain-enabled operation. With blockchain
disabled, the replacement channel-state service returns an empty
`ChannelStateReply` without authenticating the supplied signature; that mode is
not evidence that a payment was verified.

## Signature encoding

Messages are concatenated byte sequences, not JSON or ABI tuples. Literal
prefixes use UTF-8, Ethereum contract addresses use their 20 raw bytes, and
unsigned numeric fields use 32-byte big-endian encoding. In particular, do not
concatenate hexadecimal address strings or decimal numbers where raw bytes are
required.

[utils/auth.go](../utils/auth.go) signs and recovers the digest:

```text
Keccak256("\x19Ethereum Signed Message:\n32" || Keccak256(message))
```

Signatures are 65-byte `R || S || V` values. The verifier accepts recovery IDs
encoded as 0/1 or 27/28. Binary gRPC metadata fields contain signature bytes,
not their hex text representation.

## Read payment-channel state

Call `escrow.PaymentChannelStateService/GetChannelState` using
[state_service.proto](state_service.proto). The signed message is:

```text
"__get_channel_state" || MPEAddress || ChannelID || CurrentBlock
```

The signer must be the channel's signer, sender, or recipient. The supplied
current block must be within five blocks of the daemon's latest observed block,
not ten. Use the returned state when determining the nonce and cumulative amount
for subsequent calls; claims can leave local and on-chain nonces temporarily
different. See [state_service.go](state_service.go).

## Execute a paid service call

Send the following gRPC metadata:

| Header | Value |
| --- | --- |
| `snet-payment-type` | `escrow` |
| `snet-payment-channel-id` | Channel ID as a decimal string. |
| `snet-payment-channel-nonce` | Current channel nonce as a decimal string. |
| `snet-payment-channel-amount` | Cumulative signed amount for that nonce, as a decimal string in token base units. |
| `snet-payment-channel-signature-bin` | Signature bytes. |

The payment signature covers:

```text
"__MPE_claim_message" || MPEAddress || ChannelID || ChannelNonce || SignedAmount
```

The signer must be the channel signer or sender. The signed amount is cumulative,
not just the price of this one call. The daemon validates the channel, nonce,
balance, and price before committing the payment transaction.

If `allowed_user_flag` is enabled, the signer must also be in
`allowed_user_addresses`. This optional testing restriction is rejected for
the `main` network. It is unrelated to operator-API authentication.

See [validation.go](validation.go) and the
[stream metadata definitions](../handler/stream_interceptor.go). For free-call
headers and tokens, use the [free-call guide](../freecall/README.md); the old
separate `snet-free-call-token-expiry-block` header is not used.

## Provider claims

[control_service.proto](control_service.proto) exposes
`GetListUnclaimed`, `GetListInProgress`, `StartClaim`, and
`StartClaimForMultipleChannels`. These methods require provider authorization
against the recipient configured in organization metadata.

List requests sign `"__list_unclaimed"` or `"__list_in_progress"`, followed by
the MPE address and current block. Their freshness window is five blocks.
Claim methods have different payloads, including channel identifiers/nonces;
use [control_service.go](control_service.go) or a compatible provider SDK to
construct them.

Starting a claim prepares and stores claim data and advances local channel state.
Use the separate [SNET CLI treasurer commands](https://dev.singularitynet.io/docs/products/DecentralizedAIPlatform/CLI/Manual/Treasurer/)
to complete the workflow: `snet treasurer print-unclaimed` queries available
payments, while `snet treasurer claim` and `snet treasurer claim-all` obtain claim
data from the daemon and sign/submit blockchain transactions. A compatible
provider client can perform the same steps.

The CLI's `--endpoint` identifies the daemon API, not etcd. Select the same
blockchain network and the provider identity matching the group's payment
recipient address, with funds for transaction gas. See the
[claim command examples](../README.md#commands); they keep interactive transaction
confirmation enabled.

There is no `snetd claim` CLI command. The daemon's local `list` and `channel`
commands inspect stored state or perform explicitly requested administrative
operations; they do not replace provider transaction submission.

## Prepaid tokens and storage

[TokenService](token_service.proto) issues HS256 JWT tokens for prepaid calls.
`token_expiry_in_minutes` defaults to `1440`.
`token_secret_key` must contain at least 32 bytes when blockchain is enabled.
The shipped value is a public test default: configure your own secret and use
the same secret on replicas that should accept the same tokens. Do not confuse
these JWTs with free-call tokens, whose lifetime is measured in blocks.

Payment state uses the configured atomic storage backend. Replicas sharing
payment state need the same etcd cluster; `memory` storage is process-local and
lost on exit. See the [etcd guide](../etcddb/README.md) and the
[complete configuration reference](../config/README.md).

