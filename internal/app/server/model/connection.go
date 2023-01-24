package server

import (
	chat "github.com/chat/proto"
)

type Connection struct {
	Id     string
	Stream chat.Messenger_CreateStreamServer
	Active bool
	Err    chan error
}
