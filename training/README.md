# Model training

The daemon exposes a consumer-facing `training.Daemon` gRPC service and forwards
provider operations to `training.Model`. These are separate APIs:

- [training_daemon.proto](training_daemon.proto): daemon requests, authorization,
  model access, and metadata.
- [training.proto](training.proto): provider RPCs, model records/statuses, and
  protobuf method options.

## Configuration

| Key | Default | Behavior |
| --- | --- | --- |
| `model_training_enabled` | `false` | Enables training when blockchain is enabled and the service protobuf metadata supports training. |
| `model_maintenance_endpoint` | Unset | Uses `service_endpoint` when no separate maintenance endpoint is configured. |
| `enable_dynamic_pricing` | `false` | Enables the separate dynamic-pricing integration; not a replacement for training configuration. |

See the [configuration reference](../config/README.md). Setting the training
flag alone does not make an arbitrary backend implement the provider RPCs.
Model records use the daemon's configured storage; memory-backed records do not
survive a restart.

## Consumer operations

Use the actual RPC names from `training.Daemon`:

| RPC | Purpose |
| --- | --- |
| `create_model` | Create a model, including its name, description, method/service, access list, and `is_public` value. |
| `validate_model_price` | Request a validation price. |
| `upload_and_validate` | Upload a dataset as a client stream and validate it; paid operation. |
| `validate_model` | Validate using a training-data link; paid operation. |
| `train_model_price` | Request a training price. |
| `train_model` | Start training; paid operation. |
| `get_model` | Retrieve a model record and status, subject to access checks. |
| `get_all_models` | List accessible models, with filters and pagination. |
| `update_model` | Update the name, description, and address list. |
| `delete_model` | Request deletion; the stored status becomes `DELETED`. |
| `get_training_metadata` | Read training availability and declared methods. |
| `get_method_metadata` | Read training options for a method/model without authorization. |

There are no daemon RPCs named `CreateModel`, `UpdateModelAccess`, or
`GetAllAccessibleModels`. The provider API has `get_model_status`; consumer
clients use the daemon's `get_model`.

`UpdateModelRequest` does not contain an `is_public` field, so it cannot switch
a model between public and private. Set that value when creating the model.
Public visibility/access does not grant ownership or unrestricted mutation
rights.

The status enum includes `CREATED`, `VALIDATING`, `VALIDATED`, `TRAINING`,
`READY_TO_USE`, `ERRORED`, and `DELETED`. Backend operations and failures
determine the transitions; not every request completes training synchronously.

## Authorization

Authenticated requests carry `AuthorizationDetails`. The signed message is:

```text
Message || SignerAddress || CurrentBlock
```

`Message` is the supplied UTF-8 method name, the address is 20 raw bytes, and
the block is a 32-byte big-endian unsigned integer. Signing uses the
[shared Ethereum convention](../escrow/README.md#signature-encoding).
Method-name matching is case-insensitive, but sign the exact bytes sent in
`message`.

Normally `message` must match the RPC's final method name, with a freshness
window of five blocks. The special `unified` message is accepted only for
`validate_model_price`, `train_model_price`, `get_all_models`, and `get_model`,
with a 600-block window. It does not authorize `train_model` itself.
Authorization does not replace the payment metadata required for paid methods.

## Provider metadata

A provider implementing `training.Model` supplies model creation, validation,
training, deletion, pricing, and status operations. Inference-method options
declared in `training.proto` describe:

- A default model and maximum models per user.
- Dataset size, file count, and per-file size.
- File types, archive/dataset types, and free-form requirements.

These are protobuf method options, not additional daemon JSON configuration
keys. Do not assume that publishing a dataset requirement automatically enforces
it everywhere: the provider must validate the data it accepts. Refer to
[service.go](service.go) and [util.go](util.go) for the daemon's access checks and
metadata handling.
