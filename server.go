package rio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	fs "github.com/hungdv136/rio/internal/storage"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/netkit"
	"github.com/hungdv136/rio/internal/types"
	"github.com/hungdv136/rio/internal/util"
)

const (
	createStubsPath       = "/stub/create_many"
	uploadFilePath        = "/stub/upload"
	createListRequestPath = "/incoming_request/list"
)

var (
	_ Server = (*LocalServer)(nil)
	_ Server = (*RemoteServer)(nil)
)

// ReplayWithRequestID replay with a set of captured request id
func ReplayWithRequestID(ids ...int64) ReplayOptionFunc {
	return func(o *ReplayOption) {
		o.queryOption.Ids = append(o.queryOption.Ids, ids...)
	}
}

// ReplayWithLimit sets a limit
func ReplayWithLimit(limit int) ReplayOptionFunc {
	return func(o *ReplayOption) {
		o.queryOption.Limit = limit
	}
}

// ReplayOptionFunc is function to modify replay option
type ReplayOptionFunc func(*ReplayOption)

// ReplayOption defines replaying parameter
type ReplayOption struct {
	queryOption *IncomingQueryOption
}

// Server defines server interface
type Server interface {
	SetNamespace(v string)
	GetURL(ctx context.Context) string
	Create(ctx context.Context, stubs ...*Stub) error
	UploadFile(ctx context.Context, fileID string, file []byte) (string, error)
	Close(ctx context.Context)
}

// LocalServer is local server for unit test
type LocalServer struct {
	server      *httptest.Server
	stubStore   StubStore
	handler     *Handler
	fileStorage fs.FileStorage
	namespace   string
}

// NewLocalServer returns a new instance
func NewLocalServer() *LocalServer {
	stubStore := NewStubMemory()
	fileStorage := fs.NewLocalStorage(fs.LocalStorageConfig{UseTempDir: true, StoragePath: "uploaded_files"})
	handler := NewHandler(stubStore, fileStorage)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.Handle)

	return &LocalServer{
		stubStore:   stubStore,
		fileStorage: fileStorage,
		handler:     handler,
		server:      httptest.NewServer(mux),
	}
}

// NewLocalServerWithReporter inititial a server
// Automatically clean up data when test is completed
func NewLocalServerWithReporter(t *testing.T) *LocalServer {
	s := NewLocalServer()
	t.Cleanup(func() { s.Close(context.Background()) })

	return s
}

// WithNamespace sets namespace with chaining style
func (s *LocalServer) WithNamespace(namespace string) *LocalServer {
	s.SetNamespace(namespace)
	return s
}

// SetNamespace sets namespace which can be used for isolating test data for each testing
func (s *LocalServer) SetNamespace(v string) {
	s.namespace = v
	s.handler.namespace = v
}

// GetURL gets root url of server
func (s *LocalServer) GetURL(ctx context.Context) string {
	return s.server.URL
}

// Create creates stubs in local server
func (s *LocalServer) Create(ctx context.Context, stubs ...*Stub) error {
	for _, stub := range stubs {
		stub.WithNamespace(s.namespace)
	}

	return s.stubStore.Create(ctx, stubs...)
}

// UploadFile upload file to server
func (s *LocalServer) UploadFile(ctx context.Context, fileID string, file []byte) (string, error) {
	_, err := s.fileStorage.UploadFile(ctx, fileID, bytes.NewReader(file))
	if err != nil {
		return "", err
	}

	return fileID, nil
}

// GetIncomingRequests gets recorded incoming requests
func (s *LocalServer) GetIncomingRequests(ctx context.Context, option *IncomingQueryOption) ([]*IncomingRequest, error) {
	option.Namespace = s.namespace
	return s.stubStore.GetIncomingRequests(ctx, option)
}

// Close clean up
func (s *LocalServer) Close(ctx context.Context) {
	s.server.Close()
	_ = s.fileStorage.Reset(ctx)
}

// RemoteServer communicates with remote mock server
type RemoteServer struct {
	rootURL      string
	namespace    string
	shadowServer Server
}

// NewRemoteServer returns a new instance
func NewRemoteServer(rootURL string) *RemoteServer {
	return &RemoteServer{rootURL: rootURL}
}

// NewRemoteServerWithReporter inititial a server
// Automatically clean up data when test is completed
func NewRemoteServerWithReporter(t *testing.T, rootURL string) *RemoteServer {
	s := NewRemoteServer(rootURL)
	t.Cleanup(func() { s.Close(context.Background()) })

	return s
}

// WithNamespace sets namespace which can be used for isolating test data for each testing
func (s *RemoteServer) WithNamespace(namespace string) *RemoteServer {
	s.SetNamespace(namespace)
	return s
}

// WithShadowServer sets shadow server
func (s *RemoteServer) WithShadowServer(server Server) *RemoteServer {
	s.shadowServer = server
	s.shadowServer.SetNamespace(s.namespace)
	return s
}

// SetNamespace sets namespace which can be used for isolating test data for each testing
func (s *RemoteServer) SetNamespace(v string) {
	s.namespace = v

	if s.shadowServer != nil {
		s.shadowServer.SetNamespace(v)
	}
}

// GetURL gets root url of server
func (s *RemoteServer) GetURL(ctx context.Context) string {
	return s.rootURL
}

// Create creates stubs in remote server
func (s *RemoteServer) Create(ctx context.Context, stubs ...*Stub) error {
	for _, stub := range stubs {
		stub.WithNamespace(s.namespace)
	}

	data := types.Map{"stubs": stubs, "client": "go_sdk", "return_encoded": true}
	parsedResp, err := netkit.PostJSON[netkit.InternalBody[ArrayStubs]](ctx, s.rootURL+createStubsPath, data)
	if err != nil {
		return err
	}

	if parsedResp.StatusCode != http.StatusOK {
		err := errors.New("cannot create stubs")
		log.Error(ctx, err)
		return err
	}

	if s.shadowServer != nil {
		return s.shadowServer.Create(ctx, parsedResp.Body.Data.Stubs...)
	}

	return nil
}

// UploadFile upload file to server
func (s *RemoteServer) UploadFile(ctx context.Context, fileID string, fileBody []byte) (string, error) {
	request, err := netkit.NewUploadRequest(ctx, s.rootURL+uploadFilePath, fileBody, map[string]string{"file_id": fileID})
	if err != nil {
		return "", err
	}

	res, err := netkit.SendRequest(request)
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}
	defer util.CloseSilently(ctx, res.Body.Close)

	if res == nil || res.StatusCode != http.StatusOK {
		err := fmt.Errorf("cannot upload file %v", res)
		log.Error(ctx, err)
		return "", err
	}

	parsedResp, err := netkit.ParseResponse[netkit.InternalBody[types.Map]](ctx, res)
	if err != nil {
		log.Error(ctx, err)
		return "", err
	}

	if s.shadowServer != nil {
		if _, err := s.shadowServer.UploadFile(ctx, fileID, fileBody); err != nil {
			return "", err
		}
	}

	fileID, _ = parsedResp.Body.Data.GetString("file_id")
	return fileID, nil
}

// GetIncomingRequests gets recorded incoming requests
func (s *RemoteServer) GetIncomingRequests(ctx context.Context, option *IncomingQueryOption) ([]*IncomingRequest, error) {
	option.Namespace = s.namespace
	res, err := netkit.PostJSON[netkit.InternalBody[IncomingRequests]](ctx, s.rootURL+createListRequestPath, option)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		err := errors.New("cannot get incoming requests")
		log.Error(ctx, err)
		return nil, err
	}

	return res.Body.Data.Requests, nil
}

// ReplayOnShadowServer replays incoming requests (from remote server) to a shadow server (local server)
// By default, only the last request will be replayed. Use option to change replay option
// This is to debug the stub on a remote server using IDE
func (s *RemoteServer) ReplayOnShadowServer(ctx context.Context, options ...ReplayOptionFunc) error {
	if s.shadowServer == nil {
		err := errors.New("shawdow server has not set")
		log.Error(ctx, err)
		return err
	}

	relayOpt := &ReplayOption{
		queryOption: &IncomingQueryOption{
			Namespace: s.namespace,
			Limit:     1,
		},
	}

	for _, optionFn := range options {
		optionFn(relayOpt)
	}

	requests, err := s.GetIncomingRequests(ctx, relayOpt.queryOption)
	if err != nil {
		return err
	}

	log.Info(ctx, s.namespace, "nb of requests", len(requests))

	for i := len(requests) - 1; i >= 0; i-- {
		res, err := requests[i].Replay(ctx, s.shadowServer)
		if err != nil {
			return err
		}

		log.Info(ctx, "replayed", requests[i].ID, requests[i].URL, res.StatusCode)
		res.Body.Close()
	}

	return nil
}

// Close clean up server data
func (s *RemoteServer) Close(ctx context.Context) {}
