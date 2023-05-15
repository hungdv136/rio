package util

import (
	"context"
	"fmt"

	"github.com/hungdv136/rio/internal/log"
)

// ToArrayString converts from array of interface{} to array of string
func ToArrayString(a []interface{}) ([]string, bool) {
	var ok bool
	b := make([]string, len(a))
	for i, e := range a {
		b[i], ok = e.(string)
		if !ok {
			return nil, false
		}
	}
	return b, true
}

// ToString converts anything to string
func ToString(any interface{}) string {
	switch any.(type) {
	case float32, float64:
		return fmt.Sprintf("%.6f", any)
	default:
		return fmt.Sprintf("%v", any)
	}
}

// ArrayContains checks if array a[] contains value v or not
func ArrayContains[R comparable](a []R, v R) bool {
	for _, e := range a {
		if e == v {
			return true
		}
	}

	return false
}

// CloseSilently closes and write log if an error occurs
func CloseSilently(ctx context.Context, close func() error) {
	if err := close(); err != nil {
		log.Error(ctx, "cannot close", err)
	}
}
