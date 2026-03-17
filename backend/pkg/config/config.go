package config

import (
	"log"
	"os"
)

type Config struct {
	Port     string
	LogLevel string
	Platform PlatformConfig
	DB       DBConfig
	Redis    RedisConfig
	NATS     NATSConfig
	JWT      JWTConfig
	OTel     OTelConfig
}

type PlatformConfig struct {
	WalletUserID string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (c *DBConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + c.Port +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type NATSConfig struct {
	URL string
}

type JWTConfig struct {
	Secret string
	Issuer string
}

type OTelConfig struct {
	Endpoint string
	Service  string
}

func MustLoad() *Config {
	return &Config{
		Port:     envOrDefault("PORT", "8080"),
		LogLevel: envOrDefault("LOG_LEVEL", "info"),
		Platform: PlatformConfig{
			WalletUserID: envOrDefault("PLATFORM_WALLET_USER_ID", ""),
		},
		DB: DBConfig{
			Host:     envOrDefault("DB_HOST", "localhost"),
			Port:     envOrDefault("DB_PORT", "5432"),
			User:     envOrDefault("DB_USER", "postgres"),
			Password: envOrDefault("DB_PASSWORD", "postgres"),
			Name:     envOrDefault("DB_NAME", "payments"),
			SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     envOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: envOrDefault("REDIS_PASSWORD", ""),
		},
		NATS: NATSConfig{
			URL: envOrDefault("NATS_URL", "nats://localhost:4222"),
		},
		JWT: JWTConfig{
			Secret: envOrDefault("JWT_SECRET", "dev-secret-change-me"),
			Issuer: envOrDefault("JWT_ISSUER", "payment-platform"),
		},
		OTel: OTelConfig{
			Endpoint: envOrDefault("OTEL_ENDPOINT", "localhost:4317"),
			Service:  envOrDefault("OTEL_SERVICE", "payment-platform"),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}
