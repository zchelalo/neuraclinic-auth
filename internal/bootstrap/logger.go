package bootstrap

import "go.uber.org/zap"

func NewLogger(environment string) (*zap.Logger, error) {
	if environment == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
