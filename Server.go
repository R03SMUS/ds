package main

import (
	pb "github.com/rasmus/chat/chat"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
)

type server struct {
	pb.UnimplementedChatServer
	mu           sync.Mutex
	clients      map[string]pb.Chat_JoinChatServer
	logicalClock int
}

func main() {
	lis, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Initialize the server with a client map.
	s := grpc.NewServer()
	pb.RegisterChatServer(s, &server{clients: make(map[string]pb.Chat_JoinChatServer), logicalClock: 0})

	log.Printf("Server is listening on %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func (s *server) JoinChat(stream pb.Chat_JoinChatServer) error {
	var user string

	// Receive the first message to identify the user.
	firstMessage, err := stream.Recv()
	if err != nil {
		log.Fatalf("Error receiving message: %v", err)
		return err
	}
	user = firstMessage.User

	// Register the client.
	s.mu.Lock()
	s.clients[user] = stream
	s.mu.Unlock()

	// Broadcast a message to notify that the user has joined.
	var join = "Participant " + user + " joined Chitty-Chat! "

	s.broadcast(&pb.Message{User: "Server", Text: join})

	// Continuously receive messages from this client and broadcast them.
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			// The client closed the stream.
			s.mu.Lock()
			delete(s.clients, user)
			s.mu.Unlock()

			var left = "Participant " + user + " left Chitty-Chat "
			s.broadcast(&pb.Message{User: "Server", Text: left})
			break
		}
		if err != nil {
			log.Fatalf("Error receiving from stream: %v", err)
			return err
		}

		// Broadcast the received message to all connected clients.
		s.broadcast(in)
	}

	return nil
}

func (s *server) broadcast(msg *pb.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logicalClock = s.logicalClock + 1

	log.Printf("%s: %s at lamport time %s", msg.User, msg.Text, strconv.Itoa(s.logicalClock))

	msg.Text = msg.Text + " at lamport time " + strconv.Itoa(s.logicalClock)

	for _, clientStream := range s.clients {
		if err := clientStream.Send(msg); err != nil {
			log.Printf("Error broadcasting to client: %v", err)
		}
	}
}
