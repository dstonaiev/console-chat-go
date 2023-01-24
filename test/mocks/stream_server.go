package mocks

import (
	"context"

	"github.com/chat/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockStreamServer struct {
	mock.Mock
	grpc.ServerStream
}

func (m *MockStreamServer) Send(msg *proto.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockStreamServer) Context() context.Context {
	return context.Background()
}
