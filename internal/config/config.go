package config

// baseAPIURL is the default base URL for the GigaChat API
const baseAPIURL = "https://gigachat.devices.sberbank.ru/api/v1"

// Config holds the configuration for the GigaChat API client
type Config struct {
	BaseURL string
}

// NewConfig creates a new Config instance with default values
func NewConfig() (*Config, error) {
	return &Config{
		BaseURL: baseAPIURL,
	}, nil
}
