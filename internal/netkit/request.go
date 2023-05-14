package netkit

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hungdv136/rio/internal/log"
)

var defaultClient = &http.Client{
	Timeout: 32 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		IdleConnTimeout:     8 * time.Second,
		MaxIdleConnsPerHost: 10,
	},
}

// SendRequest sends general request to a URL and returns HTTP response
func SendRequest(r *http.Request) (*http.Response, error) {
	if id := log.GetID(r.Context()); len(id) > 0 {
		r.Header.Set("X-REQUEST-ID", id)
	}

	response, err := defaultClient.Do(r)
	if err != nil {
		log.Error(r.Context(), err)
		return nil, err
	}

	return response, nil
}

// NewUploadRequest create a new http upload request
func NewUploadRequest(ctx context.Context, url string, file []byte, fields map[string]string) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", uuid.NewString())
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if _, err := part.Write(file); err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			log.Error(ctx, err)
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req.WithContext(ctx), nil
}

// PostJSON executes request with POST method and JSON as body, then parse response
// Body is structure of response body
func PostJSON[Body any](ctx context.Context, url string, body interface{}) (*Response[Body], error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Error(ctx, "cannot marshal body", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Error(ctx, "cannot create request", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := SendRequest(req)
	if err != nil {
		return nil, err
	}

	return ParseResponse[Body](ctx, res)
}

// Get executes request with GET method
// Body is structure of response body
func Get[Body any](ctx context.Context, url string) (*Response[Body], error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error(ctx, "cannot create request", err)
		return nil, err
	}

	res, err := SendRequest(req)
	if err != nil {
		return nil, err
	}

	return ParseResponse[Body](ctx, res)
}
