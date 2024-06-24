package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/gennadis/gigachatui/internal/client"
	"github.com/gennadis/gigachatui/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	clientID, clientSecret := os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatalf("CLIENT_ID and CLIENT_SECRET must be set in the environment")
	}

	cfg := config.NewConfig()

	chatName := promptUser("Enter a chat name: ")

	gigaChatClient, err := client.NewClient(ctx, strings.TrimSuffix(chatName, "\n"), clientID, clientSecret, *cfg)
	if err != nil {
		log.Fatalf("Failed to create API Client: %v", err)
	}

	wg := gigaChatClient.AuthHandler.Run(ctx)
	go func() {
		defer close(gigaChatClient.AuthHandler.ErrorChan)
		wg.Wait()
	}()

	for {
		question := promptUser("Ask a question: ")
		assistantResponse, err := handleQuestion(ctx, gigaChatClient, question)
		if err != nil {
			log.Printf("Error handling question: %v", err)
		}
		gigaChatClient.Session.Messages = append(gigaChatClient.Session.Messages, chat.ChatMessage{Role: chat.ChatRoleAssistant, Content: assistantResponse})

		fmt.Println(assistantResponse)
		fmt.Println(len(gigaChatClient.Session.Messages))
	}
}

func promptUser(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
	return strings.TrimSuffix(input, "\n")
}

func handleQuestion(ctx context.Context, client *client.Client, question string) (string, error) {
	userMessage := chat.ChatMessage{Role: chat.ChatRoleUser, Content: question}
	client.Session.Messages = append(client.Session.Messages, userMessage)
	request := chat.NewDefaultChatRequest(client.Session.Messages)

	resp, err := client.GetCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("Failed to get chat completion: %w", err)
	}

	return resp.Choices[0].Message.Content, nil

}
