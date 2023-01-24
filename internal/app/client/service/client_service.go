package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	chat "github.com/chat/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const timeFormat = "03:04:05 PM"

type Chat interface {
	SendMessage(context.Context, string, string) error
	ListChannels(context.Context) ([]string, error)
	Login(context.Context, string) (*chat.Messenger_CreateStreamClient, error)
	Logout(context.Context) error
	CreateGroupChat(context.Context, string) error
	JoinGroupChat(context.Context, string) error
	LeftGroupChat(context.Context, string) error
}

type chatApp struct {
	user *chat.User
	//isGroup    bool
	grpcClient chat.MessengerClient
	client     chat.Messenger_CreateStreamClient
	Host       string
	logger     *log.Logger
	debugMode  bool
}

func NewChat(ctx context.Context, host string, grpclient chat.MessengerClient, logger *log.Logger, debugMode bool) Chat {
	chatApp := &chatApp{
		grpcClient: grpclient,
		Host:       host,
		logger:     logger,
		debugMode:  debugMode,
	}
	return chatApp
}

func (c *chatApp) connect(ctx context.Context) error {
	var err error

	c.client, err = c.grpcClient.CreateStream(ctx, c.user)
	if err != nil {
		return err
	}
	//	defer c.client.CloseSend()

	c.clientLogf("connected to stream")

	return err
}

func toMsgAddress(address string) (string, bool) {
	if strings.HasPrefix(address, "@@") {
		return address[2:], true
	}
	return address, false
}

func (c *chatApp) SendMessage(ctx context.Context, address, text string) error {
	addr, isGroup := toMsgAddress(address)
	_, err := c.grpcClient.SendMessage(ctx, &chat.Message{
		Id:        c.user.GetId(),
		User:      c.user,
		Text:      text,
		Timestamp: timestamppb.Now(),
		Address:   addr,
		Group:     isGroup,
	})
	if err != nil {
		c.clientLogf("failed to send message: %v", err)
		return err
	}
	return nil
}

func (c *chatApp) ListChannels(ctx context.Context) ([]string, error) {
	res, err := c.grpcClient.ListChannels(ctx, c.user)
	if err != nil {
		return nil, err
	}
	return res.Name, nil
}

func (c *chatApp) Login(ctx context.Context, username string) (*chat.Messenger_CreateStreamClient, error) {
	ts := time.Now()
	id := sha256.Sum256([]byte(ts.String() + *&username))
	c.user = &chat.User{
		Id:          hex.EncodeToString(id[:]),
		DisplayName: *&username,
	}

	_, err := c.grpcClient.Login(ctx, c.user)
	if err != nil {
		return nil, err
	}

	err = c.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to the server, error: %v", err)
	}

	fmt.Printf("user %s logged in successfully", username)
	return &c.client, nil
}

func (c *chatApp) Logout(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err := c.grpcClient.Logout(ctx, c.user)
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unavailable {
		c.debugLogf("unable to logout (connection already closed)")
		return nil
	}

	return err
}

func (c *chatApp) CreateGroupChat(ctx context.Context, name string) error {
	req := &chat.GroupRequest{
		Name: name,
		User: c.user,
	}
	_, err := c.grpcClient.CreateGroupChat(ctx, req)
	return err
}

func (c *chatApp) JoinGroupChat(ctx context.Context, name string) error {
	req := &chat.GroupRequest{
		Name: name,
		User: c.user,
	}
	_, err := c.grpcClient.JoinGroupChat(ctx, req)
	return err
}

func (c *chatApp) LeftGroupChat(ctx context.Context, name string) error {
	req := &chat.GroupRequest{
		Name: name,
		User: c.user,
	}
	_, err := c.grpcClient.LeftGroupChat(ctx, req)
	return err
}

func (c *chatApp) clientLogf(format string, args ...interface{}) {
	c.logger.Printf("[%s] <<Client>>: "+format, append([]interface{}{time.Now().Format(timeFormat)}, args...)...)
}

func (c *chatApp) debugLogf(format string, args ...interface{}) {
	if c.debugMode {
		c.logger.Printf("[%s] <<Debug>>: "+format, append([]interface{}{time.Now().Format(timeFormat)}, args...)...)
	}
}
