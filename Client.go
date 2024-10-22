package main

import (
	"bufio"
	"context"
	"fmt"
	pb "github.com/rasmus/chat/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	conn, err := grpc.NewClient("localhost:42069", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Create a Chat client.
	client := pb.NewChatClient(conn)

	// Start bidirectional streaming.
	stream, err := client.JoinChat(context.Background())
	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}

	// Goroutine for receiving messages from the server
	go func() {
		for {
			// Receive a message from the server
			in, err := stream.Recv()
			if err == io.EOF {
				// The server closed the stream
				break
			}
			if err != nil {
				log.Fatalf("Failed to receive a message: %v", err)
			}
			fmt.Printf("%s: %s\n", in.User, in.Text)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if err := stream.Send(&pb.Message{User: name, Text: name + " Joined the chat"}); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	// Goroutine for sending user messages to the server
	for {
		// Read user input from the console
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if text == "/exit" || text == "/quit" {
			stream.CloseSend()
			conn.Close()
			break
		}

		// Send the message to the server
		if err := stream.Send(&pb.Message{User: name, Text: text}); err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
	}
}
