package log

import "context"

var ctxKeyContextID = &struct{ name string }{"context_id"}

// ContextFunc unwraps the context for example gin-context
func ContextFunc(ctx context.Context) context.Context {
	return ctx
}

// GetID returns the context id from context
func GetID(ctx context.Context) string {
	value, _ := ContextFunc(ctx).Value(ctxKeyContextID).(string)
	return value
}

// SaveID saves the context id to context variable and custom context store
// Return a copy of parent in which the value associated with key is
func SaveID(ctx context.Context, contextID string) context.Context {
	ctx = context.WithValue(ContextFunc(ctx), ctxKeyContextID, contextID)
	WithContextualValues(ctx, "request_id", contextID)
	return ctx
}
