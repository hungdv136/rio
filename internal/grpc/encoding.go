package grpc

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/golang/protobuf/jsonpb" //nolint:staticcheck
	"github.com/golang/protobuf/proto"  //nolint:staticcheck
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func marshalJSONPB(ctx context.Context, data *dynamic.Message) ([]byte, error) {
	b, err := data.MarshalJSONPB(&jsonpb.Marshaler{OrigName: true})
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return b, nil
}

func messageToMap(ctx context.Context, data *dynamic.Message) (types.Map, error) {
	b, err := marshalJSONPB(ctx, data)
	if err != nil {
		return nil, err
	}

	m, err := types.CreateMapFromReader(bytes.NewReader(b))
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return m, nil
}

func mapToMessage(ctx context.Context, inputMap types.Map, m *desc.MessageDescriptor) (*dynamic.Message, error) {
	input := dynamic.NewMessage(m)
	b, err := json.Marshal(inputMap)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if err := input.UnmarshalJSON(b); err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return input, nil
}

func convertGrpcStatus(ctx context.Context, d *Descriptor, res *rio.Response) *status.Status {
	code := codes.Code(res.StatusCode)
	if res.Error == nil {
		return status.New(code, "")
	}

	st := status.New(code, res.Error.Message)
	// Details are not allowed for status OK
	if len(res.Error.Details) == 0 || st.Code() == codes.OK {
		return st
	}

	details := make([]proto.Message, len(res.Error.Details))
	for i, detail := range res.Error.Details {
		msd, err := d.GetMessage(ctx, detail.Type)
		if err != nil {
			return status.New(codes.InvalidArgument, err.Error())
		}

		message, err := mapToMessage(ctx, detail.Value, msd)
		if err != nil {
			log.Error(ctx, "cannot encode to json", err)
			return status.New(codes.InvalidArgument, "invalid error detail")
		}

		details[i] = message
	}

	st, err := st.WithDetails(details...)
	if err != nil {
		log.Error(ctx, "cannot set error detail", err)
		return status.New(codes.InvalidArgument, "invalid error detail")
	}

	return st
}

func convertGrpcError(ctx context.Context, d *Descriptor, grpcErr error) *rio.ResponseError {
	if grpcErr == nil {
		return nil
	}

	st, ok := status.FromError(grpcErr)
	if !ok || st.Code() == codes.OK {
		return nil
	}

	res := &rio.ResponseError{Message: st.Message()}
	pbs := st.Proto()
	if pbs == nil || len(pbs.Details) == 0 {
		return res
	}

	res.Details = make([]*rio.ErrorDetail, 0, len(pbs.Details))
	for _, detail := range pbs.Details {
		msd, err := d.GetMessage(ctx, detail.TypeUrl)
		if err != nil {
			continue
		}

		dm := dynamic.NewMessage(msd)
		if err := dm.Unmarshal(detail.Value); err != nil {
			log.Error(ctx, "cannot convert detail", err)
			continue
		}

		mm, err := messageToMap(ctx, dm)
		if err != nil {
			continue
		}

		res.Details = append(res.Details, &rio.ErrorDetail{
			Type:  detail.TypeUrl,
			Value: mm,
		})
	}

	return res
}
