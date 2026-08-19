# Docker Configuration

This directory contains Dockerfiles for building and running the SNET daemon.
All images are based on Alpine Linux, run as a non-root user (`snet`, UID 10001)
and expose `/etc/singnet` as the standard configuration mount point.

## Files overview

| File | Purpose | Produces runnable image? |
| --- | --- | --- |
| `Dockerfile` | Multi-stage build from source. Compiles the daemon with Go and ships a minimal runtime image. | Yes |
| `Dockerfile.release` | Downloads a pre-built binary from GitHub Releases and packages it as a minimal runtime image. | Yes |
| `Dockerfile.build` | Builds the binary inside a container and leaves it at `/out/snetd` for extraction with `docker cp`. | No (build-only) |

Common build-time arguments (defaults):

| Arg | Default | Notes |
| --- | --- | --- |
| `GO_VERSION` | `1.26.5` | Pinned to the version declared in `go.mod`. |
| `VERSION` | `v6.2.3` | Version tag passed to the build script / GitHub release URL. |
| `TARGETOS` | `linux` | Target GOOS. |
| `TARGETARCH` | `amd64` | Target GOARCH. |

All runtime images use `alpine:3.24`.

## Dockerfile (build from source)

The main Dockerfile. It:

- Uses a multi-stage build with a `golang:${GO_VERSION}-alpine` builder.
- Runs `scripts/build` to produce the daemon binary.
- Creates a minimal runtime image based on `alpine:3.24`.
- Runs as a non-root user (`snet`, UID 10001) for security.
- Exposes `/etc/singnet` as a VOLUME for configuration.

### Build

```bash
docker build -f docker/Dockerfile -t snetd:v6.2.3 .
```

Override version or platform if needed:

```bash
docker build -f docker/Dockerfile \
  --build-arg VERSION=v6.2.3 \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=arm64 \
  -t snetd:v6.2.3-arm64 .
```

### Run

```bash
docker run --rm -it \
  -v "$(pwd)/snet-config:/etc/singnet:ro" \
  -p 7000:7000 \
  snetd:v6.2.3 serve -c /etc/singnet/snetd.config.json
```

If the mounted `/etc/singnet` directory is not readable by UID 10001, fix host
permissions (`chown`/`chmod`) or override the runtime user with `--user root`.

## Dockerfile.release (build from a release binary)

Alternative Dockerfile for production deployments that prefer to use the
official pre-built binary instead of compiling from source. It:

- Downloads the binary from
  `https://github.com/singnet/snet-daemon/releases/download/${VERSION}/snetd-${TARGETOS}-${TARGETARCH}-${VERSION}`.
- Creates a minimal runtime image based on `alpine:3.24`.
- Installs `ca-certificates` and `tzdata`, then removes `curl` to keep the
  final image small.

### Build

```bash
docker build -f docker/Dockerfile.release \
  --build-arg VERSION=v6.2.3 \
  -t snetd:v6.2.3 .
```

To target a different platform:

```bash
docker build -f docker/Dockerfile.release \
  --build-arg VERSION=v6.2.3 \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=arm64 \
  -t snetd:v6.2.3-arm64 .
```

### Run

```bash
docker run -d --rm \
  -v "$(pwd)/snet-config:/etc/singnet:ro" \
  -p 7000:7000 \
  snetd:v6.2.3
```

> **Note:** The downloaded binary is not currently verified against a checksum
> or signature. Only use `VERSION` values that match published GitHub releases.

## Dockerfile.build (extract a binary to the host)

Development Dockerfile for producing a binary inside a consistent containerized
environment, then copying it out to the host. It:

- Does not require Go or Protoc to be installed on the host.
- Uses the same multi-stage build process as the main `Dockerfile`.
- Leaves the compiled binary at `/out/snetd` in the final image.
- Does **not** define an `ENTRYPOINT`/`CMD`, a non-root user, or runtime
  dependencies — it is intended for binary extraction only, not for running the
  daemon.

The easiest way to use it is the helper script, which builds the image, copies
the binary out, and removes the intermediate container:

```bash
VERSION=v6.2.3 TARGET_OS=linux TARGET_ARCH=amd64 ./scripts/build_in_docker
```

The binary will be written to `build/snetd-linux-amd64-v6.2.3`.

If you want to drive the Dockerfile directly:

```bash
# Build the image
docker build -f docker/Dockerfile.build \
  --build-arg VERSION=v6.2.3 \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=amd64 \
  -t snetd:build .

# Copy the binary out
docker create --name snetd-build snetd:build
docker cp snetd-build:/out/snetd ./build/snetd-linux-amd64-v6.2.3
docker rm -f snetd-build
```

## Running with a custom config

All runnable images share the same entrypoint and expect the configuration file
at `/etc/singnet/snetd.config.json`. Mount your config directory read-only and
pass any extra `snetd` flags after the image name:

```bash
docker run -d --rm \
  --name snetd \
  -v "$(pwd)/snet-config:/etc/singnet:ro" \
  -p 7000:7000 \
  snetd:v6.2.3 serve -c /etc/singnet/snetd.config.json
```

PowerShell (Windows):

```powershell
docker run -d --rm --name snetd `
  -v "${PWD}\snet-config:/etc/singnet:ro" `
  -p 7000:7000 `
  snetd:v6.2.3 serve -c /etc/singnet/snetd.config.json
```
