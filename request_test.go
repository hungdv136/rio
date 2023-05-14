package rio

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
	"moul.io/http2curl/v2"
)

func TestIncomingRequest_Replay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := NewLocalServerWithReporter(t)

	t.Run("replay_get", func(t *testing.T) {
		t.Parallel()

		requestID := uuid.NewString()
		stubRes := NewResponse().WithBody(MustToJSON(types.Map{"id": 1}))
		err := NewStub().For("GET", Contains("animal/get")).
			WithHeader("X-REQUEST-ID", EqualTo(requestID)).
			WithQuery("id", EqualTo(123)).
			WillReturn(stubRes).
			Send(ctx, server)
		require.NoError(t, err)

		incomeRequest := &IncomingRequest{
			ID:        1,
			Namespace: "dop",
			URL:       "https://api.server.com/echo/animal/get?id=123",
			Method:    "GET",
			Header: types.Map{
				"X-REQUEST-ID": requestID,
			},
		}

		res, err := incomeRequest.Replay(ctx, server)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		res.Body.Close()
	})

	t.Run("replay_post", func(t *testing.T) {
		t.Parallel()

		requestBody := types.Map{"id": 345}
		requestID := uuid.NewString()
		stubRes := NewResponse().WithBody(MustToJSON(types.Map{"id": 1}))
		err := NewStub().For("POST", Contains("animal/create")).
			WithHeader("X-REQUEST-ID", EqualTo(requestID)).
			WithRequestBody(BodyJSONPath("id", EqualTo(345))).
			WillReturn(stubRes).
			Send(ctx, server)
		require.NoError(t, err)

		incomeRequest := &IncomingRequest{
			ID:        1,
			Namespace: "dop",
			URL:       "https://api.server.com/echo/animal/create",
			Method:    "POST",
			Header: types.Map{
				HeaderContentType: ContentTypeJSON,
				"X-REQUEST-ID":    requestID,
			},
			Body: []byte(requestBody.ForceJSON()),
		}

		res, err := incomeRequest.Replay(ctx, server)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)
		res.Body.Close()
	})
}

func TestRemoveBodyFromCurl(t *testing.T) {
	t.Parallel()

	curl := http2curl.CurlCommand([]string{"-H", "H1=val1", "-d", "data", "http://rio.com"})
	expectedCurl := http2curl.CurlCommand([]string{"-H", "H1=val1", "http://rio.com"})
	got := removeBodyFromCurl(&curl)
	require.Equal(t, &expectedCurl, got)
}
