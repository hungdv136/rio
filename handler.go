package rio

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/hungdv136/rio/internal/log"
	fs "github.com/hungdv136/rio/internal/storage"
)

// Handler handles mocking for http request
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
		matched, err := matchHTTPRequest(ctx, stub, r)
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

// matchHTTPRequest matches a stub with incoming http request
func matchHTTPRequest(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	if s.Request == nil {
		return false, nil
	}

	if len(s.Request.Method) > 0 && !strings.EqualFold(s.Request.Method, r.Method) {
		return false, nil
	}

	if matched, err := matchURL(ctx, s, r); err != nil || !matched {
		return false, err
	}

	if matched, err := matchHeader(ctx, s, r); err != nil || !matched {
		return false, err
	}

	if matched, err := matchCookies(ctx, s, r); err != nil || !matched {
		return false, err
	}

	if matched, err := matchQuery(ctx, s, r); err != nil || !matched {
		return false, err
	}

	if matched, err := matchBody(ctx, s, r); err != nil || !matched {
		return false, err
	}

	return true, nil
}

func matchURL(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	for _, op := range s.Request.URL {
		if matched, err := Match(ctx, op, r.URL.String()); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchHeader(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	for _, op := range s.Request.Header {
		if matched, err := Match(ctx, op.Operator, r.Header.Get(op.FieldName)); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchQuery(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	query := r.URL.Query()
	for _, op := range s.Request.Query {
		if matched, err := Match(ctx, op.Operator, query.Get(op.FieldName)); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchCookies(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	for _, op := range s.Request.Cookie {
		cookie, err := r.Cookie(op.FieldName)
		if err != nil {
			if !errors.Is(err, http.ErrNoCookie) {
				return false, err
			}

			// If cookie not found, then lets the operator decides the output
			if matched, err := Match(ctx, op.Operator, ""); err != nil || !matched {
				return false, err
			}

			continue
		}

		if matched, err := Match(ctx, op.Operator, cookie.Value); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchBody(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	if len(s.Request.Body) == 0 {
		return true, nil
	}

	if r.Body == nil {
		err := errors.New("missing body")
		log.Error(ctx, err)
		return false, err
	}

	contentType := r.Header.Get(HeaderContentType)
	if err := validateBodyOperator(ctx, s, contentType); err != nil {
		return false, err
	}

	if strings.HasPrefix(contentType, ContentTypeJSON) {
		return matchJSONBody(ctx, s, r)
	}

	if strings.HasPrefix(contentType, ContentTypeMultipart) {
		return matchMultiplePart(ctx, s, r)
	}

	if strings.HasPrefix(contentType, ContentTypeForm) {
		return matchURLEncodedBody(ctx, s, r)
	}

	err := fmt.Errorf("unsupported content type %s", contentType)
	log.Error(ctx, err)
	return false, err
}

func matchJSONBody(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	dataMap := map[string]interface{}{}
	decoder := json.NewDecoder(readRequestBody(r))
	decoder.UseNumber()
	if err := decoder.Decode(&dataMap); err != nil && !errors.Is(err, io.EOF) {
		log.Error(ctx, "cannot decode json", err)
		return false, err
	}

	for _, op := range s.Request.Body {
		val, err := jsonpath.Get(op.KeyPath, dataMap)
		if err != nil {
			if !strings.Contains(err.Error(), "unknown key") {
				log.Error(ctx, "error when executing json path", err)
				return false, err
			}
		}

		if matched, err := Match(ctx, op.Operator, val); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchMultiplePart(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	if err := r.ParseMultipartForm(1024 * 1024 * 20 << 20); err != nil {
		log.Error(ctx, err)
		return false, err
	}

	for _, op := range s.Request.Body {
		val := r.FormValue(op.KeyPath)
		if matched, err := Match(ctx, op.Operator, val); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

func matchURLEncodedBody(ctx context.Context, s *Stub, r *http.Request) (bool, error) {
	for _, op := range s.Request.Body {
		val := r.FormValue(op.KeyPath)
		if matched, err := Match(ctx, op.Operator, val); err != nil || !matched {
			return false, err
		}
	}

	return true, nil
}

// Each body operator is applied for a specific content type
// This is to validate whether request content type is matched with content type of all operators
func validateBodyOperator(ctx context.Context, s *Stub, contentType string) error {
	for _, op := range s.Request.Body {
		if !strings.HasPrefix(contentType, op.ContentType) {
			err := fmt.Errorf("mismatch request and operator content type %s - %s", contentType, op.ContentType)
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}
