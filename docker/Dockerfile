# syntax=docker/dockerfile:1

# Match go.mod version
ARG GO_VERSION=1.25.0

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Minimal build deps (git for ldflags, bash to run your script, certs for go mod)
RUN apk add --no-cache git bash ca-certificates && update-ca-certificates

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Project files (includes scripts/build and snet-config)
COPY . .

# Optional version tag passed at build time
ARG VERSION=6.1.0

# Run your build script and copy the resulting binary to /out
RUN chmod +x scripts/build \
 && ./scripts/build "${TARGETOS:-linux}" "${TARGETARCH:-amd64}" "${VERSION}" \
 && mkdir -p /out \
 && cp "build/snetd-${TARGETOS:-linux}-${TARGETARCH:-amd64}-${VERSION}" /out/snetd

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 snet
WORKDIR /app

COPY --from=build /out/snetd /usr/local/bin/snetd

# Standard config mount point
RUN mkdir -p /etc/singnet && chown -R snet /etc/singnet
VOLUME ["/etc/singnet"]

# Run as non-root for security (UID 10001).
# If the mounted /etc/singnet is not readable by UID 10001, fix host perms
# (chown/chmod) or override at runtime with `--user root`.
# Only comment this line if you intentionally want to run as root.
USER snet

ENTRYPOINT ["snetd"]
CMD ["serve","-c","/etc/singnet/snetd.config.json"]
