package netkit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

// TestCase is a test case for HTTP request
type TestCase struct {
	Name          string
	Method        string
	Path          string
	AuthToken     string
	Params        types.Map
	Headers       types.Map
	ExpectStatus  int
	ExpectVerdict string
	ExpectHeaders types.Map
	FilePath      string
}

// NewTestCase returns a new HTTP test case
func NewTestCase(
	name string,
	method string,
	path string,
	params types.Map,
	expectStatus int,
	expectVerdict string,
) *TestCase {
	return &TestCase{
		Name:          name,
		Method:        method,
		Path:          path,
		Params:        params,
		Headers:       types.Map{},
		ExpectStatus:  expectStatus,
		ExpectVerdict: expectVerdict,
		ExpectHeaders: types.Map{},
	}
}

func (tc *TestCase) buildRequest(ctx context.Context) (*http.Request, error) {
	if strings.EqualFold(tc.Method, http.MethodGet) {
		return NewQueryRequest(ctx, tc.Method, tc.Path, tc.Params)
	}

	return NewJSONRequest(ctx, tc.Method, tc.Path, tc.Params)
}

// ExecuteTestCase execute test case with gin engine
func ExecuteTestCase[Body any](t *testing.T, tc *TestCase, engine *gin.Engine) *Response[InternalBody[Body]] {
	ctx := log.SaveID(context.Background(), tc.Name)
	req, err := tc.buildRequest(ctx)
	require.NoError(t, err)

	recorder := NewResponseRecorder()
	engine.ServeHTTP(recorder, req)
	result := recorder.Result()
	defer result.Body.Close()

	require.Equal(t, tc.ExpectStatus, result.StatusCode)

	for k, v := range tc.ExpectHeaders {
		require.Equal(t, v, result.Header.Get(k))
	}

	res, err := ParseResponse[InternalBody[Body]](ctx, result)
	require.NoError(t, err)
	require.Equal(t, tc.ExpectVerdict, res.Body.Verdict)

	return res
}

// ResponseRecorder wraps recorder to support CloseNotify
type ResponseRecorder struct {
	*httptest.ResponseRecorder
	closeChannel chan bool
}

// CloseNotify waits for closed message
// This is required method for go-gin framework
func (r *ResponseRecorder) CloseNotify() <-chan bool {
	return r.closeChannel
}

// NewResponseRecorder returns a new instance
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{httptest.NewRecorder(), make(chan bool, 1)}
}
