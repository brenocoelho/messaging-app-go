package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brenocoelho/messaging-app-go/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run cmd/realtime-client/main.go <server:port> <jwt_token> <chat_id>")
		fmt.Println("Example: go run cmd/realtime-client/main.go localhost:50051 <your_jwt_token> <chat_id>")
		os.Exit(1)
	}

	serverAddr := os.Args[1]
	jwtToken := os.Args[2]
	chatID := os.Args[3]

	clientConn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer clientConn.Close()

	client := proto.NewMessagesServiceClient(clientConn)

	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+jwtToken)

	fmt.Printf("Subscribing to chat: %s\n", chatID)
	stream, err := client.SubscribeToChat(ctx, &proto.SubscribeToChatRequest{
		ChatId: chatID,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	fmt.Println("Listening for messages... (Press Ctrl+C to exit)")

	// Start a goroutine to send a test message after 5 seconds
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("Sending test message...")

		resp, err := client.SendMessage(ctx, &proto.SendMessageRequest{
			ChatId:         chatID,
			Content:        "Hello from real-time client!",
			IdempotencyKey: fmt.Sprintf("client_test_%d", time.Now().Unix()),
		})
		if err != nil {
			log.Printf("Failed to send message: %v", err)
		} else {
			fmt.Printf("Message sent with ID: %s\n", resp.Message.Id)
		}
	}()

	// Listen for incoming messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Stream error: %v", err)
			break
		}

		fmt.Printf("📨 [%s]\n", msg.Content)
	}
}
