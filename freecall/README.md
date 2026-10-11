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

Existing data uses `/free-call-user/storage` and locks use `/freecall/lock`.
Their keys, gob field names, and lock values are unchanged; no storage migration
is required. Compatibility tests exercise both the original protobuf schema and
records encoded with the original `FreeCallUserData` layout.
