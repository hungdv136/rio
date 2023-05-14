package rio

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
	"moul.io/http2curl/v2"
)

// RequestMatching defines request matching
type RequestMatching struct {
	// Rules to match the request method GET, POST, PUT, DELETE, PATCH
	Method string `json:"method,omitempty" yaml:"method"`

	// Rules to match the url
	URL []Operator `json:"url,omitempty" yaml:"url"`

	// Rules to match header name
	Header []FieldOperator `json:"header,omitempty" yaml:"header"`

	// Rules to match cookie
	Cookie []FieldOperator `json:"cookie,omitempty" yaml:"cookie"`

	// Rules to match request query
	Query []FieldOperator `json:"query,omitempty" yaml:"query"`

	// Rules to match request body by xml or json path
	Body []BodyOperator `json:"body,omitempty" yaml:"body"`
}

func (r *RequestMatching) Validate(ctx context.Context) error {
	if err := validateOp(ctx, r.URL...); err != nil {
		return err
	}

	if err := validateFieldOps(ctx, r.Header...); err != nil {
		return err
	}

	if err := validateFieldOps(ctx, r.Cookie...); err != nil {
		return err
	}

	if err := validateFieldOps(ctx, r.Query...); err != nil {
		return err
	}

	if err := validateBodyOps(ctx, r.Body...); err != nil {
		return err
	}

	return nil
}

// Scan implements sqlx JSON scan method
func (r *RequestMatching) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, &r)
	case string:
		return json.Unmarshal([]byte(v), &r)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Value implements sqlx JSON value method
func (r RequestMatching) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// GrpcRequest defines grpc request
type GrpcRequest struct {
	FullMethod string    `json:"full_method" yaml:"full_method"`
	InputData  types.Map `json:"input_data" yaml:"input_data"`
}

type IncomingRequests struct {
	Requests []*IncomingRequest `json:"requests" yaml:"requests"`
}

// IncomingRequest capture the incoming request
type IncomingRequest struct {
	ID        int64     `json:"id" yaml:"id"`
	Namespace string    `json:"namespace" yaml:"namespace"`
	Tag       string    `json:"tag" yaml:"tag"`
	URL       string    `json:"url" yaml:"url"`
	Method    string    `json:"method" yaml:"method"`
	Header    types.Map `json:"header" yaml:"header"`
	Body      []byte    `json:"body" yaml:"body"`
	CURL      string    `json:"curl" gorm:"column:curl" yaml:"curl"`
	StubID    int64     `json:"stub_id" yaml:"stub_id"`
}

// WithNamespace sets namespace
func (i *IncomingRequest) WithNamespace(v string) *IncomingRequest {
	i.Namespace = v
	return i
}

// Capture capture the request from http request
// Ignore body if the given request is multiparts or its body exceeds the threshold
func Capture(r *http.Request, bodyThreshold int) *IncomingRequest {
	incomingRequest := &IncomingRequest{
		Method: r.Method,
		URL:    r.URL.String(),
		Header: types.Map{},
	}

	for name := range r.Header {
		incomingRequest.Header[name] = r.Header[name]
	}

	shouldSaveBody := shouldSaveBody(r)
	if r.Body != nil && shouldSaveBody {
		bodyBuf := bytes.NewBuffer(make([]byte, 0))
		reader := io.TeeReader(r.Body, bodyBuf)
		r.Body = io.NopCloser(bodyBuf)
		body, err := io.ReadAll(reader)
		if err != nil {
			log.Error(r.Context(), err)
		} else if bodyThreshold == 0 || len(body) <= bodyThreshold {
			incomingRequest.Body = body
		} else {
			log.Info(r.Context(), "body is too large")
			shouldSaveBody = false
		}
	}

	curl, err := http2curl.GetCurlCommand(r)
	if err != nil {
		log.Error(r.Context(), "cannot parse curl", err)
		incomingRequest.CURL = err.Error()
	} else if !shouldSaveBody {
		curl = removeBodyFromCurl(curl)
		incomingRequest.CURL = curl.String()
		log.Info(r.Context(), "removed body from curl")
	} else {
		incomingRequest.CURL = curl.String()
	}

	return incomingRequest
}

// ReserveRequest converts the saved data to request
func (i *IncomingRequest) ReserveRequest(ctx context.Context) (*http.Request, error) {
	var requestBody io.Reader
	if len(i.Body) > 0 {
		requestBody = bytes.NewReader(i.Body)
	}

	req, err := http.NewRequestWithContext(ctx, i.Method, i.URL, requestBody)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	for key := range i.Header {
		if s, ok := i.Header.GetString(key); ok {
			req.Header.Add(key, s)
			continue
		}

		if a, ok := i.Header.GetArrayString(key); ok {
			for _, s := range a {
				req.Header.Add(key, s)
			}

			continue
		}

		err := fmt.Errorf("invalid header value %v", i.Header[key])
		log.Error(ctx, err)
		return nil, err
	}

	return req, nil
}

// Replay replays the http request to a server
// This is to debug with a stub from a remote server using IDE
func (i *IncomingRequest) Replay(ctx context.Context, server Server) (*http.Response, error) {
	req, err := i.ReserveRequest(ctx)
	if err != nil {
		return nil, err
	}

	requestURL := req.URL.String()
	if strings.HasPrefix(requestURL, "http") {
		requestURL = req.URL.Path
		if q := req.URL.Query(); len(q) > 0 {
			requestURL = requestURL + "?" + q.Encode()
		}
	}

	requestURL = "/" + strings.TrimPrefix(requestURL, "/")
	req.URL, err = url.Parse(server.GetURL(ctx) + requestURL)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return res, nil
}

func shouldSaveBody(r *http.Request) bool {
	contentType := r.Header.Get(HeaderContentType)
	return !strings.Contains(strings.ToLower(contentType), ContentTypeMultipart)
}

func removeBodyFromCurl(cmd *http2curl.CurlCommand) *http2curl.CurlCommand {
	slice := *cmd
	for i, item := range slice {
		if strings.EqualFold(item, "-d") {
			c := append(slice[:i], slice[i+2:]...)
			return &c
		}
	}

	return cmd
}
