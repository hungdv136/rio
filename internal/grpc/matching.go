package grpc

import (
	"context"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/log"
	"google.golang.org/grpc/metadata"
)

func match(ctx context.Context, r *rio.GrpcRequest, s *rio.Stub) (bool, error) {
	if s.Request == nil || s.Request.Method != rio.MethodGrpc {
		return false, nil
	}

	if matched, err := matchMethod(ctx, r, s); err != nil || !matched {
		return false, err
	}

	if matched, err := matchHeader(ctx, s); err != nil || !matched {
		return false, err
	}

	if matched, err := matchBody(ctx, r, s); err != nil || !matched {
		return false, err
	}

	return true, nil
}

func matchMethod(ctx context.Context, r *rio.GrpcRequest, s *rio.Stub) (bool, error) {
	for _, op := range s.Request.URL {
		if matched, err := rio.Match(ctx, op, r.FullMethod); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchBody(ctx context.Context, r *rio.GrpcRequest, s *rio.Stub) (bool, error) {
	if len(s.Request.Body) == 0 {
		return true, nil
	}

	input := map[string]interface{}(r.InputData)
	for _, op := range s.Request.Body {
		val, err := jsonpath.Get(op.KeyPath, input)
		if err != nil {
			if !strings.Contains(err.Error(), "unknown key") {
				log.Error(ctx, "error when executing json path", err)
				return false, err
			}
		}

		if matched, err := rio.Match(ctx, op.Operator, val); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchHeader(ctx context.Context, s *rio.Stub) (bool, error) {
	if len(s.Request.Header) == 0 {
		return true, nil
	}

	metadata, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false, nil
	}

	for _, op := range s.Request.Header {
		if matched, err := matchAnyString(ctx, op.Operator, metadata.Get(op.FieldName)); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchAnyString(ctx context.Context, op rio.Operator, arr []string) (bool, error) {
	if len(arr) == 0 {
		// Assert whether matching rule accepts empty value (rio.Empty())
		return rio.Match(ctx, op, "")
	}

	for _, val := range arr {
		if matched, err := rio.Match(ctx, op, val); err != nil || matched {
			return matched, err
		}
	}

	return false, nil
}
