package config

import (
	"os"
	"strings"
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
	Services ServicesConfig
	MinIO    MinIOConfig
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

// ServicesConfig holds the base addresses of all internal services.
// Used by the API gateway and by services that call each other.
type ServicesConfig struct {
	AuthAddr         string // AUTH_SERVICE_ADDR
	OrderAddr        string // ORDER_SERVICE_ADDR
	PaymentAddr      string // PAYMENT_SERVICE_ADDR
	WalletAddr       string // WALLET_SERVICE_ADDR
	ProductAddr      string // PRODUCT_SERVICE_ADDR
	CartAddr         string // CART_SERVICE_ADDR
	FraudAddr        string // FRAUD_SERVICE_ADDR
	NotificationAddr string // NOTIFICATION_SERVICE_ADDR
	StoreAddr        string // STORE_SERVICE_ADDR
}

// MinIOConfig holds connection details for the MinIO object-storage service.
type MinIOConfig struct {
	Endpoint  string // MINIO_ENDPOINT
	AccessKey string // MINIO_ACCESS_KEY
	SecretKey string // MINIO_SECRET_KEY
	Bucket    string // MINIO_BUCKET
	PublicURL string // MINIO_PUBLIC_URL
	UseSSL    bool   // MINIO_USE_SSL
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
		Services: ServicesConfig{
			AuthAddr:         envOrDefault("AUTH_SERVICE_ADDR", "http://localhost:8086"),
			OrderAddr:        envOrDefault("ORDER_SERVICE_ADDR", "http://localhost:8081"),
			PaymentAddr:      envOrDefault("PAYMENT_SERVICE_ADDR", "http://localhost:8082"),
			WalletAddr:       envOrDefault("WALLET_SERVICE_ADDR", "http://localhost:8083"),
			ProductAddr:      envOrDefault("PRODUCT_SERVICE_ADDR", "http://localhost:8087"),
			CartAddr:         envOrDefault("CART_SERVICE_ADDR", "http://localhost:8088"),
			FraudAddr:        envOrDefault("FRAUD_SERVICE_ADDR", "http://localhost:8085"),
			NotificationAddr: envOrDefault("NOTIFICATION_SERVICE_ADDR", "http://localhost:8084"),
			StoreAddr:        envOrDefault("STORE_SERVICE_ADDR", "http://localhost:8089"),
		},
		MinIO: MinIOConfig{
			Endpoint:  envOrDefault("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: envOrDefault("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: envOrDefault("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    envOrDefault("MINIO_BUCKET", "products"),
			PublicURL: envOrDefault("MINIO_PUBLIC_URL", "http://localhost:9000"),
			UseSSL:    strings.EqualFold(envOrDefault("MINIO_USE_SSL", "false"), "true"),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
