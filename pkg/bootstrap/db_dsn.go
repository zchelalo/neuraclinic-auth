package bootstrap

import (
	"fmt"
	"net/url"
)

func PostgresDSN(cfg Config) (string, error) {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPass),
		Host:   fmt.Sprintf("%s:%d", cfg.DBHost, cfg.DBPort),
		Path:   cfg.DBName,
	}

	sslMode := cfg.DBSSLMode
	if sslMode == "" {
		switch cfg.Environment {
		case "production":
			sslMode = "require"
		default:
			sslMode = "disable"
		}
	}

	q := url.Values{}
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (c Config) DBDSN() (string, error) {
	return PostgresDSN(c)
}
