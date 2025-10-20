# Docker Configuration

This directory contains Docker-related configuration files for building and running the SNET daemon.

## Dockerfile

The main Dockerfile used for building the daemon from source code. It:

- Uses multi-stage build with Go 1.25.0 Alpine base
- Builds the daemon binary from source
- Creates a minimal runtime image based on Alpine
- Configures a non-root user (snet) for security
- Exposes volume mount point for configuration at `/etc/singnet`

#### Usage
Build Docker Image

```bash
docker build -f docker/Dockerfile -t snetd:dev .
```

Run
```bash
docker run --rm -it -v $(pwd)/etc:/etc/singnet -p 7000:7000 snetd:dev serve -c /etc/singnet/snetd.config.json
```


## Dockerfile.release

Alternative Dockerfile for building from released binaries. It:

- Downloads pre-built binary from GitHub releases
- Creates minimal runtime container
- Suitable for production deployments
- Uses Alpine base image for small footprint
  Build docker with a release binary

```bash
docker build -f docker/Dockerfile.release --build-arg VERSION=v6.1.0 -t snetd:6.1.0 .
```

Run
```bash
docker run -d --rm -it -v $(pwd)/etc:/etc/singnet -p 7000:7000 snetd:6.1.0
```

## Dockerfile.build

Build binary in container
Development Dockerfile for building in containerized environment. It:

- You don't need to install Go/Protoc on your host system
- Uses the same multi-stage build process as main Dockerfile
- Copies built binary to host system
- Useful for consistent builds across different development environments
- Handles cross-compilation via TARGETOS and TARGETARCH args


Build a binary in a container and copy it to host

```bash
docker build -f docker/Dockerfile.build -t snetd:build .
```

Run
```bash
docker run -d --rm --name snetd -v "$(pwd)/snet-config:/etc/singnet:ro" -p 8080:8080 snet-daemon:v6.1.0
```

powershell:
```powershell
docker run -d --rm --name snetd -v "$( PWD )\snet-config:/etc/singnet:ro" -p 8080:8080 snet-daemon:v6.1.0
```
