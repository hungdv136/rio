package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
)

const (
	VerdictSuccess           = "success"
	VerdictFailure           = "failure"
	VerdictMissingParameters = "missing_parameters"
	VerdictInvalidParameters = "invalid_parameters"
)

// SendJSON sends JSON
func SendJSON(ctx *gin.Context, statusCode int, verdict string, message string, data interface{}) {
	ctx.Set("status_code", statusCode)
	ctx.Set("verdict", verdict)
	ctx.Set("message", message)
	ctx.Set("data", data)

	ctx.JSON(statusCode, gin.H{
		"verdict": verdict,
		"message": message,
		"data":    data,
		"time":    time.Now().Format(time.RFC3339),
	})
}

// SendError sends error
func SendError(ctx *gin.Context, _ error) {
	message := "Unexpected error. Error ID: " + log.GetID(ctx)
	SendJSON(ctx, http.StatusInternalServerError, VerdictFailure, message, types.Map{})
}

// SendSuccess sends success response
func SendSuccess(ctx *gin.Context, message string, data interface{}) {
	SendJSON(ctx, http.StatusOK, VerdictSuccess, message, data)
}

// AbortJSON abort with JSON
func AbortJSON(ctx *gin.Context, statusCode int, verdict string, message string, data interface{}) {
	ctx.Abort()
	SendJSON(ctx, statusCode, verdict, message, data)
}
