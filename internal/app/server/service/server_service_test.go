package service_test

import (
	"context"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	model "github.com/chat/internal/app/server/model"
	"github.com/chat/internal/app/server/service"
	"github.com/chat/proto"
	"github.com/chat/test/mocks"
	"github.com/chat/test/utils"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type messengerServiceSuite struct {
	suite.Suite

	ctx        context.Context
	logger     *log.Logger
	user       proto.User
	mockStream *mocks.MockStreamServer
}

func TestMessengerServiceSuite(t *testing.T) {
	suite.Run(t, new(messengerServiceSuite))
}

func (suite *messengerServiceSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.logger = log.New(os.Stdout, "testServer", 0)
	suite.user = proto.User{
		Id:          "435647757",
		DisplayName: "TEST",
	}
	suite.mockStream = new(mocks.MockStreamServer)
}

func (suite *messengerServiceSuite) TestCreateStream() {
	service := service.NewMessengerServer(suite.logger)

	err := service.CreateStream(&suite.user, suite.mockStream)

	suite.Require().NoError(err)
	suite.assertConnectionsLength(service, 1)
	suite.assertConnectionIsValid(service, suite.user.DisplayName)
}

func (suite *messengerServiceSuite) TestLoginOK() {
	service := service.NewMessengerServer(suite.logger)

	resp, err := service.Login(suite.ctx, &suite.user)

	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Equal(suite.user.Id, resp.Token)
}

func (suite *messengerServiceSuite) TestLoginDuplicate() {
	service := service.NewMessengerServer(suite.logger)

	staleConnId := "224dsfg"
	conns := make(map[string]*model.Connection)
	conns[suite.user.DisplayName] = &model.Connection{
		Id:     staleConnId,
		Stream: suite.mockStream,
		Active: true,
		Err:    make(chan error),
	}
	utils.SetFieldValue(service, "Connections", conns)

	resp, err := service.Login(suite.ctx, &suite.user)

	suite.Require().EqualError(err, "user "+suite.user.DisplayName+" was authenticated previously\n")
	suite.Require().NotNil(resp)
	suite.Require().Equal(staleConnId, resp.Token)
}

func (suite *messengerServiceSuite) TestLoginConnInactive() {
	service := service.NewMessengerServer(suite.logger)

	staleConnId := "224dsfg"
	conns := make(map[string]*model.Connection)
	conns[suite.user.DisplayName] = &model.Connection{
		Id:     staleConnId,
		Stream: suite.mockStream,
		Active: false, //connection is inactive so replace it
		Err:    make(chan error),
	}
	utils.SetFieldValue(service, "Connections", conns)

	resp, err := service.Login(suite.ctx, &suite.user)

	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Equal(suite.user.Id, resp.Token)
}

func (suite *messengerServiceSuite) TestLogoutOK() {
	service := service.NewMessengerServer(suite.logger)
	conns := make(map[string]*model.Connection)
	conns[suite.user.DisplayName] = &model.Connection{
		Id:     suite.user.Id,
		Stream: suite.mockStream,
		Active: true,
		Err:    make(chan error),
	}
	utils.SetFieldValue(service, "Connections", conns)

	_, err := service.Logout(suite.ctx, &suite.user)

	suite.Require().NoError(err)
	suite.assertConnectionsLength(service, 0)
}

func (suite *messengerServiceSuite) TestLogoutConnNotFound() {
	service := service.NewMessengerServer(suite.logger)

	_, err := service.Logout(suite.ctx, &suite.user)

	suite.Require().EqualError(err, "invalid logout request: connection for user "+suite.user.DisplayName+" not found")
}

func (suite *messengerServiceSuite) TestSendMessageSingleOK() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)

	msg := &proto.Message{
		Id:   "434675899",
		User: &suite.user,
		Text: "Hi there, how are you?",
		Timestamp: &timestamppb.Timestamp{
			Seconds: 13254677889,
			Nanos:   54674,
		},
		Address: "userA",
		Group:   false,
	}
	suite.mockStream.On("Send", msg).Return(nil)

	_, err := service.SendMessage(suite.ctx, msg)

	suite.mockStream.AssertCalled(suite.T(), "Send", msg)

	suite.Require().NoError(err)

}

func (suite *messengerServiceSuite) TestSendMessageGroupOK() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	msg := &proto.Message{
		Id:        "434675899",
		User:      &suite.user,
		Text:      "Hi there, how are you?",
		Timestamp: timestamppb.Now(),
		Address:   "",
		Group:     true,
	}
	_, err := service.SendMessage(suite.ctx, msg)

	suite.Require().NoError(err)

}

func (suite *messengerServiceSuite) TestSendMessageSingleConnNotFound() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)

	msg := &proto.Message{
		Id:        "434675899",
		User:      &suite.user,
		Text:      "Hi there, how are you?",
		Timestamp: timestamppb.Now(),
		Address:   "noname",
		Group:     false,
	}
	_, err := service.SendMessage(suite.ctx, msg)

	suite.Require().EqualErrorf(err, "", "")

}

func (suite *messengerServiceSuite) TestSendMessageSingleConnInactive() {
	service := service.NewMessengerServer(suite.logger)

	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	msg := &proto.Message{
		Id:        "434675899",
		User:      &suite.user,
		Text:      "Hi there, how are you?",
		Timestamp: timestamppb.Now(),
		Address:   "",
		Group:     false,
	}
	_, err := service.SendMessage(suite.ctx, msg)

	suite.Require().EqualErrorf(err, "", "")

}

func (suite *messengerServiceSuite) TestSendMessageGroupNotFound() {
	service := service.NewMessengerServer(suite.logger)
	msg := &proto.Message{
		Id:        "434675899",
		User:      &suite.user,
		Text:      "Hi there, how are you?",
		Timestamp: timestamppb.Now(),
		Address:   "",
		Group:     true,
	}
	_, err := service.SendMessage(suite.ctx, msg)

	suite.Require().EqualErrorf(err, "", "")

}

func (suite *messengerServiceSuite) TestCreateGroupChat() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "testGroup"
	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}
	_, err := service.CreateGroupChat(suite.ctx, grReq)

	suite.Require().NoError(err)
	group := suite.extractGroup(service, grName)
	suite.Require().NotNil(group)
	name, ok := group.Users[suite.user.Id]
	suite.Require().True(ok)
	suite.Require().Equal(suite.user.GetDisplayName(), name)
}

func (suite *messengerServiceSuite) TestCreateGroupChatConflict() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "groupZ" //existent group
	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}

	_, err := service.CreateGroupChat(suite.ctx, grReq)

	suite.Require().EqualError(err, "group '"+grName+"' already created")
	group := suite.extractGroup(service, grName)
	suite.Require().NotNil(group)
	_, ok := group.Users[suite.user.Id]
	suite.Require().False(ok)
}

func (suite *messengerServiceSuite) TestJoinGroupChatOK() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "groupY" //existent group
	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}
	_, err := service.JoinGroupChat(suite.ctx, grReq)

	suite.Require().NoError(err)
	group := suite.extractGroup(service, grName)
	suite.Require().NotNil(group)
	name, ok := group.Users[suite.user.Id]
	suite.Require().True(ok)
	suite.Require().Equal(suite.user.GetDisplayName(), name)
}

func (suite *messengerServiceSuite) TestJoinGroupChatNotFound() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "nonExistentGroup"

	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}
	_, err := service.JoinGroupChat(suite.ctx, grReq)

	suite.Require().EqualError(err, "Unable to find group 'nonExistentGroup'")
	group := suite.extractGroup(service, grName)
	suite.Require().Nil(group)
}

func (suite *messengerServiceSuite) TestLeftGroupChatOK() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "groupX"

	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}
	_, err := service.LeftGroupChat(suite.ctx, grReq)

	suite.Require().NoError(err)
	group := suite.extractGroup(service, grName)
	suite.Require().NotNil(group)
	_, ok := group.Users[suite.user.Id]
	suite.Require().False(ok)
}

func (suite *messengerServiceSuite) TestLeftGroupChatConnNotFound() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(false)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "groupY"

	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}
	_, err := service.LeftGroupChat(suite.ctx, grReq)

	suite.Require().EqualError(err, "user '"+suite.user.DisplayName+"' not found")
	group := suite.extractGroup(service, grName)
	suite.Require().NotNil(group)
}

func (suite *messengerServiceSuite) TestLeftGroupChatLastContact() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	grName := "groupU"

	grReq := &proto.GroupRequest{
		Name: grName,
		User: &suite.user,
	}
	_, err := service.LeftGroupChat(suite.ctx, grReq)

	suite.Require().NoError(err)
	group := suite.extractGroup(service, grName)
	suite.Require().Nil(group)
}

func (suite *messengerServiceSuite) TestListChannels() {
	service := service.NewMessengerServer(suite.logger)

	conns := suite.seedConnections(true)
	utils.SetFieldValue(service, "Connections", conns)
	groups := suite.seedGroups(true)
	utils.SetFieldValue(service, "Groups", groups)

	resp, err := service.ListChannels(suite.ctx, &suite.user)

	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.Name, 6) //4 other connections + 2 groups (where use is participant)
}

func (suite *messengerServiceSuite) assertConnectionsLength(service proto.MessengerServer, expectedSize int) {
	rv := reflect.ValueOf(service)
	connLen := rv.Elem().FieldByName("Connections").Len()
	suite.Require().Equal(expectedSize, connLen)
}

func (suite *messengerServiceSuite) assertConnectionIsValid(service proto.MessengerServer, userName string) {
	rv := reflect.ValueOf(service)
	connVal := rv.Elem().FieldByName("Connections").MapIndex(reflect.ValueOf(userName))
	conn := connVal.Interface().(*model.Connection)
	suite.Require().NotNil(conn)
	suite.Require().True(conn.Active)
}

func (suite *messengerServiceSuite) extractGroup(service proto.MessengerServer, groupName string) *model.Group {
	rv := reflect.ValueOf(service)
	grVal := rv.Elem().FieldByName("Groups").MapIndex(reflect.ValueOf(groupName))
	if grVal.IsNil() {
		return nil
	}
	return grVal.Interface().(*model.Group)
}

func (suite *messengerServiceSuite) seedGroups(includeCurrent bool) map[string]*model.Group {
	groups := make(map[string]*model.Group)
	groups["groupX"] = &model.Group{
		Name:  "groupX",
		Users: createUsersMap([]string{"3253556:userA", "3356467:userB"}),
	}
	groups["groupY"] = &model.Group{
		Name:  "groupY",
		Users: createUsersMap([]string{"3356467:userB", "114435:userC"}),
	}
	groups["groupZ"] = &model.Group{
		Name:  "groupZ",
		Users: createUsersMap([]string{"3253556:userA", "114435:userC", "631767:userD"}),
	}
	if includeCurrent {
		groups["groupX"].Users[suite.user.Id] = suite.user.DisplayName
		groups["groupU"] = &model.Group{
			Name:  "groupU",
			Users: createUsersMap([]string{suite.user.Id + ":" + suite.user.DisplayName}),
		}
	}
	return groups
}

func (suite *messengerServiceSuite) seedConnections(includeCurrent bool) map[string]*model.Connection {
	conns := make(map[string]*model.Connection)
	if includeCurrent {
		conns[suite.user.DisplayName] = &model.Connection{
			Id:     suite.user.Id,
			Stream: suite.mockStream,
			Active: true,
			Err:    make(chan error),
		}
	}
	conns["userA"] = &model.Connection{
		Id:     "3253556",
		Stream: suite.mockStream,
		Active: true,
		Err:    make(chan error),
	}
	conns["userB"] = &model.Connection{
		Id:     "3356467",
		Stream: suite.mockStream,
		Active: true,
		Err:    make(chan error),
	}
	conns["userC"] = &model.Connection{
		Id:     "114435",
		Stream: suite.mockStream,
		Active: true,
		Err:    make(chan error),
	}
	conns["userD"] = &model.Connection{
		Id:     "631767",
		Stream: suite.mockStream,
		Active: true,
		Err:    make(chan error),
	}
	return conns
}

func createUsersMap(names []string) map[string]string {
	mapOfUsers := make(map[string]string)
	for _, name := range names {
		parts := strings.Split(name, ":")
		mapOfUsers[parts[0]] = parts[1]
	}
	return mapOfUsers
}
