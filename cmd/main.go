package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zchelalo/neuraclinic-auth/pkg/bootstrap"
	"go.uber.org/zap"
)

func main() {
	cfg, err := bootstrap.LoadConfig(".env")
	if err != nil {
		panic(err)
	}

	logger := bootstrap.GetLogger()
	defer bootstrap.SyncLogger()

	app, err := bootstrap.InitApp(context.Background(), logger, cfg)
	if err != nil {
		logger.Fatal("cannot initialize application", zap.Error(err))
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("grpc server starting", zap.Int("port", cfg.Port))
		errCh <- app.Server.Start()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		logger.Info("signal received, shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		logger.Error("grpc server stopped", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = app.Cleanup(ctx)
	logger.Info("shutdown complete")
}
