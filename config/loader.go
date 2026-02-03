package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Port       int    `yaml:"port"`
		LogLevel   string `yaml:"log_level"`
		TimeoutSec int    `yaml:"timeout_sec"`
		RateLimit  struct {
			Enabled  bool `yaml:"enabled"`
			Requests int  `yaml:"requests"`
			Window   int  `yaml:"window_seconds"`
		} `yaml:"rate_limit"`
	} `yaml:"app"`

	Telegram struct {
		BotToken         string `yaml:"bot_token"`
		WebhookApiSecret string `yaml:"webhook_secret"`
	} `yaml:"telegram"`

	GoPlus struct {
		ApiKey    string `yaml:"key"`
		ApiSecret string `yaml:"secret"`
	}
	Etherscan struct {
		URL    string `yaml:"url"`
		ApiKey string `yaml:"key"`
	}
	Alchemy struct {
		ApiKey string `yaml:"api_key"`
		URL    string `yaml:"url"`
	} `yaml:"alchemy"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	Scoring struct {
		BaseScore float64            `yaml:"base_score"`
		Weights   map[string]float64 `yaml:"weights"`
	} `yaml:"scoring"`
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func Load() (*Config, error) {
	// Load environment variables from .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Read YAML config file
	configPath := filepath.Join("config", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Override with environment variables if provided
	if goplusKey := getEnv("GOPLUS_API_KEY", ""); goplusKey != "" {
		config.GoPlus.ApiKey = goplusKey
	}
	if goplusSecret := getEnv("GOPLUS_API_SECRET", ""); goplusSecret != "" {
		config.GoPlus.ApiSecret = goplusSecret
	}
	if etherscanURL := getEnv("ETHERSCAN_API_URL", ""); etherscanURL != "" {
		config.Etherscan.URL = etherscanURL
	}
	if etherscanApiKey := getEnv("ETHERSCAN_API_KEY", ""); etherscanApiKey != "" {
		config.Etherscan.ApiKey = etherscanApiKey
	}
	if alchemyApiKey := getEnv("ALCHEMY_RPC_API_KEY", ""); alchemyApiKey != "" {
		config.Alchemy.ApiKey = alchemyApiKey
	}
	if alchemyURL := getEnv("ALCHEMY_API_URL", ""); alchemyURL != "" {
		config.Alchemy.URL = alchemyURL
	}
	if redisAddr := getEnv("REDIS_ADDR", ""); redisAddr != "" {
		config.Redis.Addr = redisAddr
	}
	if redisPassword := getEnv("REDIS_PASSWORD", ""); redisPassword != "" {
		config.Redis.Password = redisPassword
	}
	if redisDB := getEnv("REDIS_DB", ""); redisDB != "" {
		var db int
		_, err = fmt.Sscanf(redisDB, "%d", &db)
		if err == nil {
			config.Redis.DB = db
		}
	}

	if botToken := getEnv("TELEGRAM_BOT_KEY", ""); botToken != "" {
		config.Telegram.BotToken = botToken
	}
	if webhookApiSecret := getEnv("WEBHOOK_API_SECRET", ""); webhookApiSecret != "" {
		config.Telegram.WebhookApiSecret = webhookApiSecret
	}

	return &config, nil
}
