package config

const baseApiUrl = "https://gigachat.devices.sberbank.ru/api/v1"

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

func NewConfig(clientID, clientSecret string) *Config {
	return &Config{
		BaseURL:      baseApiUrl,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}
