package api

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/log"
)

// RequestIDMiddleware adds X-Request-ID value to request, response and save to context variable
func RequestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Request.Header.Get(rio.HeaderXRequestID)
		if len(id) == 0 {
			id = uuid.NewString()
			ctx.Request.Header.Set(rio.HeaderXRequestID, id)
		}

		wrappedCtx := log.SaveID(ctx, id)
		ctx.Request = ctx.Request.WithContext(wrappedCtx)
		ctx.Writer.Header().Set(rio.HeaderXRequestID, id)
		ctx.Next()
	}
}

// RequestTimeMiddleware logs request time
func RequestTimeMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		ctx.Next()
		runTime := time.Since(startTime)

		log.Fields(ctx,
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"route_path", ctx.FullPath(),
			"status_code", ctx.Writer.Status(),
			"response_verdict", ctx.GetString("verdict"),
			"response_message", ctx.GetString("message"),
			"duration_ns", runTime.Nanoseconds(),
		).Info(ctx)
	}
}

// Recovery returns a middleware for a given writer that recovers from any panics and calls the provided handle func to handle it
func Recovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if rErr := recover(); rErr != nil {
				err, ok := rErr.(error)
				if !ok {
					err = fmt.Errorf("invalid recover error %v", rErr)
				}

				log.Fields(ctx, "error.stack", debug.Stack()).Error(ctx, "panic recovered:", err)

				// If the connection is dead, we can't write a status to it.
				if isBrokenPipeError(rErr) {
					_ = ctx.Error(err)
					ctx.Abort()
				} else {
					ctx.Abort()
					SendError(ctx, err)
				}
			}
		}()

		ctx.Next()
	}
}

// Check for a broken connection, as it is not really a condition that warrants a panic stack trace
func isBrokenPipeError(err interface{}) bool {
	ne, ok := err.(*net.OpError)
	if !ok {
		return false
	}

	se := &os.SyscallError{}
	if errors.As(ne, &se) {
		if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
			return true
		}
	}

	return false
}
