package rio

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hungdv136/rio/internal/log"
	fs "github.com/hungdv136/rio/internal/storage"
	"github.com/hungdv136/rio/internal/types"
)

var _ json.Unmarshaler = (*Body)(nil)

// Cookie defines cookie
type Cookie struct {
	Name      string    `json:"name" yaml:"name"`
	Value     string    `json:"value" yaml:"value"`
	ExpiredAt time.Time `json:"expired_at" yaml:"expired_at"`
}

// Body is a custom encoded body value to support submit body with base64 encoded or raw string
type Body []byte

// Support submit body with base64 encoded or raw string
func (m *Body) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("rio.Body: UnmarshalJSON on nil pointer")
	}

	// Client submits base64 encoded string to API
	if m1, ok := tryParseBase64String(data); ok {
		*m = append((*m)[0:0], m1...)
		return nil
	}

	// Encoded data from database
	if m1, ok := tryParseBase64(data); ok {
		*m = append((*m)[0:0], m1...)
		return nil
	}

	// Can submit raw data (JSON object or string) through API
	// It is convenient for team to submit JSON as raw json instead of wrap in a string
	*m = append((*m)[0:0], data...)
	return nil
}

// It is more convenient to submit data with raw string in YAML
func (m *Body) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var body string
	if err := unmarshal(&body); err != nil {
		return err
	}

	*m = append((*m)[0:0], []byte(body)...)
	return nil
}

func tryParseBase64String(data []byte) ([]byte, bool) {
	if len(data) <= 2 {
		return nil, false
	}

	n := len(data)
	if data[0] != '"' || data[n-1] != '"' {
		return nil, false
	}

	return tryParseBase64(data[1 : n-1])
}

func tryParseBase64(data []byte) ([]byte, bool) {
	dbuf := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	if n, err := base64.StdEncoding.Decode(dbuf, data); err == nil && n > 0 {
		return trimRightByte(dbuf, '\x00'), true
	}

	return nil, false
}

func trimRightByte(s []byte, c byte) []byte {
	for len(s) > 0 && s[len(s)-1] == c {
		s = s[:len(s)-1]
	}
	return s
}

func isJSONObject(data []byte) bool {
	if len(data) == 0 || !json.Valid(data) {
		return false
	}

	firstChar := string(data[0])
	return firstChar == "{" || firstChar == "["
}

// ResponseError defines the response error. Only applicable for grpc
// This is equivalent to status.Status in GRPC
type ResponseError struct {
	Message string         `json:"message,omitempty" yaml:"message"`
	Details []*ErrorDetail `json:"details,omitempty" yaml:"details"`
}

// Clone clones the response error and its properties
func (r *ResponseError) Clone() *ResponseError {
	if r == nil {
		return nil
	}

	nr := &ResponseError{
		Message: r.Message,
	}

	if r.Details != nil {
		nr.Details = make([]*ErrorDetail, len(r.Details))
		copy(nr.Details, r.Details)
	}

	return nr
}

// ErrorDetail is to define details for error
type ErrorDetail struct {
	// Type defines message type of error detail. For example: common.v1.CommonError
	// This is to get message descriptor to encode/decode message
	// The proto of defined type must be included in proto compressed file
	Type string `json:"type,omitempty" yaml:"type"`

	// Value holds the payload of the error
	Value types.Map `json:"value,omitempty" yaml:"value"`
}

// Response defines a response
type Response struct {
	// Required. Define the response status code
	// GRPC. Default 0 OK: https://grpc.github.io/grpc/core/md_doc_statuscodes.html
	// HTTP. Default 200 OK: https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
	StatusCode int `json:"status_code,omitempty" yaml:"status_code"`

	// Body defines response body
	// Client can submit raw JSON object or base64 encoded string for HTML, XML, ... via API
	// If client submits stubs using YAML format via API, then raw string can be used for any text based content
	Body Body `json:"body,omitempty" yaml:"body"`

	// This is the id of uploaded file that can be used to simulate the download
	// Or can be used to respond a large data payload which is not suitable to save in database
	BodyFile string `json:"body_file,omitempty" yaml:"body_file"`

	// Optional. Define response cookies
	// This is not applied for GRPC
	Cookies []Cookie `json:"cookies,omitempty" yaml:"cookies"`

	// Optional. Define response http headers
	// This is equivalent to response metadata in GRPC
	Header map[string]string `json:"header,omitempty" yaml:"header"`

	// Error is optional. Defines response error for grpc
	// This is not applied for HTTP since body and status code can be used
	Error *ResponseError `json:"error,omitempty" yaml:"error"`

	// Optional. If defined, then executed template will override response data
	Template *Template `json:"template,omitempty" yaml:"template"`
}

// NewResponse creates new response
func NewResponse() *Response {
	return &Response{
		Header: map[string]string{},
		Body:   []byte{},
	}
}

// NewResponseFromHTTP parses from http response
func NewResponseFromHTTP(res *http.Response) *Response {
	r := &Response{
		StatusCode: res.StatusCode,
		Header:     map[string]string{},
		Body:       []byte{},
	}

	for key := range res.Header {
		r.Header[key] = res.Header.Get(key)
	}

	for _, c := range res.Cookies() {
		r.Cookies = append(r.Cookies, Cookie{Name: c.Name, Value: c.Value, ExpiredAt: c.Expires})
	}

	return r
}

// JSONResponse is convenient constructor to initialize response with JSON body
// The input parameter will be decoded to JSON
func JSONResponse(body interface{}) *Response {
	return NewResponse().WithBody(MustToJSON(body))
}

// XMLResponse is convenient constructor to initialize response with XML body
// The input parameter will be decoded to XML
func XMLResponse(body interface{}) *Response {
	return NewResponse().WithBody(MustToXML(body))
}

// HTMLResponse is convenient constructor to initialize response with htnl body
func HTMLResponse(html string) *Response {
	return NewResponse().WithBody(HTMLContent(html))
}

// Clone clones response and its properties
func (r *Response) Clone() *Response {
	nr := &Response{
		StatusCode: r.StatusCode,
		BodyFile:   r.BodyFile,
		Error:      r.Error.Clone(),
		Template:   r.Template,
		Header:     cloneStringMap(r.Header),
	}

	if r.Body != nil {
		nr.Body = make([]byte, len(r.Body))
		copy(nr.Body, r.Body)
	}

	if r.Cookies != nil {
		nr.Cookies = make([]Cookie, len(r.Cookies))
		copy(nr.Cookies, r.Cookies)
	}

	return nr
}

// Validate returns a non-nil error if invalid
func (r *Response) Validate(ctx context.Context) error {
	if r == nil {
		return nil
	}

	if r.Error != nil {
		for _, d := range r.Error.Details {
			if len(d.Type) == 0 {
				err := errors.New("missing type for error")
				log.Error(ctx, err)
				return err
			}
		}
	}

	return nil
}

// WithStatusCode sets the status code
func (r *Response) WithStatusCode(s int) *Response {
	r.StatusCode = s
	return r
}

// WithHeader sets header
func (r *Response) WithHeader(name string, value string) *Response {
	r.Header[name] = value
	return r
}

func (r *Response) WithError(msg string, details ...*ErrorDetail) *Response {
	if r.Error == nil {
		r.Error = &ResponseError{}
	}

	r.Error.Message = msg
	r.Error.Details = append(r.Error.Details, details...)
	return r
}

// WithRedirect sets redirect url
// Use WithStatusCode if want to customize the redirect code
func (r *Response) WithRedirect(url string) *Response {
	r.StatusCode = http.StatusTemporaryRedirect
	r.Header[HeaderLocation] = url
	return r
}

// WithCookie set cookies
func (r *Response) WithCookie(name string, value string) *Response {
	r.Cookies = append(r.Cookies, Cookie{Name: name, Value: value, ExpiredAt: time.Now().AddDate(0, 0, 1)})
	return r
}

// WithBody sets body
func (r *Response) WithBody(contentType string, body []byte) *Response {
	r.Body = body
	return r.WithHeader(HeaderContentType, contentType)
}

// WithFileBody sets file id for response body
// This should be used for large data response such as images, pdf, ... which are too large to store at database
// This can also be used to simulate the download request
// Handler will download file by this id and assign to body
func (r *Response) WithFileBody(contentType string, fileID string) *Response {
	r.BodyFile = fileID
	return r.WithHeader(HeaderContentType, contentType)
}

// LoadBodyFromFile loads body from file storage
func (r *Response) LoadBodyFromFile(ctx context.Context, fileStorage fs.FileStorage) error {
	data, err := fileStorage.DownloadFile(ctx, r.BodyFile)
	if err != nil {
		log.Error(ctx, "cannot download file", r.BodyFile, err)
		return err
	}

	r.Body, err = io.ReadAll(data)
	if err != nil {
		log.Error(ctx, "cannot read downloaded data", r.BodyFile, err)
		return err
	}

	log.Info(ctx, "downloaded file", r.BodyFile, "length", len(r.Body))
	return nil
}

// LoadBodyFromTemplate parses dynamic response from template
func (r *Response) LoadBodyFromTemplate(ctx context.Context, data *TemplateData) error {
	res, err := r.Template.Execute(ctx, data)
	if err != nil {
		log.Error(ctx, "cannot execute template", err)
		return err
	}

	res.AssignTo(r)
	return nil
}

// WriteTo writes response
func (r *Response) WriteTo(ctx context.Context, w http.ResponseWriter) error {
	for k, v := range r.Header {
		w.Header().Set(k, v)
	}

	for _, c := range r.Cookies {
		http.SetCookie(w, &http.Cookie{Name: c.Name, Value: c.Value, Expires: c.ExpiredAt})
	}

	if r.StatusCode == 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(r.StatusCode)
	}

	if _, err := w.Write(r.Body); err != nil {
		log.Error(ctx, err)
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	return nil
}

// Scan implements sqlx JSON scan method
func (r *Response) Scan(val interface{}) error {
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
func (r Response) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// MustToJSON converts a struct to JSON. Panic if error
func MustToJSON(v interface{}) (string, []byte) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		panic(err)
	}

	return ContentTypeJSON, buffer.Bytes()
}

// MustToXML converts a struct to XML. Panic if error
func MustToXML(v interface{}) (string, []byte) {
	b, err := xml.Marshal(v)
	if err != nil {
		panic(err)
	}

	return ContentTypeXML, b
}

// HTMLContent response html
func HTMLContent(html string) (string, []byte) {
	return ContentTypeHTML, []byte(html)
}
