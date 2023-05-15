package rio

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/hungdv136/rio/internal/log"
	fs "github.com/hungdv136/rio/internal/storage"
)

// Handler handles http equest
type Handler struct {
	fileStorage fs.FileStorage
	stubStore   StubStore
	namespace   string
	basePath    string

	// If the number of bytes of an incomming request's body is larger than this threshold
	// then body won't be saved to DB to avoid hurting DB performance when uploading with a file
	// If set to zero, then body is always saved to database
	bodyStoreThreshold int
}

// NewHandler handles request
func NewHandler(stubStore StubStore, fileStorage fs.FileStorage) *Handler {
	return &Handler{
		stubStore:          stubStore,
		fileStorage:        fileStorage,
		basePath:           "/echo/",
		bodyStoreThreshold: 1 << 20, // Default 1MB is a lot of text
	}
}

// WithNamespace sets namespace
func (h *Handler) WithNamespace(namespace string) *Handler {
	h.namespace = namespace
	return h
}

func (h *Handler) WithBodyStoreThreshold(v int) *Handler {
	h.bodyStoreThreshold = v
	return h
}

// Handle handles http request
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	incomeRequest := Capture(r, h.bodyStoreThreshold).WithNamespace(h.namespace)

	defer h.stubStore.CreateIncomingRequest(ctx, incomeRequest)

	stubs, err := h.stubStore.GetAll(ctx, h.namespace)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(stubs) == 0 {
		log.Info(ctx, "no stubs found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	matchedStubs := make([]*Stub, 0, len(stubs))
	for _, stub := range stubs {
		matched, err := stub.Match(ctx, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if matched {
			matchedStubs = append(matchedStubs, stub)
		}
	}

	if len(matchedStubs) == 0 {
		log.Info(ctx, "no matched stub found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	stub := SelectStubs(matchedStubs)
	incomeRequest.StubID = stub.ID
	incomeRequest.Tag = stub.Tag

	log.Info(ctx, "matched stub", stub.ID, stub.Description, "nb stubs", len(stubs), "in", h.namespace)

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

	if stub.IsReversed() {
		if err := h.reverse(w, r, stub); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	if err := h.processResponse(ctx, r, stub); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := stub.Response.WriteTo(ctx, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) processResponse(ctx context.Context, r *http.Request, stub *Stub) error {
	if len(stub.Response.BodyFile) > 0 {
		if err := stub.Response.LoadBodyFromFile(ctx, h.fileStorage); err != nil {
			return err
		}
	}

	if stub.HasTemplate() {
		if err := stub.Response.LoadBodyFromTemplate(ctx, &TemplateData{Request: r}); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) reverse(w http.ResponseWriter, r *http.Request, stub *Stub) error {
	target, err := url.Parse(stub.Proxy.TargetURL)
	if err != nil {
		log.Error(r.Context(), "cannot parse target url", stub.Proxy.TargetURL, err)
		return err
	}

	newReq := r.Clone(r.Context())
	newReq.Host = target.Host

	if len(stub.Proxy.TargetPath) > 0 {
		newReq.URL.Path = stub.Proxy.TargetPath
	} else {
		newReq.URL.Path = h.rewritePath(r.URL.Path)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Error(r.Context(), "cannot forward request to", newReq.URL.String(), err)
		rw.WriteHeader(http.StatusBadGateway)
	}

	if stub.Proxy.EnableRecord {
		proxy.ModifyResponse = h.proxyRecorder(stub) //nolint
		log.Info(r.Context(), "enabled response recording")
	} else {
		proxy.ModifyResponse = func(res *http.Response) error {
			log.Info(r.Context(), "forwarded with status code", res.StatusCode, res.Request.Method, res.Request.URL.String())
			return nil
		}
	}

	log.Info(r.Context(), "forward to", r.Method, stub.Proxy.TargetURL+newReq.URL.String())
	proxy.ServeHTTP(w, newReq)
	return nil
}

// The url path contains /echo as prefix of mock service
// This should be removed before forwarding to real service
func (h *Handler) rewritePath(urlPath string) string {
	if len(h.namespace) > 0 {
		return path.Join("/", strings.TrimPrefix(urlPath, "/"+h.namespace+h.basePath))
	}

	return path.Join("/", strings.TrimPrefix(urlPath, h.basePath))
}

func (h *Handler) proxyRecorder(stub *Stub) func(*http.Response) error {
	return func(res *http.Response) error {
		ctx := res.Request.Context()
		clonedStub := stub.Clone()
		clonedStub.ID = 0
		clonedStub.Description = fmt.Sprintf("Proxy record from stub id %d", stub.ID)
		clonedStub.Proxy = &Proxy{}
		clonedStub.Active = false
		clonedStub.Response = NewResponseFromHTTP(res)
		clonedStub.Tag = TagRecordedStub

		// TODO: Consider to support large body payload using file store
		if res.Body != nil {
			b := bytes.NewBuffer(make([]byte, 0))
			reader := io.TeeReader(res.Body, b)
			body, err := io.ReadAll(reader)
			if err != nil {
				log.Error(ctx, "cannot parse body", err)
				return err
			}

			if err := res.Body.Close(); err != nil {
				log.Error(ctx, "cannot close body", err)
			}

			res.Body = io.NopCloser(b)
			clonedStub.Response.Body = body
			log.Info(ctx, "parsed body", len(body))
		}

		if err := h.stubStore.Create(ctx, clonedStub); err != nil {
			return err
		}

		log.Info(ctx, "recording has been created in stub id", clonedStub.ID)
		return nil
	}
}
