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
	"math"
	"os"
	"strings"
)

func main() {
	var locallamport int64 = 1

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
			locallamport = int64(math.Max(float64(locallamport), float64(in.Lamport)) + 1) //this seems stupid. lol

			log.Printf("%s: %s %d\n", in.User, in.Text, in.Lamport)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if err := stream.Send(&pb.Message{User: name, Text: name + " Joined the chat", Lamport: locallamport}); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	// Goroutine for sending user messages to the server
	for {
		// Read user input from the console
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if len(text) > 128 { //IIRC chinese charecters and emojis, and such counts for 2 (sometimes?) correct way would be converting to runes and counting them maybe.
			log.Printf("message too long!")
			continue
		}

		if text == "/exit" || text == "/quit" {
			stream.CloseSend()
			conn.Close()
			break
		}
		locallamport++
		// Send the message to the server

		if err := stream.Send(&pb.Message{User: name, Text: text, Lamport: locallamport}); err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
	}
}
