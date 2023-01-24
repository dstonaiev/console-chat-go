package mocks

import (
	"github.com/chat/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockStreamClient struct {
	mock.Mock
	grpc.ClientStream
}

func (m *MockStreamClient) Recv() (*proto.Message, error) {
	args := m.Called()
	return args.Get(0).(*proto.Message), args.Error(1)
}
