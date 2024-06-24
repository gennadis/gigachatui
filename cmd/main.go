package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/gennadis/gigachatui/internal/client"
	"github.com/gennadis/gigachatui/internal/config"

	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	godotenv.Load(".env")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	cfg := config.NewConfig()

	gigaChatClient, err := client.NewClient(ctx, clientID, clientSecret, *cfg)
	if err != nil {
		log.Fatalf("Failed to create API Client: %s", err)
	}

	wg := gigaChatClient.AuthHandler.Run(ctx)
	go func() {
		defer close(gigaChatClient.AuthHandler.ErrorChan)
		wg.Wait()
	}()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Ask a question: ")
	question, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	message := client.ChatMessage{Role: client.ChatRoleUser, Content: question}
	request := client.NewDefaultChatRequest(message)

	resp, err := gigaChatClient.GetComplition(ctx, request)
	if err != nil {
		slog.Error("Failed to get chat completion", "error", err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
