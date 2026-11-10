# Rate limiting

The daemon uses a token bucket from [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate).
Tokens replenish continuously; a full bucket permits an initial burst, then
additional requests depend on token availability. Excess calls are rejected,
not queued until the next time window.

## Configuration

| Key | Effective default | Behavior |
|-----|-------------------|----------|
| `rate_limit_per_minute` | Unlimited | Refill rate per minute. Accepts numbers or numeric strings, including fractional values. Missing, `0`, or unparsable input currently disables limiting. |
| `burst_size` | `2147483647` | Integer bucket capacity. Missing/`0` becomes `math.MaxInt32`; ignored when the rate is unlimited. |

Use a positive finite rate and an explicit positive burst together. Setting only
a finite rate leaves a very large initial bucket, so it will not enforce the
small immediate burst most operators expect. Do not use invalid or negative
values as a way to express a policy.

The limiter is local to the daemon's handler/interceptor instances, not a shared
etcd quota or a per-wallet allowance. Separate replicas have independent buckets.
Free-call quotas are a different mechanism; see [free calls](../freecall/README.md).

## Example

Merge into the top-level daemon configuration:

```json
{
  "rate_limit_per_minute": 60,
  "burst_size": 10
}
```

This allows an initial burst of ten calls and then replenishes approximately one
token per second. With `rate_limit_per_minute: 1` and `burst_size: 1`, one call is
available immediately and one token replenishes each minute. A rate of `0.5`
replenishes one token every two minutes.

The gRPC rate-limit interceptor rejects excess calls with `ResourceExhausted`;
the simple HTTP proxy uses HTTP 429 when passthrough is enabled.
The configuration reference is in [config/README.md](../config/README.md#rate-limiting).
