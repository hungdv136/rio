package log

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/rs/zerolog"
	rs "github.com/rs/zerolog/log"
)

const (
	// Call depth level from actual caller to the Zero Log Func
	// Since we add one more level on top of Zerolog
	// Actual Caller -> Zero Log Func (Info, Warn)
	outputCallDepth = 1

	logFlags = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
)

// Setup setups logger
func Setup(module string, writer io.Writer) {
	rs.Logger = zerolog.New(writer).With().Str("service", module).Timestamp().Logger()
	defaultContextLogger := zerolog.New(writer).With().Str("service", module).Timestamp().Logger()
	zerolog.DefaultContextLogger = &defaultContextLogger
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// NewLogger returns new instance
func NewLogger() Logger {
	return &logger{callDepth: outputCallDepth}
}

// Logger implements a default logger
type logger struct {
	rs        *zerolog.Logger
	callDepth int
}

// Info calls Output to print to the standard logger with info tag
func (f *logger) Info(ctx context.Context, v ...interface{}) {
	f.msg(f.getLogger(ctx).Info(), v...)
}

// Warn calls Output to print to the standard logger with info tag
func (f *logger) Warn(ctx context.Context, v ...interface{}) {
	f.msg(f.getLogger(ctx).Warn(), v...)
}

// Debug calls Output to print to the standard logger with info tag
func (f *logger) Debug(ctx context.Context, v ...interface{}) {
	f.msg(f.getLogger(ctx).Debug(), v...)
}

// Error calls Output to print to the standard logger with error tag
func (f *logger) Error(ctx context.Context, v ...interface{}) {
	f.msg(f.getLogger(ctx).Error(), v...)
}

// Fields sets log fields
func (f *logger) Fields(ctx context.Context, keysAndValues ...interface{}) Logger {
	l := f.getLogger(ctx).With().Fields(keysAndValues).Logger()
	return &logger{rs: &l, callDepth: outputCallDepth}
}

// AddCallDepth adds call depth level
func (f *logger) AddCallDepth(i int) Logger {
	f.callDepth = outputCallDepth + i
	return f
}

// WithContextualValues sets contextual keys and values which will be shared with downstream functions
// Pass keys/values in log function if you want to write keys/values for a single log, don't use this function
// keysAndValues must be string keys and arbitrary values, and extraneous ones are ignored
func (f *logger) WithContextualValues(ctx context.Context, keysAndValues ...interface{}) context.Context {
	if f.rs != nil {
		l := f.rs.With().Fields(keysAndValues).Logger()
		f.rs = &l
	}

	ctx = UnwrapContext(ctx)
	return rs.Ctx(ctx).With().Fields(keysAndValues).Logger().WithContext(ctx)
}

func (f *logger) getLogger(ctx context.Context) *zerolog.Logger {
	if f.rs != nil {
		return f.rs
	}

	return rs.Ctx(UnwrapContext(ctx))
}

func (f *logger) msg(e *zerolog.Event, v ...interface{}) {
	if len(v) == 0 {
		e.Caller(f.callDepth + 1).Send()
		return
	}

	// TODO: Find a proper print function to avoid printing with new line
	s := fmt.Sprintln(v...)
	e.Caller(f.callDepth + 1).Msg(s[:len(s)-1])
}
