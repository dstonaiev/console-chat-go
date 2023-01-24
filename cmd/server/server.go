package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/chat/internal/app/server/service"
	"github.com/chat/internal/common"
	chat "github.com/chat/proto"
	"google.golang.org/grpc"
)

var (
	port   int
	logger *log.Logger
)

func init() {
	var logFile string
	flag.IntVar(&port, "p", 8080, "server port")
	flag.StringVar(&logFile, "lf", "server.log", "file path use as server log")
	flag.Parse()
	logger = common.InitLog(logFile)
}

func main() {
	server := service.NewMessengerServer(logger)

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		logger.Fatalf("error creating the server %v\n", err)
		return
	}

	fmt.Printf("Starting server at port :%d\n", port)

	chat.RegisterMessengerServer(grpcServer, server)
	err = grpcServer.Serve(listener)
	if err != nil {
		logger.Fatalf("error starting the server %v\n", err)
		return
	}
	fmt.Println("Server started")
}
