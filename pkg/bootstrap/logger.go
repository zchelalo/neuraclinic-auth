package bootstrap

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger     *zap.Logger
	loggerOnce sync.Once
)

func GetLogger() *zap.Logger {
	loggerOnce.Do(func() {
		cfg := GetConfig()

		var zapCfg zap.Config
		if cfg.Environment == "development" || cfg.Environment == "dev" {
			zapCfg = zap.NewDevelopmentConfig()
			zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		} else {
			zapCfg = zap.NewProductionConfig()
			zapCfg.OutputPaths = []string{"stdout"}
			zapCfg.ErrorOutputPaths = []string{"stderr"}
			zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}

		zapCfg.EncoderConfig.TimeKey = "ts"
		zapCfg.EncoderConfig.CallerKey = "caller"
		zapCfg.DisableCaller = false

		l, err := zapCfg.Build()
		if err != nil {
			logger = zap.NewNop()
			return
		}

		logger = l.With(
			zap.String("env", cfg.Environment),
			zap.String("service", cfg.ServiceName),
		)
	})

	return logger
}

func SyncLogger() {
	if logger != nil {
		_ = logger.Sync()
	}
}
