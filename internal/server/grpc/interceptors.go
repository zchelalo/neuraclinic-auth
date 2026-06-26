package grpcserver

import (
	"context"
	"time"

	"github.com/zchelalo/neuraclinic-auth/internal/shared/appctx"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/i18n"
	"github.com/zchelalo/neuraclinic-auth/internal/shared/uuidx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	headerRequestID = "x-request-id"
	headerTraceID   = "x-trace-id"
	headerLanguage  = "accept-language"
)

func UnaryInterceptor(baseLogger *zap.Logger, serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		startedAt := time.Now()
		requestID := metadataValue(ctx, headerRequestID)
		if requestID == "" {
			requestID = uuidx.NewString()
		}
		traceID := metadataValue(ctx, headerTraceID)
		if traceID == "" {
			traceID = uuidx.NewString()
		}
		language := i18n.Normalize(metadataValue(ctx, headerLanguage))

		logger := baseLogger.With(
			zap.String("service", serviceName),
			zap.String("request_id", requestID),
			zap.String("trace_id", traceID),
			zap.String("grpc_method", info.FullMethod),
		)

		ctx = appctx.WithLogger(ctx, logger)
		ctx = appctx.WithRequestID(ctx, requestID)
		ctx = appctx.WithTraceID(ctx, traceID)
		ctx = appctx.WithLanguage(ctx, language)

		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", zap.Any("panic", recovered))
				err = status.Error(codes.Internal, i18n.Message(appctx.Language(ctx), i18n.KeyInternalServerError))
			}

			code := status.Code(err)
			fields := []zap.Field{
				zap.String("grpc_code", code.String()),
				zap.Duration("duration", time.Since(startedAt)),
			}
			if err != nil {
				fields = append(fields, zap.Error(err))
			}

			if code == codes.OK {
				logger.Info("grpc request completed", fields...)
				return
			}
			logger.Warn("grpc request failed", fields...)
		}()

		return handler(ctx, req)
	}
}

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
