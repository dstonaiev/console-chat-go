package service_test

import (
	"context"
	"errors"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/chat/internal/app/client/service"
	"github.com/chat/proto"
	"github.com/chat/test/mocks"
	"github.com/chat/test/utils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type chatAppSuite struct {
	suite.Suite
	ctx         context.Context
	logger      *log.Logger
	serviceName string
	chatService service.Chat
	grpcClient  *mocks.MockMessengerClient
}

func TestChatServiceSuite(t *testing.T) {
	suite.Run(t, new(chatAppSuite))
}

func (suite *chatAppSuite) SetupTest() {
	suite.logger = log.New(os.Stdout, "testClient", 0)
	suite.ctx = context.Background()
	suite.grpcClient = &mocks.MockMessengerClient{}
	suite.chatService = service.NewChat(suite.ctx, "localhost:7777", suite.grpcClient, suite.logger, true)
}

func (suite *chatAppSuite) TestSendMessage() {
	user := &proto.User{
		Id:          "11111111",
		DisplayName: "TOM",
	}

	text := "hello"
	address := "testAddr"

	messageMatcher := mock.MatchedBy(func(msg *proto.Message) bool {
		return msg.Id == user.Id &&
			msg.User == user &&
			msg.Text == text &&
			msg.Address == address &&
			msg.Group == false
	})

	utils.SetFieldValue(suite.chatService, "user", user)
	suite.grpcClient.On("SendMessage", suite.ctx, messageMatcher).Return(&proto.Close{}, nil)

	suite.chatService.SendMessage(suite.ctx, address, text)

	suite.grpcClient.AssertCalled(suite.T(), "SendMessage", suite.ctx, messageMatcher)
}

func (suite *chatAppSuite) TestSendMessageToGroup() {
	user := &proto.User{
		Id:          "222222",
		DisplayName: "DAN",
	}

	text := "hello group"
	messageMatcher := mock.MatchedBy(func(msg *proto.Message) bool {
		return msg.Id == user.Id &&
			msg.User == user &&
			msg.Text == text &&
			msg.Address == "my_group" &&
			msg.Group == true
	})

	utils.SetFieldValue(suite.chatService, "user", user)
	suite.grpcClient.On("SendMessage", suite.ctx, messageMatcher).Return(&proto.Close{}, nil)

	suite.chatService.SendMessage(suite.ctx, "@@my_group", text)

	suite.grpcClient.AssertCalled(suite.T(), "SendMessage", suite.ctx, messageMatcher)
}

func (suite *chatAppSuite) TestListChannels() {
	user := &proto.User{
		Id:          "333333",
		DisplayName: "Peter",
	}
	utils.SetFieldValue(suite.chatService, "user", user)
	suite.grpcClient.On("ListChannels", suite.ctx, user).Return(&proto.ChannelResponse{
		Name: []string{"testContact", "@@testGroup"},
	}, nil)

	channels, err := suite.chatService.ListChannels(suite.ctx)

	suite.grpcClient.AssertCalled(suite.T(), "ListChannels", suite.ctx, user)
	suite.Require().NoError(err)
	suite.Require().NotNil(channels)
	suite.Require().Len(channels, 2)
	suite.Require().Equal("testContact", channels[0])
	suite.Require().Equal("@@testGroup", channels[1])
}

func (suite *chatAppSuite) TestLogin() {
	name := "testUser"

	userMatcher := mock.MatchedBy(func(usr *proto.User) bool {
		return usr.DisplayName == name
	})
	mockStreamClient := &mocks.MockStreamClient{}
	suite.grpcClient.On("Login", suite.ctx, userMatcher).Return(&proto.LoginResponse{}, nil)
	suite.grpcClient.On("CreateStream", suite.ctx, userMatcher).Return(mockStreamClient, nil)

	streamClient, err := suite.chatService.Login(suite.ctx, name)

	suite.grpcClient.AssertCalled(suite.T(), "Login", suite.ctx, userMatcher)
	suite.grpcClient.AssertCalled(suite.T(), "CreateStream", suite.ctx, userMatcher)
	suite.Require().NoError(err)
	suite.Require().NotNil(streamClient)
}

func (suite *chatAppSuite) TestLoginFailed() {
	name := "testUser2"
	userMatcher := mock.MatchedBy(func(usr *proto.User) bool {
		return usr.DisplayName == name
	})
	mockStreamClient := &mocks.MockStreamClient{}
	suite.grpcClient.On("Login", suite.ctx, userMatcher).Return(&proto.LoginResponse{}, errors.New("user exists"))
	suite.grpcClient.On("CreateStream", suite.ctx, userMatcher).Return(mockStreamClient, nil)

	streamClient, err := suite.chatService.Login(suite.ctx, name)

	suite.grpcClient.AssertCalled(suite.T(), "Login", suite.ctx, userMatcher)
	suite.grpcClient.AssertNotCalled(suite.T(), "CreateStream", suite.ctx, userMatcher)
	suite.Require().EqualError(err, "user exists")
	suite.Require().Nil(streamClient)
}

func (suite *chatAppSuite) TestLoginConnectionFailed() {
	name := "testUser"
	user := &proto.User{
		Id:          "555555",
		DisplayName: name,
	}
	userMatcher := mock.MatchedBy(func(usr *proto.User) bool {
		return usr.DisplayName == name
	})
	utils.SetFieldValue(suite.chatService, "user", user)
	suite.grpcClient.On("Login", suite.ctx, userMatcher).Return(&proto.LoginResponse{}, nil)
	suite.grpcClient.On("CreateStream", suite.ctx, userMatcher).Return(nil, errors.New("failed to connect to server"))

	streamClient, err := suite.chatService.Login(suite.ctx, name)

	suite.grpcClient.AssertCalled(suite.T(), "Login", suite.ctx, userMatcher)
	suite.grpcClient.AssertCalled(suite.T(), "CreateStream", suite.ctx, userMatcher)
	suite.Require().EqualError(err, "unable to connect to the server, error: failed to connect to server")
	suite.Require().Nil(streamClient)
}

func (suite *chatAppSuite) TestLogout() {
	user := &proto.User{
		Id:          "6666666",
		DisplayName: "Name3",
	}
	utils.SetFieldValue(suite.chatService, "user", user)
	suite.grpcClient.On("Logout", mock.AnythingOfType(reflect.TypeOf(suite.ctx).Name()), user).Return(&proto.Close{}, nil)

	err := suite.chatService.Logout(suite.ctx)

	suite.grpcClient.AssertCalled(suite.T(), "Logout", mock.AnythingOfType(reflect.TypeOf(suite.ctx).Name()), user)
	suite.Require().NoError(err)
}

func (suite *chatAppSuite) TestLogoutFailed() {
	user := &proto.User{
		Id:          "777777",
		DisplayName: "Name4",
	}
	utils.SetFieldValue(suite.chatService, "user", user)
	suite.grpcClient.On("Logout", mock.AnythingOfType(reflect.TypeOf(suite.ctx).Name()), user).Return(&proto.Close{}, errors.New("failed to logout"))

	err := suite.chatService.Logout(suite.ctx)

	suite.grpcClient.AssertCalled(suite.T(), "Logout", mock.AnythingOfType(reflect.TypeOf(suite.ctx).Name()), user)
	suite.Require().EqualError(err, "failed to logout")
}

func (suite *chatAppSuite) TestCreateGroupChat() {
	user := &proto.User{
		Id:          "888888",
		DisplayName: "Name5",
	}
	utils.SetFieldValue(suite.chatService, "user", user)
	groupName := "testGroup"
	req := &proto.GroupRequest{
		Name: groupName,
		User: user,
	}
	suite.grpcClient.On("CreateGroupChat", suite.ctx, req).Return(&proto.Close{}, nil)

	err := suite.chatService.CreateGroupChat(suite.ctx, groupName)

	suite.grpcClient.AssertCalled(suite.T(), "CreateGroupChat", suite.ctx, req)
	suite.Require().NoError(err)
}

func (suite *chatAppSuite) TestJoinGroupChat() {
	user := &proto.User{
		Id:          "888888",
		DisplayName: "Name5",
	}
	utils.SetFieldValue(suite.chatService, "user", user)
	groupName := "testGroup2"
	req := &proto.GroupRequest{
		Name: groupName,
		User: user,
	}
	suite.grpcClient.On("JoinGroupChat", suite.ctx, req).Return(&proto.Close{}, nil)

	err := suite.chatService.JoinGroupChat(suite.ctx, groupName)

	suite.grpcClient.AssertCalled(suite.T(), "JoinGroupChat", suite.ctx, req)
	suite.Require().NoError(err)
}

func (suite *chatAppSuite) TestLeftGroupChat() {
	user := &proto.User{
		Id:          "9999999",
		DisplayName: "Name6",
	}
	utils.SetFieldValue(suite.chatService, "user", user)
	groupName := "testGroup3"
	req := &proto.GroupRequest{
		Name: groupName,
		User: user,
	}
	suite.grpcClient.On("LeftGroupChat", suite.ctx, req).Return(&proto.Close{}, nil)

	err := suite.chatService.LeftGroupChat(suite.ctx, groupName)

	suite.grpcClient.AssertCalled(suite.T(), "LeftGroupChat", suite.ctx, req)
	suite.Require().NoError(err)
}
