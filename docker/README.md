# Docker Configuration

This directory contains Dockerfiles for building and running the SNET daemon.
Run the examples from the repository root. The commands below use Bash unless
marked as PowerShell.

## Files overview

| File | Purpose | Runtime user |
| --- | --- | --- |
| [Dockerfile](Dockerfile) | Build from a prepared source checkout and package the daemon. | `snet`, UID 10001 |
| [Dockerfile.release](Dockerfile.release) | Download a release binary and package the daemon. | `snet`, UID 10001 |
| [Dockerfile.build](Dockerfile.build) | Build a binary at `/out/snetd` for extraction. | No non-root user configured; not a daemon runtime image |

The runtime images use `alpine:3.24`, include CA certificates and timezone data,
and use `/etc/singnet` as the configuration mount point.

| Build argument | Default | Availability |
| --- | --- | --- |
| `GO_VERSION` | `1.27.2` | Source and binary-extraction builds; maintained alongside `go.mod`, not read from it automatically. |
| `VERSION` | `v6.2.3` | All three Dockerfiles; build version or release download tag. This is not an automatic selection of the latest release. |
| `TARGETOS` | `linux` | Explicitly declared in `Dockerfile.release` and `Dockerfile.build`. |
| `TARGETARCH` | `amd64` | Explicitly declared in `Dockerfile.release` and `Dockerfile.build`. |

## Dockerfile (build from source)

The source-build Dockerfiles run `scripts/build`, which does not generate
protobuf bindings. Prepare the checkout first: install Go, `protoc`, and the Go
protobuf generators, put the generators on `PATH`, and run:

```bash
go generate ./...
docker build -f docker/Dockerfile -t snetd:v6.2.3 .
```

See the root [build instructions](../README.md#build). Generated `*.pb.go` files
are not tracked by Git but must be present in the build context. Keep the Git
metadata available too: the build script reads `git rev-parse HEAD`.

To change the compiled version:

```bash
docker build -f docker/Dockerfile \
  --build-arg VERSION=v6.2.3 \
  -t snetd:v6.2.3 .
```

The main Dockerfile currently does not declare `TARGETOS`/`TARGETARCH` inside
its build stage. Its shell fallbacks compile for Linux/amd64; passing those
build arguments or changing `--platform` alone is not a supported cross-platform
build recipe for this Dockerfile. Use the explicit target arguments in the
binary-extraction Dockerfile, or a matching release binary, instead.

## Dockerfile.release (build from a release binary)

This Dockerfile downloads
`snetd-${TARGETOS}-${TARGETARCH}-${VERSION}` from the specified GitHub release.
It does not need a source build or generated bindings.

```bash
docker build -f docker/Dockerfile.release \
  --platform linux/amd64 \
  --build-arg VERSION=v6.2.3 \
  --build-arg TARGETARCH=amd64 \
  -t snetd:v6.2.3 .
```

For ARM64, use both `--platform linux/arm64` and
`--build-arg TARGETARCH=arm64`. Keep `TARGETOS=linux` for these Alpine runtime
images. The selected release must contain the requested asset.

The download currently has no checksum or signature verification. Use a known
published release tag; the default tag is a fixed value, not “latest”.

## Dockerfile.build (extract a binary to the host)

Prepare the source checkout and generated bindings as described above. This
Dockerfile compiles in a container but does not install `protoc` or generate
bindings itself.

```bash
docker build -f docker/Dockerfile.build \
  --build-arg VERSION=v6.2.3 \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=amd64 \
  -t snetd:build .

mkdir -p build
docker create --name snetd-build snetd:build /bin/true
docker cp snetd-build:/out/snetd ./build/snetd-linux-amd64-v6.2.3
docker rm snetd-build
```

The temporary container is not started. This Dockerfile does not configure a
daemon entrypoint; the explicit dummy command avoids depending on the base
image's default command. Choose another container name if `snetd-build` already
exists. For another Linux architecture, change
`TARGETARCH` and the output filename together. The current copy step does not
handle the `.exe` suffix used by Windows builds.

The legacy `scripts/build_in_docker` helper is not equivalent to this recipe:
it does not forward its version/target variables as Docker build arguments, so
the output filename can disagree with the actual binary version/target. It also
forcibly removes its fixed temporary container and version-tagged image. Use the explicit commands
above until that helper is updated.

## Running with a custom config

Both runtime images use entrypoint `snetd` and default arguments
`serve -c /etc/singnet/snetd.config.json`. Mount a real configuration file;
an explicitly requested missing file is an error.

For published container ports, set `daemon_endpoint` to `0.0.0.0:8080` inside
the container. The daemon default `127.0.0.1:8080` is only reachable inside that
container. Configure `service_endpoint` and etcd client endpoints for addresses
reachable from the container; `localhost` refers to the container itself.

For example, merge this fragment into your complete configuration:

```json
{
  "daemon_endpoint": "0.0.0.0:8080",
  "log": {
    "output": {
      "type": ["stdout"]
    }
  },
  "payment_channel_storage_server": {
    "enabled": false
  }
}
```

This fragment does not supply organization/service registration, a backend,
external etcd settings, or a deployment-specific token secret. See the complete
[configuration reference](../config/README.md).

```bash
docker run -d --rm --name snetd \
  -v "$(pwd)/snet-config:/etc/singnet:ro" \
  -p 8080:8080 \
  snetd:v6.2.3
```

PowerShell:

```powershell
docker run -d --rm --name snetd `
  -v "${PWD}\snet-config:/etc/singnet:ro" `
  -p 8080:8080 `
  snetd:v6.2.3
```

Mounted config and certificate files must be readable by UID 10001. The working
directory `/app` is not made writable for that user, so default relative file
logging, embedded etcd data/log paths, and ACME cache paths require attention.
Use stdout logging and, where needed, separate writable mounts with explicit
paths for `payment_channel_storage_server.data_dir`, its `log_outputs`,
`log.output.file_pattern`/`current_link`, and `auto_ssl_cache_dir`. Do not put
writable data under the read-only config mount.

For embedded etcd, set `enabled: true`, persist its data directory, and configure
client endpoints separately. Server startup does not redirect clients to it.
Publish peer/client ports only when other cluster members or daemon replicas
need to reach them. See the [etcd guide](../etcddb/README.md).
