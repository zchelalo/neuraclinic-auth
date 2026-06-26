package appctx

import (
	"context"

	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
	"go.uber.org/zap"
)

type key string

const (
	loggerKey    key = "logger"
	requestIDKey key = "request_id"
	traceIDKey   key = "trace_id"
	languageKey  key = "language"
)

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func Logger(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok && logger != nil {
		return logger
	}
	return zap.NewNop()
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDKey).(string); ok {
		return value
	}
	return ""
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func TraceID(ctx context.Context) string {
	if value, ok := ctx.Value(traceIDKey).(string); ok {
		return value
	}
	return ""
}

func WithLanguage(ctx context.Context, language i18n.Language) context.Context {
	return context.WithValue(ctx, languageKey, language)
}

func Language(ctx context.Context) i18n.Language {
	if value, ok := ctx.Value(languageKey).(i18n.Language); ok {
		return value
	}
	return i18n.English
}
