package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var (
	config   Config
	configMu sync.RWMutex
)

type Config struct {
	Environment string
	ServiceName string
	Port        int

	DBHost    string
	DBPort    int
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string

	GRPCTLSCertPath string
	GRPCTLSKeyPath  string

	UsersGRPCAddr               string
	UsersGRPCTLSEnabled         bool
	UsersGRPCCACertPath         string
	UsersGRPCInsecureSkipVerify bool
	InternalServiceToken        string

	AccessTokenTTL           time.Duration
	RefreshTokenTTL          time.Duration
	PasswordResetOTPTTL      time.Duration
	PasswordResetTokenTTL    time.Duration
	PasswordResetMaxAttempts int
	TokenHashSecret          string

	JWTAccessPrivateKeyPath  string
	JWTAccessPublicKeyPath   string
	JWTRefreshPrivateKeyPath string
	JWTRefreshPublicKeyPath  string

	RabbitMQURL                     string
	RabbitMQExchange                string
	RabbitMQPasswordResetRoutingKey string
}

func LoadConfig(dotenvPath string) (Config, error) {
	if dotenvPath != "" {
		_ = godotenv.Load(dotenvPath)
	}

	cfg := Config{
		Environment:                     getEnv("ENVIRONMENT", "development"),
		ServiceName:                     getEnv("SERVICE_NAME", "neuraclinic-auth"),
		Port:                            getEnvInt("PORT", 8000),
		DBHost:                          getEnv("DB_HOST", ""),
		DBPort:                          getEnvInt("DB_PORT", 5432),
		DBUser:                          getEnv("DB_USER", ""),
		DBPass:                          getEnv("DB_PASS", ""),
		DBName:                          getEnv("DB_NAME", ""),
		DBSSLMode:                       getEnv("DB_SSLMODE", "disable"),
		GRPCTLSCertPath:                 getEnv("GRPC_TLS_CERT_PATH", ""),
		GRPCTLSKeyPath:                  getEnv("GRPC_TLS_KEY_PATH", ""),
		UsersGRPCAddr:                   getEnv("USERS_GRPC_ADDR", ""),
		UsersGRPCTLSEnabled:             getEnvBool("USERS_GRPC_TLS_ENABLED", true),
		UsersGRPCCACertPath:             getEnv("USERS_GRPC_CA_CERT_PATH", ""),
		UsersGRPCInsecureSkipVerify:     getEnvBool("USERS_GRPC_INSECURE_SKIP_VERIFY", false),
		InternalServiceToken:            getEnv("INTERNAL_SERVICE_TOKEN", ""),
		AccessTokenTTL:                  getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:                 getEnvDuration("REFRESH_TOKEN_TTL", 720*time.Hour),
		PasswordResetOTPTTL:             getEnvDuration("PASSWORD_RESET_OTP_TTL", 10*time.Minute),
		PasswordResetTokenTTL:           getEnvDuration("PASSWORD_RESET_TOKEN_TTL", 15*time.Minute),
		PasswordResetMaxAttempts:        getEnvInt("PASSWORD_RESET_MAX_ATTEMPTS", 5),
		TokenHashSecret:                 getEnv("TOKEN_HASH_SECRET", ""),
		JWTAccessPrivateKeyPath:         getEnv("JWT_ACCESS_PRIVATE_KEY_PATH", ""),
		JWTAccessPublicKeyPath:          getEnv("JWT_ACCESS_PUBLIC_KEY_PATH", ""),
		JWTRefreshPrivateKeyPath:        getEnv("JWT_REFRESH_PRIVATE_KEY_PATH", ""),
		JWTRefreshPublicKeyPath:         getEnv("JWT_REFRESH_PUBLIC_KEY_PATH", ""),
		RabbitMQURL:                     getEnv("RABBITMQ_URL", ""),
		RabbitMQExchange:                getEnv("RABBITMQ_EXCHANGE", "neuraclinic.events"),
		RabbitMQPasswordResetRoutingKey: getEnv("RABBITMQ_PASSWORD_RESET_ROUTING_KEY", "auth.password_reset_requested.v1"),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	setConfig(cfg)
	return cfg, nil
}

func GetConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return config
}

func setConfig(cfg Config) {
	configMu.Lock()
	config = cfg
	configMu.Unlock()
}

func (c Config) Validate() error {
	required := map[string]string{
		"DB_HOST":                      c.DBHost,
		"DB_USER":                      c.DBUser,
		"DB_PASS":                      c.DBPass,
		"DB_NAME":                      c.DBName,
		"GRPC_TLS_CERT_PATH":           c.GRPCTLSCertPath,
		"GRPC_TLS_KEY_PATH":            c.GRPCTLSKeyPath,
		"USERS_GRPC_ADDR":              c.UsersGRPCAddr,
		"INTERNAL_SERVICE_TOKEN":       c.InternalServiceToken,
		"TOKEN_HASH_SECRET":            c.TokenHashSecret,
		"JWT_ACCESS_PRIVATE_KEY_PATH":  c.JWTAccessPrivateKeyPath,
		"JWT_ACCESS_PUBLIC_KEY_PATH":   c.JWTAccessPublicKeyPath,
		"JWT_REFRESH_PRIVATE_KEY_PATH": c.JWTRefreshPrivateKeyPath,
		"JWT_REFRESH_PUBLIC_KEY_PATH":  c.JWTRefreshPublicKeyPath,
	}

	for key, value := range required {
		if value == "" {
			return fmt.Errorf("missing required config key: %s", key)
		}
	}

	if c.Port <= 0 {
		return fmt.Errorf("PORT must be greater than zero")
	}
	if c.AccessTokenTTL <= 0 || c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("token TTLs must be greater than zero")
	}
	if c.AccessTokenTTL >= c.RefreshTokenTTL {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be smaller than REFRESH_TOKEN_TTL")
	}
	if c.PasswordResetMaxAttempts <= 0 {
		return fmt.Errorf("PASSWORD_RESET_MAX_ATTEMPTS must be greater than zero")
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
