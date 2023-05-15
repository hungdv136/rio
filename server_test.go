package rio

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio/internal/netkit"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

func TestLocalServer_EndToEnd(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := NewLocalServerWithReporter(t)

	animalName := uuid.NewString()
	animalID := uuid.New().String()

	require.NoError(t, NewStub().For("POST", Contains("animal/create")).
		WithRequestBody(BodyJSONPath("name", EqualTo(animalName))).
		WillReturn(NewResponse().WithBody(MustToJSON(types.Map{"id": animalID}))).
		Send(ctx, server))

	requestURL := server.GetURL(ctx) + "/animal/create"
	body := types.Map{"name": animalName, "type": "bird"}
	res, err := netkit.PostJSON[types.Map](ctx, requestURL, body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.NotEmpty(t, res.Body)
	require.Equal(t, animalID, res.Body.ForceString("id"))

	incomeRequests, err := server.GetIncomingRequests(ctx, &IncomingQueryOption{})
	require.NoError(t, err)
	require.NotEmpty(t, incomeRequests)
}

func TestLocalServer_UploadFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := NewLocalServerWithReporter(t)

	fileData, err := os.ReadFile("server_test.go")
	require.NoError(t, err)

	fileID, err := server.UploadFile(ctx, uuid.NewString(), fileData)
	require.NoError(t, err)
	require.NotEmpty(t, fileID)

	path := "/animal/image/download"
	err = NewStub().For("GET", Contains(path)).WillReturn(NewResponse().WithFileBody("text/htm", fileID)).Send(ctx, server)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.GetURL(ctx)+path, nil)
	require.NoError(t, err)

	res, err := netkit.SendRequest(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	defer res.Body.Close()

	gotData, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, fileData, gotData)
}

func TestLocalServer_Deactivate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := NewLocalServerWithReporter(t)

	err := NewStub().For("GET", Contains("animal/create")).ShouldDeactivateWhenMatched().Send(ctx, server)
	require.NoError(t, err)

	requestURL := server.GetURL(ctx) + "/animal/create"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	require.NoError(t, err)

	res, err := netkit.SendRequest(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.NoError(t, res.Body.Close())

	// Not found since stub is deactivated
	res, err = netkit.SendRequest(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.NoError(t, res.Body.Close())
}

func TestLocalServer_ReserveProxy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expectedData := types.Map{"uuid": uuid.NewString()}
	targetServer := NewLocalServerWithReporter(t)
	targetURL := targetServer.GetURL(ctx) + "/proxy_server"
	require.NoError(t, NewStub().
		For("GET", Contains("proxy_server/animal/get")).
		WithQuery("id", EqualTo("1")).
		WithQuery("name", EqualTo("cat")).
		WillReturn(NewResponse().WithBody(MustToJSON(expectedData))).
		Send(ctx, targetServer))

	server := NewLocalServerWithReporter(t)
	require.NoError(t, NewStub().
		For("GET", Contains("animal/get")).
		WithTargetURL(targetURL).
		Send(ctx, server))

	requestURL := server.GetURL(ctx) + "/animal/get?id=1&name=cat"
	parsedRes, err := netkit.Get[types.Map](ctx, requestURL)
	require.NoError(t, err)
	require.Equal(t, expectedData, parsedRes.Body)

	incomeRequests, err := server.GetIncomingRequests(ctx, &IncomingQueryOption{})
	require.NoError(t, err)
	require.NotEmpty(t, incomeRequests)
}

func TestRemoteServer_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	expectedNamespace := uuid.NewString()
	mockingServer := NewLocalServerWithReporter(t)

	resStub := NewResponse().WithBody(MustToJSON(types.Map{"verdict": "success"}))
	require.NoError(t, NewStub().
		For("POST", Contains("stub/create_many")).
		WithRequestBody(BodyJSONPath("$.stubs[0].namespace", EqualTo(expectedNamespace))).
		WillReturn(resStub).Send(ctx, mockingServer))

	remoteServer := NewRemoteServer(mockingServer.GetURL(ctx)).WithNamespace(expectedNamespace)
	defer remoteServer.Close(ctx)

	err := remoteServer.Create(ctx, NewStub(), NewStub())
	require.NoError(t, err)
}

func TestRemoteServer_UploadFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockingServer := NewLocalServer()
	defer mockingServer.Close(ctx)

	resData := types.Map{
		"data":    types.Map{"file_id": uuid.New().String()},
		"verdict": "success",
	}

	resStub := NewResponse().WithBody(MustToJSON(resData))
	require.NoError(t, NewStub().For("POST", Contains("/stub/upload")).WillReturn(resStub).Send(ctx, mockingServer))

	remoteServer := NewRemoteServer(mockingServer.GetURL(ctx))
	defer remoteServer.Close(ctx)

	fileData, err := os.ReadFile("server_test.go")
	require.NoError(t, err)

	fileID, err := remoteServer.UploadFile(ctx, uuid.NewString(), fileData)
	require.NoError(t, err)
	require.Equal(t, resData.ForceMap("data")["file_id"], fileID)
}

func TestRemoteServer_ReplayOnShadowServer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockingServer := NewLocalServerWithReporter(t)

	incomeRequests := []*IncomingRequest{
		{
			ID:     1,
			URL:    "https://api.server.com/echo/animal/get",
			Method: "GET",
			Header: types.Map{},
		},
	}

	resData := types.Map{"verdict": "success", "data": types.Map{"requests": incomeRequests}}
	resStub := NewResponse().WithBody(MustToJSON(resData))
	err := NewStub().For("POST", Contains("/incoming_request/list")).WillReturn(resStub).Send(ctx, mockingServer)
	require.NoError(t, err)

	err = NewStub().For("GET", Contains("/animal/get")).WillReturn(NewResponse()).Send(ctx, mockingServer)
	require.NoError(t, err)

	remoteServer := NewRemoteServer(mockingServer.GetURL(ctx)).WithShadowServer(NewLocalServerWithReporter(t))
	err = remoteServer.ReplayOnShadowServer(ctx)
	require.NoError(t, err)
}
