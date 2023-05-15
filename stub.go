package rio

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/hungdv136/rio/internal/log"
)

// Defines request content types
const (
	ContentTypeJSON      = "application/json"
	ContentTypeXML       = "text/xml"
	ContentTypeHTML      = "text/html"
	ContentTypeText      = "text/plain"
	ContentTypeMultipart = "multipart/form-data"
	ContentTypeForm      = "application/x-www-form-urlencoded"
)

// Defines request header
const (
	HeaderContentType   = "Content-Type"
	HeaderContentLength = "Content-Length"
	HeaderLocation      = "Location"
	HeaderXRequestID    = "X-Request-Id"
)

const (
	MethodGrpc   = "grpc"
	ProtocolHTTP = "http"
	ProtocolGrpc = "grpc"
)

// Used for recorded stub when recording is enabled
const TagRecordedStub = "recorded_stub"

type ArrayStubs struct {
	Stubs []*Stub `json:"stubs" yaml:"stubs"`
}

// Stub defines a stub
type Stub struct {
	ID int64 `json:"id" yaml:"id"`

	// Description describes the stub
	Description string `json:"description,omitempty" yaml:"description"`

	// WithNamespace sets namespace which can be used for isolating test data for each testing
	// This can be applied for the unit test which uses different test server for different test
	// With integration or unit test with single test server (which is not recommended), namespace should be empty
	Namespace string `json:"namespace,omitempty" yaml:"namespace"`

	// Tag is to add custom tag for grouping stub
	// Owner can add whatever they want to classify the stub
	Tag string `json:"tag,omitempty" yaml:"tag"`

	// Protocol defines protocol of incoming requests
	// Value is either: http or grpc. Default value is http
	Protocol string `json:"protocol,omitempty" yaml:"protocol"`

	// Matching rules which will be matched against the incoming requests
	Request *RequestMatching `json:"request,omitempty" yaml:"request"`

	// The expected response which includes the body, header and cookies
	Response *Response `json:"response,omitempty" yaml:"response"`

	// The mock server will act as reserved proxy if this settings are provided
	Proxy *Proxy `json:"proxy,omitempty" yaml:"proxy"`

	Active bool `json:"active,omitempty" yaml:"active"`

	// WithWeight sets weight. The higher weight, the higher priority
	// It is sometimes the case that you'll want to declare two or more stub mappings that "overlap",
	// in that a given request would be a match for more than one of them
	// By default, the most recently added matching stub will be used to satisfy the request
	// However, in some cases it is useful to exert more control
	Weight int `json:"weight,omitempty" yaml:"weight"`

	Settings StubSettings `json:"settings,omitempty" yaml:"settings"`

	CreatedAt time.Time `json:"created_at,omitempty" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty" yaml:"updated_at"`
}

// Validate returns an non-nil error if stub is invalid
func (s *Stub) Validate(ctx context.Context) error {
	if err := s.Request.Validate(ctx); err != nil {
		return err
	}

	if s.Proxy == nil && s.Response == nil {
		err := errors.New("proxy or response must be defined")
		log.Error(ctx, err)
		return err
	}

	if err := s.Response.Validate(ctx); err != nil {
		return err
	}

	return nil
}

// IsReversed returns true if stub is reverse proxy
func (s *Stub) IsReversed() bool {
	return s.Proxy != nil && len(s.Proxy.TargetURL) > 0
}

// HasTemplate returns true if response template is defined
func (s *Stub) HasTemplate() bool {
	return s.Response != nil && s.Response.Template != nil && len(s.Response.Template.Script) > 0
}

// Clone clones stubs
func (s *Stub) Clone() *Stub {
	return &Stub{
		ID:          s.ID,
		Description: s.Description,
		Namespace:   s.Namespace,
		Request:     s.Request,
		Response:    s.Response.Clone(),
		Proxy:       s.Proxy,
		Active:      s.Active,
		Settings:    s.Settings,
		Tag:         s.Tag,
		Protocol:    s.Protocol,
		Weight:      s.Weight,
	}
}

// Proxy defines proxy settings
type Proxy struct {
	// TargetURL is the root url of the target server
	// The relative path will be parsed from incoming request
	TargetURL string `json:"target_url,omitempty" yaml:"target_url"`

	// TargetPath is the relative path of the target API
	// This is optional field, if not provided, it will be the same as original request path
	TargetPath string `json:"target_path,omitempty" yaml:"target_path"`

	// EnableRecord is to enable/disable recording response from remote server
	// A stub will be automatically created in stub store
	EnableRecord bool `json:"enable_record,omitempty" yaml:"enable_record"`
}

// Scan implements sqlx JSON scan method
func (r *Proxy) Scan(val interface{}) error {
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
func (r Proxy) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// StubSettings stub settings
type StubSettings struct {
	// This is to disable the matched stub, it is not used for the next request.
	// In the following example, the first request will return the first stub with higher weight,
	// then that stub is not available for the next request anymore
	DeactivateWhenMatched bool `json:"deactivate_when_matched,omitempty" yaml:"deactivate_when_matched"`

	// It is sometimes we want to simulate slow response API.
	// Rio supports this feature by set delay duration
	DelayDuration time.Duration `json:"delay_duration,omitempty" swaggertype:"primitive,integer" yaml:"delay_duration"`

	// StoreVersion is a system field to control data structure version for stub
	// Value will be overrided by system
	StoreVersion int `json:"store_version,omitempty" yaml:"store_version"`
}

// Scan implements sqlx JSON scan method
func (r *StubSettings) Scan(val interface{}) error {
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
func (r StubSettings) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Clone clones new instance
func (r *StubSettings) Clone() *StubSettings {
	return &StubSettings{
		DeactivateWhenMatched: r.DeactivateWhenMatched,
		DelayDuration:         r.DelayDuration,
		StoreVersion:          r.StoreVersion,
	}
}

// NewStub returns a new stub
func NewStub() *Stub {
	return &Stub{
		Response: NewResponse(),
		Active:   true,
	}
}

// WithNamespace sets namespace which can be used for isolating test data for each testing
// This can be applied for the unit test which uses different test server for different test
// With integration or unit test with single test server (which is not recommended), namespace should be empty
func (s *Stub) WithNamespace(namespace string) *Stub {
	s.Namespace = namespace
	return s
}

// WithTag sets tag for stub
func (s *Stub) WithTag(tag string) *Stub {
	s.Tag = tag
	return s
}

// WithID sets id
func (s *Stub) WithID(id int64) *Stub {
	s.ID = id
	return s
}

// For sets http request method and matching functions for URL
// If method is empty, it will be ignored with matching logic
func (s *Stub) For(method string, urlMatchingFuncs ...CreateOperator) *Stub {
	s.Request = &RequestMatching{
		Header: []FieldOperator{},
		Query:  []FieldOperator{},
		Cookie: []FieldOperator{},
		Body:   []BodyOperator{},
	}

	s.Request.Method = method

	if len(s.Protocol) == 0 && method != MethodGrpc {
		s.Protocol = ProtocolHTTP
	}

	for _, matchingFunc := range urlMatchingFuncs {
		s.Request.URL = append(s.Request.URL, matchingFunc())
	}

	return s
}

// ForAny for matching request with any method
func (s *Stub) ForAny(urlMatchingFuncs ...CreateOperator) *Stub {
	return s.For("", urlMatchingFuncs...)
}

// ForGRPC for matching request with grpc method
func (s *Stub) ForGRPC(urlMatchingFuncs ...CreateOperator) *Stub {
	s.Protocol = ProtocolGrpc
	return s.For(MethodGrpc, urlMatchingFuncs...)
}

// WithDescription sets description
func (s *Stub) WithDescription(desc string) *Stub {
	s.Description = desc
	return s
}

// WithHeader sets header matching operator
func (s *Stub) WithHeader(name string, createFunc CreateOperator) *Stub {
	op := FieldOperator{FieldName: name, Operator: createFunc()}
	s.Request.Header = append(s.Request.Header, op)
	return s
}

// WithCookie sets cookie matching operator
func (s *Stub) WithCookie(name string, createFunc CreateOperator) *Stub {
	op := FieldOperator{FieldName: name, Operator: createFunc()}
	s.Request.Cookie = append(s.Request.Cookie, op)
	return s
}

// WithQuery sets query matching operator
func (s *Stub) WithQuery(name string, createFunc CreateOperator) *Stub {
	op := FieldOperator{FieldName: name, Operator: createFunc()}
	s.Request.Query = append(s.Request.Query, op)
	return s
}

// WithRequestBody sets body matching operator
func (s *Stub) WithRequestBody(createFunc CreateBodyOperator) *Stub {
	s.Request.Body = append(s.Request.Body, createFunc())
	return s
}

// WillReturn sets the response
func (s *Stub) WillReturn(resp *Response) *Stub {
	s.Response = resp
	return s
}

// WithWeight sets weight. The higher weight, the higher priority
// It is sometimes the case that you'll want to declare two or more stub mappings that "overlap",
// in that a given request would be a match for more than one of them
// By default, the most recently added matching stub will be used to satisfy the request
// However, in some cases it is useful to exert more control
func (s *Stub) WithWeight(weight int) *Stub {
	s.Weight = weight
	return s
}

// WithInactive sets to inactive
func (s *Stub) WithInactive() *Stub {
	s.Active = false
	return s
}

// ShouldDeactivateWhenMatched deactivates when matched
func (s *Stub) ShouldDeactivateWhenMatched() *Stub {
	s.Settings.DeactivateWhenMatched = true
	return s
}

// ShouldDelay sets delay duration
// Use this to simulate the slow API response time
func (s *Stub) ShouldDelay(d time.Duration) *Stub {
	s.Settings.DelayDuration = d
	return s
}

// WithTargetURL sets base target url, request will be forwarded to the given url
func (s *Stub) WithTargetURL(url string) *Stub {
	if s.Proxy == nil {
		s.Proxy = &Proxy{}
	}

	s.Proxy.TargetURL = url
	return s
}

// WithEnableRecord enables recording proxy as a stub
func (s *Stub) WithEnableRecord(v bool) *Stub {
	if s.Proxy == nil {
		s.Proxy = &Proxy{}
	}

	s.Proxy.EnableRecord = v
	return s
}

// Send submits stub to server for matching upcoming requests
func (s *Stub) Send(ctx context.Context, server Server) error {
	if ReleaseMode == Debug {
		log.Info(ctx, s.ForceJSON(ctx))
	}

	return server.Create(ctx, s)
}

// ForceJSON marshall to json
func (s *Stub) ForceJSON(ctx context.Context) string {
	data, err := json.Marshal(s)
	if err != nil {
		log.Error(ctx, err)
		return "unable to marshall"
	}

	return string(data)
}

// SelectStubs gets the item with the highest priority. If weight is not set, then get the latest
func SelectStubs(stubs []*Stub) *Stub {
	if len(stubs) == 0 {
		return nil
	}

	sort.SliceStable(stubs, func(i, j int) bool {
		if stubs[i].Weight != stubs[j].Weight {
			return stubs[i].Weight < stubs[j].Weight
		}

		return stubs[i].ID < stubs[j].ID
	})

	// Return cloned stub to avoid override the original stub
	return stubs[len(stubs)-1].Clone()
}
