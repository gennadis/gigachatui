package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

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
		question := promptUser("\nAsk a question: ")
		err := gigaChatClient.GetCompletion(ctx, question)
		if err != nil {
			log.Printf("Error handling question: %v", err)
		}

	}
}

func promptUser(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
	return strings.TrimSpace(input)
}
