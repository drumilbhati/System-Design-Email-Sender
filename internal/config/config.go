package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	GeminiAPIKey string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	SenderEmail  string
	Port         string
}

func Load() *Config {
	return &Config{
		GeminiAPIKey: getEnvOrFatal("GEMINI_API_KEY"),
		SMTPHost:     getEnvOrFatal("SMTP_HOST"),
		SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
		SMTPUser:     getEnvOrFatal("SMTP_USER"),
		SMTPPass:     getEnvOrFatal("SMTP_PASS"),
		SenderEmail:  getEnvOrFatal("SENDER_EMAIL"),
		Port:         getEnvOrDefault("PORT", "8080"),
	}
}

func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required", key)
	}
	return value
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid integer for %s, using default: %d", key, fallback)
		return fallback
	}
	return value
}
