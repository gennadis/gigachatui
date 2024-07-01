package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/gennadis/gigachatui/internal/auth"
	"github.com/gennadis/gigachatui/internal/chat"
	"github.com/gennadis/gigachatui/internal/client"
	"github.com/gennadis/gigachatui/internal/config"
	"github.com/gennadis/gigachatui/storage"
	"github.com/joho/godotenv"
)

const databaseFilePath = "./sqlite.db"

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

	// Initialize database
	dataDB, err := storage.NewSqliteDB(databaseFilePath)
	if err != nil {
		log.Fatalf("failed to create database file %s: %v", databaseFilePath, err)
	}
	slog.Debug("database filepath", "path", databaseFilePath)

	// Make store and load sessions
	sessionsStore, err := storage.NewSessions(dataDB)
	if err != nil {
		log.Fatalf("failed to make sessions store: %v", err)
	}
	// Make store and load messages
	messagesStore, err := storage.NewMessages(dataDB)
	if err != nil {
		log.Fatalf("failed to make messages store: %v", err)
	}

	authManager, err := auth.NewManager(ctx, clientID, clientSecret)
	if err != nil {
		log.Fatalf("failed to init auth manager: %v", err)
	}

	// Create a new GigaChat client
	gcc, err := client.NewClient(*cfg, *authManager, *sessionsStore, *messagesStore)
	if err != nil {
		log.Fatalf("failed to create GigaChat API client: %v", err)
	}

	// Run the authentication handler in a separate goroutine
	wg := gcc.AuthManager.Run(ctx)
	go func() {
		defer close(gcc.AuthManager.ErrorChan)
		wg.Wait()
	}()

	// Prompt user for chat name
	chatName, err := promptUser("Enter a chat name: ")
	if err != nil {
		log.Fatalf("failed to handle chat name user prompt: %v", err)
	}

	// Create new Session
	session := chat.NewSession(chatName)
	if err := gcc.SessionStorage.Write(*session); err != nil {
		log.Fatalf("failed to write session to storage: %s", err)
	}

	// Main loop to handle user questions
	for {
		userPromt, err := promptUser("\nAsk a question: ")
		if err != nil {
			log.Fatalf("failed to handle user question prompt: %v", err)
		}

		if err := gcc.RequestCompletion(ctx, session.ID, userPromt); err != nil {
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
		return "", fmt.Errorf("failed to read user input: %w", err)
	}
	return strings.TrimSpace(inp), nil
}
