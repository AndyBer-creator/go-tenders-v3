package config

import "os"

const (
	defaultServerAddress = "0.0.0.0:8081"
	defaultPostgresConn  = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
)

type Config struct {
	ServerAddress string
	PostgresConn  string
}

func FromEnv() Config {
	return Config{
		ServerAddress: getOrDefault("SERVER_ADDRESS", defaultServerAddress),
		PostgresConn:  getOrDefault("POSTGRES_CONN", defaultPostgresConn),
	}
}

func getOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
