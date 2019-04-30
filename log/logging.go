package log

import (
	"context"

	"go.uber.org/zap"
)

var defaultLogger = zap.NewNop()

func SetLogger(z *zap.Logger) {
	defaultLogger = z.WithOptions(zap.AddCallerSkip(1))
}

func GetLogger() *zap.Logger {
	return defaultLogger
}

func Debug(key string, fields ...zap.Field) { defaultLogger.Debug(key, fields...) }
func Info(key string, fields ...zap.Field)  { defaultLogger.Info(key, fields...) }
func Warn(key string, fields ...zap.Field)  { defaultLogger.Warn(key, fields...) }
func Error(key string, fields ...zap.Field) { defaultLogger.Error(key, fields...) }
func Fatal(key string, fields ...zap.Field) { defaultLogger.Fatal(key, fields...) }

func Close(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		_ = defaultLogger.Sync()
		done <- struct{}{}
	}()

	select {
	case <-done:
		return
	case <-ctx.Done():
		return
	}
}
