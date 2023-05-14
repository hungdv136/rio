package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/database"
	fs "github.com/hungdv136/rio/internal/storage"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServerEndToEnd(t *testing.T) {
	t.Parallel()

	ctx := log.SaveID(context.Background(), t.Name())
	storageCfg := fs.LocalStorageConfig{StoragePath: "../testdata"}
	storage := fs.NewLocalStorage(storageCfg)
	stubStore, err := database.NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	sd := NewServiceDescriptor(storage)
	sd.cachedDir = uuid.NewString()
	cleanup(t, sd)

	server := NewServer(stubStore, storage, sd)
	require.NoError(t, server.StartAsync(ctx, ""))
	serverAddr := server.listener.Addr().String()

	// Arrange the proto configuration and stub
	fullMethod := "/offers.v1.OfferService/ValidateOffer"
	proto := &rio.Proto{
		Name:    "offer",
		FileID:  "offer_proto",
		Methods: []string{fullMethod},
	}
	require.NoError(t, stubStore.CreateProto(ctx, proto))

	descriptor, err := sd.GetDescriptor(ctx, proto.FileID)
	require.NoError(t, err)

	m, err := descriptor.GetMethod(ctx, fullMethod)
	require.NoError(t, err)

	t.Run("matched", func(t *testing.T) {
		t.Parallel()

		requestID := uuid.NewString()
		outputMap := types.Map{"id": uuid.NewString(), "request_id": uuid.NewString()}
		require.NoError(t, stubStore.Create(ctx, rio.NewStub().
			ForGRPC(rio.EqualTo(fullMethod)).
			WithRequestBody(rio.BodyJSONPath("$.request_id", rio.EqualTo(requestID))).
			WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(outputMap)))))

		input, err := mapToMessage(ctx, types.Map{"request_id": requestID}, m.GetInputType())
		require.NoError(t, err)

		actualOutput, err := invokeGrpc(ctx, serverAddr, m, input)
		require.NoError(t, err)

		actualOutputMap, err := messageToMap(ctx, actualOutput)
		require.NoError(t, err)
		require.Equal(t, actualOutputMap, outputMap)
	})

	t.Run("not_matched_input_param", func(t *testing.T) {
		t.Parallel()

		input, err := mapToMessage(ctx, types.Map{"request_id": uuid.NewString()}, m.GetInputType())
		require.NoError(t, err)

		actualOutputMap, err := invokeGrpc(ctx, serverAddr, m, input)
		require.Error(t, err)
		require.Nil(t, actualOutputMap)

		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("reverse_proxy", func(t *testing.T) {
		t.Parallel()

		// Expected that server will make another request to the target server
		proxyRequestID := uuid.NewString()
		proxyStub := rio.NewStub().
			ForGRPC(rio.EqualTo(fullMethod)).
			WithRequestBody(rio.BodyJSONPath("$.request_id", rio.EqualTo(proxyRequestID))).
			WithHeader("X-PROXY", rio.Empty()).
			WithTargetURL(serverAddr).
			WithEnableRecord(true).
			WillReturn(rio.NewResponse())

		// Stub for a proxy request
		xRequestID := uuid.NewString()
		proxyOutputMap := types.Map{"id": uuid.NewString(), "request_id": uuid.NewString()}
		proxyResponseStub := rio.NewStub().
			ForGRPC(rio.EqualTo(fullMethod)).
			WithRequestBody(rio.BodyJSONPath("$.request_id", rio.EqualTo(proxyRequestID))).
			WithHeader("X-PROXY", rio.EqualTo("rio")).
			WithHeader("X-REQUEST-ID", rio.EqualTo(xRequestID)).
			WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(proxyOutputMap)))

		require.NoError(t, stubStore.Create(ctx, proxyStub, proxyResponseStub))

		// Assure metadata will be forwarded to remote server
		ctx := metadata.AppendToOutgoingContext(ctx, "X-REQUEST-ID", xRequestID)
		input, err := mapToMessage(ctx, types.Map{"request_id": proxyRequestID}, m.GetInputType())
		require.NoError(t, err)

		actualOutput, err := invokeGrpc(ctx, serverAddr, m, input)
		require.NoError(t, err)

		actualOutputMap, err := messageToMap(ctx, actualOutput)
		require.NoError(t, err)
		require.Equal(t, actualOutputMap, proxyOutputMap)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		requestID := uuid.NewString()
		errDetails := &rio.ErrorDetail{
			Type: "common.v1.CommonError",
			Value: types.Map{
				"verdict": "success",
			},
		}

		require.NoError(t, stubStore.Create(ctx, rio.NewStub().
			ForGRPC(rio.EqualTo(fullMethod)).
			WithRequestBody(rio.BodyJSONPath("$.request_id", rio.EqualTo(requestID))).
			WillReturn(rio.NewResponse().WithStatusCode(int(codes.Internal)).WithError("error", errDetails))))

		input, err := mapToMessage(ctx, types.Map{"request_id": requestID}, m.GetInputType())
		require.NoError(t, err)

		actualOutputMap, grpcErr := invokeGrpc(ctx, serverAddr, m, input)
		require.Error(t, grpcErr)
		require.Nil(t, actualOutputMap)

		s, ok := status.FromError(grpcErr)
		require.True(t, ok)
		require.Equal(t, codes.Internal, s.Code())

		resErr := convertGrpcError(ctx, descriptor, grpcErr)
		require.NotNil(t, resErr)
		require.NotEmpty(t, resErr.Details)
		require.Equal(t, "success", resErr.Details[0].Value.ForceString("verdict"))
	})
}
