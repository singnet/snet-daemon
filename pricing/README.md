# Pricing

The `pricing` package calculates the price of an incoming gRPC request in cogs.
Prices are integers represented by
`big.Int`; dynamic pricing responses carry an unsigned 64-bit integer.

## Pricing models

| Model | Behavior |
| --- | --- |
| `fixed_price` | Returns the selected group's default `price_in_cogs` for every request. |
| `fixed_price_per_method` | Looks up the full RPC path in the selected group's default pricing details. A missing method price returns an error. |
| `dynamic_pricing` | Calls the backend's mapped pricing RPC with the first message of the incoming request. |

`InitPricingStrategy` registers the default fixed model from service metadata.
When `enable_dynamic_pricing` is enabled, it also registers dynamic pricing.
For each request, a nonempty dynamic method mapping takes precedence over the
default fixed model. Requests without a mapping use that default model.

The metadata group is selected using `daemon_group_name`. A fixed per-method
entry uses `package_name`, `service_name`, and `method_name` to construct a path
such as `/example.Calculator/Add`.

## Dynamic pricing

The service API imports `pricing.proto` and annotates the inference RPC with
the full path of its pricing RPC:

```proto
import "pricing.proto";

service Calculator {
  rpc Add(AddRequest) returns (AddResponse) {
    option (pricing.my_method_option).estimatePriceMethod =
      "/example.Calculator/EstimateAdd";
  }

  rpc EstimateAdd(AddRequest) returns (pricing.PriceInCogs);
}
```

In this example the proto file declares `package example;` and defines
`AddRequest` and `AddResponse`. The daemon parses the annotation into
`ServiceMetadata.DynamicPriceMethodMapping` when loading the service API.
The pricing RPC must accept the same request message as the inference RPC and
return `pricing.PriceInCogs`, whose `price` field is a `uint64` number of cogs.

The daemon sends the buffered first request message to the mapped RPC at
`service_endpoint`. It forwards incoming gRPC metadata and propagates request
cancellation and deadlines. The buffered message remains available for the
subsequent inference call. The current pricing connection uses insecure gRPC
transport to the endpoint's host; configure transport protection separately
where required. The connection closes after estimation.

The estimate uses one request message and one pricing response, including for
streaming inference methods. It does not accumulate a price across later
messages. Both zero and the full `uint64` price range are supported.

Missing method context or incoming metadata, invalid endpoint URLs, RPC
transport failures, and invalid protobuf responses return errors. A mapped
pricing RPC failure propagates to the caller; fixed pricing is used only when
no dynamic mapping is selected.

## Tests

```shell
go test ./pricing -count=1
```

The tests cover fixed pricing, method lookup and fallback, dynamic RPC request
and metadata forwarding, preservation of the inference message, integer
boundaries, canceled requests, transport errors, and malformed responses.
