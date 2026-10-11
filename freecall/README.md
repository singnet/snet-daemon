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

The [Java SDK free-call client](https://github.com/singnet/snet-sdk-java/blob/master/sdk/src/main/java/io/singularitynet/sdk/freecall/FreeCallStateService.java)
uses `StateService.*`, and its
[schema download task](https://github.com/singnet/snet-sdk-java/blob/master/sdk/get-dependencies.xml)
collects only the old state and control schemas. That SDK needs a coordinated
update when adopting this extraction. Such an update is not included in this
daemon PR, and its rollout has not been confirmed. Existing clients do not need
new bindings solely because Java class names changed; wire compatibility and
Java source compatibility are separate concerns.
