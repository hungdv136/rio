package log

import "context"

var (
	ctxKeyRequestID = &struct{ name string }{"request_id"}
	ctxKeyAttrs     = &struct{ name string }{"attrs"}
)

func ContextFunc(ctx context.Context) context.Context {
	return ctx
}

// GetID returns value of key either from context
func GetID(ctx context.Context) string {
	value, _ := ContextFunc(ctx).Value(ctxKeyRequestID).(string)
	return value
}

// SaveID saves the request id to context variable and custom context store
// Return a copy of parent in which the value associated with key is
func SaveID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ContextFunc(ctx), ctxKeyRequestID, requestID)
	WithContextualValues(ctx, "request_id", requestID)
	return ctx
}
