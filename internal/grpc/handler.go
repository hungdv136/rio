package grpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/log"
	fs "github.com/hungdv136/rio/internal/storage"
	"github.com/hungdv136/rio/internal/types"
	"github.com/hungdv136/rio/internal/util"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type requestContext struct {
	fullMethod string
	stream     grpc.ServerStream
	methodDesc *desc.MethodDescriptor
	protoInput *dynamic.Message
	jsonInput  []byte
	mapInput   types.Map
	stub       *rio.Stub
	descriptor *Descriptor
}

type handler struct {
	descriptor  *ServiceDescriptor
	stubStore   rio.StubStore
	fileStorage fs.FileStorage
}

func newHandler(stubStore rio.StubStore, fileStorage fs.FileStorage, descriptor *ServiceDescriptor) *handler {
	return &handler{
		descriptor:  descriptor,
		stubStore:   stubStore,
		fileStorage: fileStorage,
	}
}

// handleRequest is a generic handler to handle incoming grpc stream
func (h *handler) handleRequest(srv interface{}, stream grpc.ServerStream) error {
	ctx := stream.Context()
	tranStream := grpc.ServerTransportStreamFromContext(ctx)
	if tranStream == nil {
		err := errors.New("cannot get transport from context")
		log.Error(ctx, err)
		return err
	}

	log.Info(ctx, "received grpc with full method", tranStream.Method())

	fullMethod := tranStream.Method()
	incomingRequest := captureIncomingRequest(ctx, fullMethod)

	defer util.CloseSilently(ctx, func() error {
		return h.stubStore.CreateIncomingRequest(ctx, incomingRequest)
	})

	descriptor, err := h.getProtoDescriptor(ctx, fullMethod)
	if err != nil {
		return err
	}

	methodDesc, err := descriptor.GetMethod(ctx, fullMethod)
	if err != nil {
		return err
	}

	rawInput := dynamic.NewMessage(methodDesc.GetInputType())
	if err := stream.RecvMsg(rawInput); err != nil {
		log.Error(ctx, "cannot parse input", err)
		return err
	}

	inputData, err := marshalJSONPB(ctx, rawInput)
	if err != nil {
		return err
	}

	inputMap, err := types.CreateMapFromReader(bytes.NewReader(inputData))
	if err != nil {
		log.Error(ctx, "cannot marshal input to map", err)
		return err
	}

	grpcRequest := &rio.GrpcRequest{FullMethod: fullMethod, InputData: inputMap}
	stub, err := h.getMatchedStub(ctx, grpcRequest)
	if err != nil {
		return err
	}

	incomingRequest.StubID = stub.ID
	incomingRequest.Tag = stub.Tag
	incomingRequest.Body = inputData

	if stub.Settings.DeactivateWhenMatched {
		log.Info(ctx, "remove used stub", stub.ID)
		if err := h.stubStore.Delete(ctx, stub.ID); err != nil {
			log.Error(ctx, "cannot delete stub, ignore error")
		}
	}

	if stub.Settings.DelayDuration > 0 {
		log.Info(ctx, "delay response", stub.Settings.DelayDuration)
		time.Sleep(stub.Settings.DelayDuration)
	}

	reqCtx := &requestContext{
		fullMethod: fullMethod,
		stream:     stream,
		methodDesc: methodDesc,
		protoInput: rawInput,
		jsonInput:  inputData,
		stub:       stub,
		descriptor: descriptor,
	}

	if stub.IsReversed() {
		return h.reverseProxy(ctx, reqCtx)
	}

	if err := h.processResponse(ctx, reqCtx); err != nil {
		return err
	}

	return writeGrpcResponse(ctx, reqCtx)
}

func (h *handler) getProtoDescriptor(ctx context.Context, fullMethod string) (*Descriptor, error) {
	protos, err := h.stubStore.GetProtos(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range protos {
		if util.ArrayContains(p.Methods, fullMethod) {
			return h.descriptor.GetDescriptor(ctx, p.FileID)
		}
	}

	err = status.Errorf(codes.NotFound, "no proto for %s", fullMethod)
	log.Error(ctx, err)
	return nil, err
}

func (h *handler) getMatchedStub(ctx context.Context, r *rio.GrpcRequest) (*rio.Stub, error) {
	stubs, err := h.stubStore.GetAll(ctx, "")
	if err != nil {
		return nil, err
	}

	if len(stubs) == 0 {
		err := status.Errorf(codes.NotFound, "no stub for %s", r.FullMethod)
		log.Error(ctx, err)
		return nil, err
	}

	matchedStubs := make([]*rio.Stub, 0, len(stubs))
	for _, stub := range stubs {
		matched, err := match(ctx, r, stub)
		if err != nil {
			return nil, err
		}

		if matched {
			matchedStubs = append(matchedStubs, stub)
		}
	}

	if len(matchedStubs) == 0 {
		err := status.Errorf(codes.NotFound, "no matched stub found for %s", r.FullMethod)
		log.Error(ctx, err)
		return nil, err
	}

	stub := rio.SelectStubs(matchedStubs)
	log.Info(ctx, "matched stub", stub.ID, stub.Description, "nb stubs", len(stubs))
	return stub, nil
}

func (h *handler) processResponse(ctx context.Context, r *requestContext) error {
	if len(r.stub.Response.BodyFile) > 0 {
		if err := r.stub.Response.LoadBodyFromFile(ctx, h.fileStorage); err != nil {
			return err
		}
	}

	if r.stub.HasTemplate() {
		grpcRequest := &rio.GrpcRequest{FullMethod: r.fullMethod, InputData: r.mapInput}
		if err := r.stub.Response.LoadBodyFromTemplate(ctx, &rio.TemplateData{Grpc: grpcRequest}); err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) reverseProxy(ctx context.Context, r *requestContext) error {
	log.Info(ctx, "forward", getFullMethod(r.methodDesc), "to", r.stub.Proxy.TargetURL)

	md, _ := metadata.FromIncomingContext(ctx)
	ctx = metadata.NewOutgoingContext(ctx, md.Copy())
	ctx = metadata.AppendToOutgoingContext(ctx, "X-PROXY", "rio")

	header := metadata.MD{}
	output, grpcErr := invokeGrpc(ctx, r.stub.Proxy.TargetURL, r.methodDesc, r.protoInput, grpc.Header(&header))

	if len(header) > 0 {
		if err := r.stream.SendHeader(header); err != nil {
			log.Error(ctx, "cannot send header", err)
			return err
		}
	}

	// If error occurs, then output might contain illegal data
	if output != nil && grpcErr == nil {
		if err := r.stream.SendMsg(output); err != nil {
			log.Error(ctx, "cannot send message", err)
			return err
		}
	}

	if err := h.recordResponse(ctx, r, output, grpcErr); err != nil {
		log.Error(ctx, "cannot record response", err)
	}

	return grpcErr
}

func (h *handler) recordResponse(ctx context.Context, r *requestContext, output *dynamic.Message, grpcErr error) error {
	if !r.stub.Proxy.EnableRecord {
		return nil
	}

	clonedStub := r.stub.Clone()
	clonedStub.ID = 0
	clonedStub.Description = fmt.Sprintf("Proxy record from stub id %d", r.stub.ID)
	clonedStub.Proxy = &rio.Proxy{}
	clonedStub.Active = false
	clonedStub.Response = rio.NewResponse()
	clonedStub.Response.Error = convertGrpcError(ctx, r.descriptor, grpcErr)
	clonedStub.Tag = rio.TagRecordedStub

	if st, ok := status.FromError(grpcErr); ok {
		clonedStub.Response.StatusCode = int(st.Code())
	}

	if output != nil {
		body, err := marshalJSONPB(ctx, output)
		if err != nil {
			return err
		}

		clonedStub.Response = clonedStub.Response.WithBody(rio.ContentTypeJSON, body)
	}

	if err := h.stubStore.Create(ctx, clonedStub); err != nil {
		return err
	}

	log.Info(ctx, "recording has been created in stub id", clonedStub.ID)
	return nil
}

func writeGrpcResponse(ctx context.Context, r *requestContext) error {
	if len(r.stub.Response.Header) > 0 {
		if err := r.stream.SendHeader(metadata.New(r.stub.Response.Header)); err != nil {
			log.Error(ctx, "cannot send header", err)
			return err
		}
	}

	if len(r.stub.Response.Body) > 0 {
		outputData := dynamic.NewMessage(r.methodDesc.GetOutputType())
		if err := outputData.UnmarshalJSON(r.stub.Response.Body); err != nil {
			log.Error(ctx, "cannot encode message", err)
			return err
		}

		if err := r.stream.SendMsg(outputData); err != nil {
			log.Error(ctx, "cannot send message", err)
			return err
		}
	}

	return convertGrpcStatus(ctx, r.descriptor, r.stub.Response).Err()
}

func captureIncomingRequest(ctx context.Context, fullMethod string) *rio.IncomingRequest {
	r := &rio.IncomingRequest{
		Method: rio.MethodGrpc,
		URL:    fullMethod,
		Header: types.Map{},
	}

	md, _ := metadata.FromIncomingContext(ctx)
	for k, v := range md {
		r.Header[k] = v
	}

	return r
}
