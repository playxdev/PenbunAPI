package config

import "os"

type EnvConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	FiberPort  string
	JWTSecret  string
	LogFile    string
}

var Cfg *EnvConfig

func LoadEnv() *EnvConfig {
	Cfg = &EnvConfig{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "1433"),
		DBUser:     getEnv("DB_USER", "sa"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "PENBUN"),
		FiberPort:  getEnv("FIBER_PORT", "8089"),
		JWTSecret:  getEnv("JWT_SECRET", "default-secret"),
		LogFile:    getEnv("LOG_FILE", "logs/transaction.log"),
	}
	return Cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
