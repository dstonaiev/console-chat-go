package mocks

import (
	"context"

	"github.com/chat/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockMessengerClient struct {
	mock.Mock
}

func (m *MockMessengerClient) Login(ctx context.Context, in *proto.User, opts ...grpc.CallOption) (*proto.LoginResponse, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.LoginResponse), args.Error(1)
}

func (m *MockMessengerClient) Logout(ctx context.Context, in *proto.User, opts ...grpc.CallOption) (*proto.Close, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.Close), args.Error(1)
}

func (m *MockMessengerClient) CreateStream(ctx context.Context, in *proto.User, opts ...grpc.CallOption) (proto.Messenger_CreateStreamClient, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(proto.Messenger_CreateStreamClient), args.Error(1)
}

func (m *MockMessengerClient) SendMessage(ctx context.Context, in *proto.Message, opts ...grpc.CallOption) (*proto.Close, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.Close), args.Error(1)
}

func (m *MockMessengerClient) CreateGroupChat(ctx context.Context, in *proto.GroupRequest, opts ...grpc.CallOption) (*proto.Close, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.Close), args.Error(1)
}

func (m *MockMessengerClient) JoinGroupChat(ctx context.Context, in *proto.GroupRequest, opts ...grpc.CallOption) (*proto.Close, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.Close), args.Error(1)
}

func (m *MockMessengerClient) LeftGroupChat(ctx context.Context, in *proto.GroupRequest, opts ...grpc.CallOption) (*proto.Close, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.Close), args.Error(1)
}

func (m *MockMessengerClient) ListChannels(ctx context.Context, in *proto.User, opts ...grpc.CallOption) (*proto.ChannelResponse, error) {
	args := m.Called(ctx, in)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.ChannelResponse), args.Error(1)
}
