package log

import (
	"context"
	"os"
)

// Logger defines the default logger
var defaultLogger Logger = func() Logger {
	Setup("rio", os.Stdout)
	return NewLogger().AddCallDepth(1)
}()

// Logger define interface for logger
type Logger interface {
	Info(ctx context.Context, v ...interface{})
	Warn(ctx context.Context, v ...interface{})
	Debug(ctx context.Context, v ...interface{})
	Error(ctx context.Context, v ...interface{})
	Fields(ctx context.Context, kvList ...interface{}) Logger
	AddCallDepth(i int) Logger
	WithContextualValues(ctx context.Context, kvList ...interface{}) context.Context
}

// Info logs with info level
func Info(ctx context.Context, v ...interface{}) {
	defaultLogger.Info(ctx, v...)
}

// Warn logs with warn level
func Warn(ctx context.Context, v ...interface{}) {
	defaultLogger.Warn(ctx, v...)
}

// Debug logs with debug level
func Debug(ctx context.Context, v ...interface{}) {
	defaultLogger.Debug(ctx, v...)
}

// Error logs with error level
func Error(ctx context.Context, v ...interface{}) {
	defaultLogger.Error(ctx, v...)
}

// Fields sets log fields for a single log line
// Info, Warn, Debug, Error must be called. Otherwise, log will be ignored
func Fields(ctx context.Context, keysAndValues ...interface{}) Logger {
	return defaultLogger.Fields(ctx, keysAndValues...)
}

// WithContextualValues sets contextual keys and values which will be shared with downstream functions
// Pass keys and values in log functions if you want to write keys/values for a single log
// keysAndValues must be string keys and arbitrary values, and extraneous ones are ignored
func WithContextualValues(ctx context.Context, keysAndValues ...interface{}) context.Context {
	return defaultLogger.WithContextualValues(ctx, keysAndValues...)
}
