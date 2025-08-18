package handler

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type serverStreamMock struct {
	context context.Context
}

func (m *serverStreamMock) Context() context.Context {
	return m.context
}

func (m *serverStreamMock) SetHeader(metadata.MD) error {
	return nil
}

func (m *serverStreamMock) SendHeader(metadata.MD) error {
	return nil
}

func (m *serverStreamMock) SetTrailer(metadata.MD) {
}

func (m *serverStreamMock) SendMsg(any) error {
	return nil
}

func (m *serverStreamMock) RecvMsg(any) error {
	return nil
}
