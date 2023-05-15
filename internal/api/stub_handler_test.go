package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/netkit"
	"github.com/hungdv136/rio/internal/test/mock"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

func TestCreateStub(t *testing.T) {
	t.Parallel()

	// Assert that can submit JSON from Go client to API
	stub := rio.NewStub().
		For("GET", rio.Contains("animal/create")).
		WithDescription("this is description").
		WithHeader("X-REQUEST-ID", rio.EqualTo(uuid.NewString())).
		WithQuery("search_term", rio.EqualTo(uuid.NewString())).
		WithCookie("SESSION_ID", rio.EqualTo(uuid.NewString())).
		WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(types.Map{"data": uuid.NewString()})))
	goClientParams := types.Map{"stubs": []*rio.Stub{stub}}

	// Assert that can submit with raw JSON without Go client
	rawJSONParams := parseJSONFileToMap(t, "../../testdata/stubs.json")

	// Assert that can submit raw html text from Go client
	html, err := os.ReadFile("../../testdata/html.html")
	require.NoError(t, err)

	htmlStub := rio.NewStub().
		For("GET", rio.Contains("animal/create_html")).
		WillReturn(rio.NewResponse().WithBody(rio.ContentTypeHTML, html))
	htmlParams := types.Map{"stubs": []*rio.Stub{htmlStub}}

	testCases := []*netkit.TestCase{
		netkit.NewTestCase("missing_required_params", http.MethodPost, "/stub/create_many", types.Map{}, http.StatusBadRequest, VerdictMissingParameters),
		netkit.NewTestCase("success", http.MethodPost, "/stub/create_many", goClientParams, http.StatusOK, VerdictSuccess),
		netkit.NewTestCase("success_from_raw_json", http.MethodPost, "/stub/create_many", rawJSONParams, http.StatusOK, VerdictSuccess),
		netkit.NewTestCase("success_body_as_html", http.MethodPost, "/stub/create_many", htmlParams, http.StatusOK, VerdictSuccess),
	}

	app, err := NewApp(context.Background(), config.NewConfig())
	require.NoError(t, err)

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			res := netkit.ExecuteTestCase[rio.ArrayStubs](t, tc, app.kit)
			if tc.ExpectStatus == http.StatusOK {
				for _, gotStub := range res.Body.Data.Stubs {
					if tc.Name == "success" {
						require.Equal(t, stub.Description, gotStub.Description)
					} else if tc.Name == "success_from_raw_json" {
						body := types.Map{} // Verify that body is raw json
						require.NoError(t, json.Unmarshal(gotStub.Response.Body, &body))
						require.NotEmpty(t, body)
					} else if tc.Name == "success_body_as_html" {
						require.Equal(t, string(html), string(gotStub.Response.Body))
					}
				}
			}
		})
	}
}

func TestCreateStubWithYaml(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fileData, err := os.ReadFile("../../testdata/stubs.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, fileData)

	app, err := NewApp(ctx, config.NewConfig())
	require.NoError(t, err)

	w := netkit.NewResponseRecorder()
	req, err := http.NewRequestWithContext(ctx, "POST", "/stub/create_many", bytes.NewReader(fileData))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/x-yaml")
	app.kit.ServeHTTP(w, req)

	result := w.Result()
	require.Equal(t, 200, result.StatusCode)

	res, err := netkit.ParseResponse[netkit.InternalBody[rio.ArrayStubs]](ctx, result)
	require.NoError(t, err)
	require.Equal(t, VerdictSuccess, res.Body.Verdict)
}

func TestGetStubs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.NewConfig()

	app, err := NewApp(ctx, cfg)
	require.NoError(t, err)

	namespace := uuid.NewString()
	stub := rio.NewStub().
		WithNamespace(namespace).
		For("GET", rio.Contains("animal/create")).
		WithHeader("X-REQUEST-ID", rio.EqualTo(uuid.NewString())).
		WithQuery("search_term", rio.EqualTo(uuid.NewString())).
		WithCookie("SESSION_ID", rio.EqualTo(uuid.NewString())).
		WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(types.Map{"data": uuid.NewString()})))

	err = app.stubStore.Create(ctx, stub)
	require.NoError(t, err)

	validParams := types.Map{"namespace": namespace}
	testCases := []*netkit.TestCase{
		netkit.NewTestCase("success", http.MethodGet, "/stub/list", validParams, http.StatusOK, VerdictSuccess),
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			res := netkit.ExecuteTestCase[rio.ArrayStubs](t, tc, app.kit)
			if tc.ExpectStatus == http.StatusOK {
				require.Len(t, res.Body.Data.Stubs, 1)
				require.Equal(t, res.Body.Data.Stubs[0].ID, stub.ID)
			}
		})
	}
}

func TestUploadFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	app, err := NewApp(ctx, config.NewConfig(), func(app *App) {
		fileStorage := mock.NewMockFileStorage(ctrl)
		fileStorage.EXPECT().UploadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).Times(1)
		app.fileStorage = fileStorage
	})
	require.NoError(t, err)

	w := netkit.NewResponseRecorder()
	fileID := uuid.NewString()

	metadata := map[string]string{"file_id": fileID}
	fileContents, err := os.ReadFile("stub_handler_test.go")
	require.NoError(t, err)

	req, err := netkit.NewUploadRequest(ctx, "/stub/upload", fileContents, metadata)
	require.NoError(t, err)

	app.kit.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	res, err := netkit.ParseResponse[netkit.InternalBody[types.Map]](ctx, w.Result())
	require.NoError(t, err)

	returnedFileID := res.Body.Data.ForceString("file_id")
	require.NotEmpty(t, returnedFileID)
	require.Equal(t, fileID, returnedFileID)
}

func TestUploadProtos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	defer os.RemoveAll("cached_grpc_protos")

	metadata := map[string]string{"name": "offer proto"}
	fileContents, err := os.ReadFile("../../testdata/offer_proto")
	require.NoError(t, err)
	reader := io.NopCloser(bytes.NewReader(fileContents))

	app, err := NewApp(ctx, config.NewConfig(), func(app *App) {
		fileStorage := mock.NewMockFileStorage(ctrl)
		fileStorage.EXPECT().UploadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).Times(1)
		fileStorage.EXPECT().DownloadFile(gomock.Any(), gomock.Any()).Return(reader, nil).Times(1)
		app.fileStorage = fileStorage
	})
	require.NoError(t, err)

	req, err := netkit.NewUploadRequest(ctx, "/proto/upload", fileContents, metadata)
	require.NoError(t, err)

	w := netkit.NewResponseRecorder()
	app.kit.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	type resData struct {
		Proto *rio.Proto `json:"proto"`
	}

	res, err := netkit.ParseResponse[netkit.InternalBody[resData]](ctx, w.Result())
	require.NoError(t, err)

	createdProtos, err := app.stubStore.GetProtos(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, createdProtos)

	proto := findProtoByID(createdProtos, res.Body.Data.Proto.ID)
	require.NotNil(t, proto)
	require.Equal(t, []string{"/offers.v1.OfferService/ValidateOffer"}, proto.Methods)
}

func TestGetIncomingRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app, err := NewApp(ctx, config.NewConfig())
	require.NoError(t, err)

	requestBody, err := os.ReadFile("stub_handler_test.go")
	require.NoError(t, err)

	request := &rio.IncomingRequest{
		URL:    uuid.NewString(),
		Method: "GET",
		Header: types.Map{
			"key": uuid.NewString(),
		},
		CURL:   uuid.NewString(),
		Body:   requestBody,
		StubID: 1,
	}

	err = app.stubStore.CreateIncomingRequest(ctx, request)
	require.NoError(t, err)

	validParams := types.Map{"ids": []int64{request.ID}}
	testCases := []*netkit.TestCase{
		netkit.NewTestCase("success", http.MethodPost, "/incoming_request/list", validParams, http.StatusOK, VerdictSuccess),
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			res := netkit.ExecuteTestCase[rio.IncomingRequests](t, tc, app.kit)
			require.NoError(t, err)
			require.Len(t, res.Body.Data.Requests, 1)
			require.Equal(t, request.ID, res.Body.Data.Requests[0].ID)
			require.Equal(t, requestBody, res.Body.Data.Requests[0].Body)
		})
	}
}

func TestEchoHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app, err := NewApp(ctx, config.NewConfig())
	require.NoError(t, err)

	expectedData := types.Map{"data": uuid.NewString(), "verdict": VerdictSuccess}
	namespace := uuid.NewString()
	stub := rio.NewStub().ForAny(rio.Contains("admin/animal")).WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(expectedData)))
	stubWithNS := rio.NewStub().ForAny(rio.Contains("phone_owner/verify")).WithNamespace(namespace).WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(expectedData)))
	require.NoError(t, app.stubStore.Create(ctx, stub, stubWithNS))

	validParams := types.Map{"name": uuid.NewString()}
	testCases := []*netkit.TestCase{
		netkit.NewTestCase("success_post", http.MethodPost, "/echo/admin/animal/create", validParams, http.StatusOK, VerdictSuccess),
		netkit.NewTestCase("success_get", http.MethodGet, "/echo/admin/animal/get", types.Map{}, http.StatusOK, VerdictSuccess),
		netkit.NewTestCase("success_namespace", http.MethodGet, "/"+namespace+"/echo/phone_owner/verify", types.Map{}, http.StatusOK, VerdictSuccess),
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			res := netkit.ExecuteTestCase[string](t, tc, app.kit)
			require.Equal(t, expectedData["data"], res.Body.Data)
			require.Equal(t, expectedData["verdict"], res.Body.Verdict)
		})
	}
}

func TestEchoHandler_Reverse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app, err := NewApp(ctx, config.NewConfig())
	require.NoError(t, err)

	expectedData := types.Map{
		"data":    uuid.NewString(),
		"verdict": VerdictSuccess,
	}

	targetServer := rio.NewLocalServerWithReporter(t)
	targetURL := targetServer.GetURL(ctx) + "/proxy_server"
	require.NoError(t, rio.NewStub().
		For("POST", rio.Contains("proxy_server/reverse/animal/create")).
		WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(expectedData))).
		Send(ctx, targetServer))

	require.NoError(t, app.stubStore.Create(ctx, rio.NewStub().
		ForAny(rio.Contains("reverse/animal/create")).
		WithTargetURL(targetURL)))

	validParams := types.Map{"name": uuid.NewString()}
	testCases := []*netkit.TestCase{
		netkit.NewTestCase("success", http.MethodPost, "/echo/reverse/animal/create", validParams, http.StatusOK, VerdictSuccess),
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			res := netkit.ExecuteTestCase[string](t, tc, app.kit)
			require.Equal(t, expectedData["data"], res.Body.Data)
			require.Equal(t, expectedData["verdict"], res.Body.Verdict)
		})
	}
}

func TestEchoHandler_Reverse_Recording(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app, err := NewApp(ctx, config.NewConfig())
	require.NoError(t, err)

	expectedData := types.Map{"data": uuid.NewString(), "verdict": VerdictSuccess}
	targetServer := rio.NewLocalServerWithReporter(t)
	targetURL := targetServer.GetURL(ctx) + "/proxy_server"
	require.NoError(t, rio.NewStub().
		For("POST", rio.Contains("proxy_server/reverse_recording/animal/create")).
		WillReturn(rio.NewResponse().WithBody(rio.MustToJSON(expectedData))).
		Send(ctx, targetServer))

	proxyStubs := rio.NewStub().
		ForAny(rio.Contains("reverse_recording/animal/create")).
		WithTargetURL(targetURL).
		WithEnableRecord(true)
	require.NoError(t, app.stubStore.Create(ctx, proxyStubs))

	validParams := types.Map{"name": uuid.NewString()}
	testCases := []*netkit.TestCase{
		netkit.NewTestCase("success", http.MethodPost, "/echo/reverse_recording/animal/create", validParams, http.StatusOK, VerdictSuccess),
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			res := netkit.ExecuteTestCase[string](t, tc, app.kit)
			require.Equal(t, expectedData["data"], res.Body.Data)
			require.Equal(t, expectedData["verdict"], res.Body.Verdict)
		})
	}
}

func findProtoByID(protos []*rio.Proto, id int64) *rio.Proto {
	for _, p := range protos {
		if p.ID == id {
			return p
		}
	}

	return nil
}
