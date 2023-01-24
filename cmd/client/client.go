package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chat/internal/app/client/service"
	"github.com/chat/internal/common"
	"github.com/chat/proto"
	"github.com/manifoldco/promptui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	serverMode bool
	debugMode  bool
	host       string
	username   string

	client   service.Chat
	logger   *log.Logger
	cacheMap sync.Map
	openTalk atomic.Value
	wait     sync.WaitGroup
)

func init() {
	var logFile string
	flag.BoolVar(&debugMode, "d", false, "enable debug logging")
	flag.StringVar(&host, "h", "0.0.0.0:17100", "the chat server's host")
	flag.StringVar(&username, "n", "user1", "the username for the client")
	flag.StringVar(&logFile, "lf", "client.log", "file path use as client log")
	flag.Parse()

	logger = common.InitLog(logFile)
	cacheMap = sync.Map{}
	openTalk = atomic.Value{}
	openTalk.Store("")
	wait = sync.WaitGroup{}
}

func main() {
	ctx := context.Background()
	var err error

	logger.Println("client mode")

	connCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(connCtx, host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		logger.Fatalf("Couldn't connect to service: %v", err)
	}
	defer conn.Close()

	grpcClient := proto.NewMessengerClient(conn)

	client = service.NewChat(ctx, host, grpcClient, logger, debugMode)

	streamCl, err := client.Login(ctx, username)
	if err != nil {
		logger.Fatalf("Unable to create chat client: %v", err)
	}

	go func(str proto.Messenger_CreateStreamClient) {
		var err error
		// defer wait.Done()
		for {
			msg, err := str.Recv()

			if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
				fmt.Println("stream canceled (usually indicates shutdown)")
				break
			} else if err == io.EOF {
				fmt.Println("stream closed by server")
				break
			} else if err != nil {
				break
			}

			var key string
			if msg.Group == true { //assume message addressed to group
				key = "@@" + msg.GetAddress()
			} else { //this user, message addressed to me
				key = msg.GetUser().GetDisplayName()
			}
			if openTalk.Load().(string) == key { //if talk window open and match to current message initiator
				//then display it on the screen
				printMessage(msg)
			} else {
				//store to corresponding channel for further usage
				//TODO use lock for 3 statements?
				res, _ := cacheMap.LoadOrStore(key, make([]*proto.Message, 0))
				resArr := res.([]*proto.Message)
				resArr = append(resArr, msg)
				cacheMap.Store(key, resArr)
			}
		}

		if err != nil {
			fmt.Printf("failed to receive message: %v\n", err)
		}
	}(*streamCl)

	validator := func(input string) error {
		ln := len(input)
		if ln < 3 || ln > 30 {
			return errors.New("Group name length should be between 3 and 30")
		}
		if strings.ContainsAny(input, "!@#$%^*") {
			return fmt.Errorf("%s is not valid input for group name", input)
		}
		return nil
	}

	defer client.Logout(ctx)

	for {
		//select menu option
		prompt := promptui.Select{
			Label: "Select action",
			Items: []string{
				"Start conversation",
				"Create group chat",
				"Join group chat",
				"Leave group chat",
				"Quit",
			},
		}

		pos, menu, err := prompt.Run()

		if err != nil {
			logger.Printf("Prompt failed %v\n", err)
			return
		}

		//execute corresponding operation based of choice
		switch pos {
		case 0: //Start conversation
			contacts, err := client.ListChannels(ctx)
			if err != nil {
				fmt.Println("server error")
				logger.Printf("Get channels failed %v\n", err)
				continue
			}
			if len(contacts) == 0 {
				fmt.Println("no available contacts found.")
				fmt.Println("please try again later.")
				continue
			}
			prompt := promptui.Select{
				Label: "Select action",
				Items: contacts,
			}

			_, address, err := prompt.Run()

			if err != nil {
				logger.Printf("Prompt failed %v\n", err)
				continue
			}
			openTalk.Store(address)
			/*
				testMessages := []*proto.Message{
					{
						Id:        "4fetrytuyi",
						Address:   "Dima",
						Text:      "Good morning!",
						Group:     false,
						Timestamp: timestamppb.Now(),
					},
					{
						Id:        "4758699",
						Address:   "Alex",
						Text:      "Good afternoon!",
						Group:     false,
						Timestamp: timestamppb.Now(),
					},
					{
						Id:        "23934935406",
						Address:   "Vova",
						Text:      "Good evening!",
						Group:     false,
						Timestamp: timestamppb.Now(),
					},
				}

				res := make(chan *proto.Message)
				for _, msg := range testMessages {
					res <- msg
				}
				close(res)
			*/
			res, ok := cacheMap.LoadAndDelete(address)
			if ok {
				resArr := res.([]*proto.Message)
				for _, msg := range resArr {
					//display messages from history if any
					printMessage(msg)
				}
			}

			//			fmt.Println()
			//			time.Sleep(2 * time.Second)
			openConversation()

		case 1: // create group chat
			prompt := promptui.Prompt{
				Label:    "Enter group name:",
				Validate: validator, //check min length, spec chars
			}

			grName, err := prompt.Run()

			if err != nil {
				logger.Printf("Prompt failed %v\n", err)
			} else {
				err = client.CreateGroupChat(ctx, grName)
				if err == nil {
					openTalk.Store("@@" + grName)
					openConversation()
				}
			}
		case 2: //Join group chat
			prompt := promptui.Prompt{
				Label:    "Enter group name:",
				Validate: validator, //check min length, spec chars
			}

			grName, err := prompt.Run()

			if err != nil {
				logger.Printf("Prompt failed %v\n", err)
			} else {
				err = client.JoinGroupChat(ctx, grName)
				if err == nil {
					openTalk.Store("@@" + grName)
					openConversation()
				}
			}
		case 3: //Leave group chat
			prompt := promptui.Prompt{
				Label:    "Enter group name:",
				Validate: validator, //check min length, spec chars
			}

			grName, err := prompt.Run()

			if err != nil {
				logger.Printf("Prompt failed %v\n", err)
			} else {
				ch, ok := cacheMap.LoadAndDelete("@@" + grName)
				if ok {
					close(ch.(chan *proto.Message))
					logger.Printf("closed channel for group %s", grName)
				}
				err = client.LeftGroupChat(ctx, grName)
			}
		default: //Quit
			client.Logout(ctx)
			return
		}
		if err != nil {
			fmt.Printf("invalid operation: %s, error: %v\n", menu, err)
		}
		// select {
		// case <-interrupt():
		// 	break
		// }
		openTalk.Store("")
	}

	// <-interrupt()
}

func interrupt() chan os.Signal {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	client.Logout(context.Background())
	return interrupt
}

func printMessage(msg *proto.Message) {
	ts := msg.Timestamp.AsTime().In(time.Local)
	fmt.Printf("[%s] %s: %s\n", ts.Format("03:04:05 PM"), msg.GetUser().GetDisplayName(), msg.GetText())
}

func openConversation() {
	address := openTalk.Load().(string)
	//	fmt.Printf("Start chatting to %s. Press :q if you want to leave conversation:\n", address)
	for {
		prompt := promptui.Prompt{
			Label: ">>",
		}

		text, err := prompt.Run()
		if err != nil {
			logger.Printf("failed typing message. error: %v\n", err)
			return
		}
		if text == ":q" {
			return
		}

		go func(cl service.Chat) {
			err = cl.SendMessage(context.Background(), address, text)
			if err != nil {
				fmt.Printf("could not send message to '%s'. error: %v\n", address, err)
			}
		}(client)

		//		time.Sleep(time.Second)
	}
	//	<-interrupt()
	// select {
	// case <-interrupt():
	// 	break
	// }

}
