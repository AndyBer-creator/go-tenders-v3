package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddress   string
	PostgresConn    string
	PostgresUser    string
	PostgresPass    string
	PostgresHost    string
	PostgresPort    string
	PostgresDB      string
	DatabaseSSLMode string
}

func LoadConfig() *Config {
	// Загружаем из .env в переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	cfg := &Config{
		ServerAddress:   getEnv("SERVER_ADDRESS", "0.0.0.0:8080"),
		PostgresConn:    getEnv("POSTGRES_CONN", ""),
		PostgresUser:    getEnv("POSTGRES_USERNAME", ""),
		PostgresPass:    getEnv("POSTGRES_PASSWORD", ""),
		PostgresHost:    getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:    getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:      getEnv("POSTGRES_DATABASE", ""),
		DatabaseSSLMode: getEnv("DATABASE_SSLMODE", "disable"),
	}

	// Если нет полной строки подключения, собрать из параметров
	if cfg.PostgresConn == "" {
		if cfg.PostgresUser == "" || cfg.PostgresPass == "" || cfg.PostgresHost == "" || cfg.PostgresDB == "" {
			log.Fatal("DB connection params missing in environment")
		}
		cfg.PostgresConn = "postgres://" + cfg.PostgresUser + ":" + cfg.PostgresPass + "@" +
			cfg.PostgresHost + ":" + cfg.PostgresPort + "/" + cfg.PostgresDB + "?sslmode=" + cfg.DatabaseSSLMode
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
