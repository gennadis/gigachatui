package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/gennadis/gigachatui/internal/client"
	"github.com/gennadis/gigachatui/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Load environment variables from .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("failed to load `.env` file: %v", err)
	}

	// Retrieve client ID and client secret from environment variables
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatalf("CLIENT_ID or CLIENT_SECRET must be set in the environment")
	}

	// Initialize configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to create config: %v", err)
	}

	// Prompt user for chat name
	chatName, err := promptUser("Enter a chat name: ")
	if err != nil {
		log.Fatalf("failed to handle chat name user prompt: %v", err)
	}

	// Create a new GigaChat client
	gcc, err := client.NewClient(ctx, *cfg, chatName, clientID, clientSecret)
	if err != nil {
		log.Fatalf("failed to create GigaChat API client: %v", err)
	}

	// Run the authentication handler in a separate goroutine
	wg := gcc.AuthManager.Run(ctx)
	go func() {
		defer close(gcc.AuthManager.ErrorChan)
		wg.Wait()
	}()

	// Main loop to handle user questions
	for {
		userPromt, err := promptUser("\nAsk a question: ")
		if err != nil {
			log.Fatalf("failed to handle user question prompt: %v", err)
		}

		if err := gcc.GetCompletion(ctx, userPromt); err != nil {
			slog.Error("failed to handle user promt completion", "error", err)
		}
	}
}

// promptUser prompts the user with a given message and returns the input
func promptUser(prompt string) (string, error) {
	r := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	inp, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %v", err)
	}
	return strings.TrimSpace(inp), nil
}
