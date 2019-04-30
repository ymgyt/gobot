package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/ymgyt/gobot/di"
	"github.com/ymgyt/gobot/log"
)

func main() {
	ctx := watchSignal()
	service, cleanup := di.InitializeService(ctx)
	defer cleanup()
	service.Run(ctx)
}

func watchSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	// see https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods
	// at least, we should handle term
	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	// nolint:gocritic
	go func() {
		sig := <-ch
		switch sig {
		default:
			log.Info("receive signal", zap.String("signal", sig.String()))
			cancel()
		}
	}()

	return ctx
}
