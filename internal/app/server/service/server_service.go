package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	model "github.com/chat/internal/app/server/model"
	chat "github.com/chat/proto"
)

func NewMessengerServer(logger *log.Logger) chat.MessengerServer {
	return &messengerService{
		logger:      logger,
		Groups:      make(map[string]*model.Group),
		Connections: make(map[string]*model.Connection),
	}
}

type messengerService struct {
	chat.UnimplementedMessengerServer
	Connections map[string]*model.Connection
	Groups      map[string]*model.Group
	logger      *log.Logger
}

func (s *messengerService) Login(ctx context.Context, user *chat.User) (*chat.LoginResponse, error) {
	s.logger.Printf("user %s attempts to log in\n", user.GetDisplayName())
	conn, ok := s.Connections[user.GetDisplayName()]
	if ok && conn.Active == true {
		return &chat.LoginResponse{
			Token: conn.Id,
		}, fmt.Errorf("user %s was authenticated previously\n", user.GetDisplayName())
	}
	s.logger.Printf("user %s logged in successfully\n", user.GetDisplayName())
	return &chat.LoginResponse{Token: user.GetId()}, nil
}

func (s *messengerService) Logout(ctx context.Context, user *chat.User) (*chat.Close, error) {
	s.logger.Printf("user %s attempts to log out\n", user.GetDisplayName())
	conn, ok := s.Connections[user.GetDisplayName()]
	if !ok {
		return &chat.Close{}, fmt.Errorf("invalid logout request: connection for user %s not found", user.GetDisplayName())
	}
	conn.Stream.Context().Done()
	delete(s.Connections, user.GetDisplayName())
	s.logger.Printf("user %s logged out successfully\n", user.GetDisplayName())
	return &chat.Close{}, nil
}

func (s *messengerService) CreateStream(user *chat.User, stream chat.Messenger_CreateStreamServer) error {
	conn := &model.Connection{
		Stream: stream,
		Id:     user.GetId(),
		Active: true,
		Err:    make(chan error),
	}
	s.Connections[user.GetDisplayName()] = conn

	return <-conn.Err
}

func (s *messengerService) SendMessage(ctx context.Context, msg *chat.Message) (*chat.Close, error) {
	if msg.GetGroup() == false {
		conn, ok := s.Connections[msg.GetAddress()]
		if !ok {
			s.logger.Printf("Unable to find user '%s'\n", msg.GetAddress())
			return &chat.Close{}, fmt.Errorf("Unable to find user '%s'", msg.GetAddress())
		}
		s.send(msg, conn)
		return &chat.Close{}, <-conn.Err
	}

	wait := sync.WaitGroup{}
	done := make(chan int)

	group, ok := s.Groups[msg.GetAddress()]
	if !ok {
		return &chat.Close{}, fmt.Errorf("group '%s' is not found", msg.GetAddress())
	}
	for _, username := range group.Users {

		conn, ok := s.Connections[username]
		if ok {
			wait.Add(1)
			go func(msg *chat.Message, conn *model.Connection, username string) {
				defer wait.Done()
				s.send(msg, conn)
				s.logger.Printf("Unable to send message to group '%s' member '%s'. error: %v\n", group.Name, username, <-conn.Err)
			}(msg, conn, username)
		}
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done

	return &chat.Close{}, nil
}

func (s *messengerService) send(msg *chat.Message, conn *model.Connection) {
	if conn.Active == true {
		err := conn.Stream.Send(msg)
		s.logger.Printf("Sending message %v to user %v\n", msg.Id, conn.Id)

		if err != nil {
			s.logger.Printf("Error with stream %v. Error: %v\n", conn.Stream, err)
			conn.Active = false
			conn.Err <- err
		}
	} else {
		conn.Err <- fmt.Errorf("connection %v is not active", conn.Id)
	}
}

func (s *messengerService) CreateGroupChat(ctx context.Context, req *chat.GroupRequest) (*chat.Close, error) {
	_, ok := s.Groups[req.GetName()]
	if ok {
		s.logger.Printf("group '%s' already created\n", req.GetName())
		return &chat.Close{}, fmt.Errorf("group '%s' already created", req.GetName())
	}
	group := &model.Group{
		Name:  req.GetName(),
		Users: make(map[string]string),
	}
	group.Users[req.GetUser().GetId()] = req.GetUser().GetDisplayName()
	s.Groups[req.GetName()] = group

	return &chat.Close{}, nil
}

func (s *messengerService) JoinGroupChat(ctx context.Context, req *chat.GroupRequest) (*chat.Close, error) {
	group, ok := s.Groups[req.GetName()]
	if !ok {
		s.logger.Printf("Unable to find group '%s'\n", req.GetName())
		return &chat.Close{}, fmt.Errorf("Unable to find group '%s'", req.GetName())
	}
	group.Users[req.GetUser().GetId()] = req.GetUser().GetDisplayName()
	return &chat.Close{}, nil
}

func (s *messengerService) LeftGroupChat(ctx context.Context, req *chat.GroupRequest) (*chat.Close, error) {
	group, ok := s.Groups[req.GetName()]
	if !ok {
		s.logger.Printf("Unable to find group '%s'\n", req.GetName())
		return &chat.Close{}, fmt.Errorf("Unable to find group '%s'", req.GetName())
	}

	userName := req.GetUser().GetDisplayName()
	conn, ok := s.Connections[userName]
	if !ok {
		s.logger.Printf("user '%s' not found\n", userName)
		return &chat.Close{}, fmt.Errorf("user '%s' not found", userName)
	}
	if conn.Active == false {
		s.logger.Printf("user '%s' is inactive\n", userName)
	}
	delete(group.Users, req.GetUser().GetId())
	if len(group.Users) == 0 {
		delete(s.Groups, req.GetName())
		s.logger.Printf("Deleted group %s as no more users exists\n", req.GetName())
	}
	return &chat.Close{}, nil
}

func (s *messengerService) ListChannels(ctx context.Context, user *chat.User) (*chat.ChannelResponse, error) {
	resp := &chat.ChannelResponse{}
	for name, conn := range s.Connections {
		if conn.Active == true && name != user.GetDisplayName() {
			resp.Name = append(resp.Name, name)
		}
	}
	for name, group := range s.Groups {
		_, ok := group.Users[user.GetId()]
		if ok {
			resp.Name = append(resp.Name, "@@"+name)
		}
	}
	return resp, nil
}
