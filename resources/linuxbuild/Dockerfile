FROM golang:alpine
RUN apk update && apk add --no-cache make protobuf-dev git bash
SHELL ["/bin/bash", "-c"]
RUN mkdir -p /snet-daemon
WORKDIR /snet-daemon
RUN git clone --single-branch --branch dev https://github.com/semyon-dev/snet-daemon.git .
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go generate ./...
RUN chmod +x ./scripts/build
RUN ./scripts/build linux amd64 dev
