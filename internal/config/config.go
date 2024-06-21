package config

const baseApiUrl = "https://gigachat.devices.sberbank.ru/api/v1"

type Config struct {
	BaseURL string
}

func NewConfig() *Config {
	return &Config{
		BaseURL: baseApiUrl,
	}
}
