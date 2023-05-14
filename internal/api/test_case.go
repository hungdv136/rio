package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hungdv136/rio/internal/types"
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

// Execute execute test case with gin engine
func (tc *TestCase) Execute(t *testing.T, engine *gin.Engine) types.Map {
	// ctx := log.SaveID(context.Background(), tc.Name)

	// var (
	// 	req *http.Request
	// 	err error
	// )

	// if tc.Method == http.MethodPost {
	// 	req, err = http.NewRequestWithContext(ctx, tc.Method, tc.Path, nil)
	// 	require.NoError(t, err)
	// } else {

	// }

	// w := NewResponseRecorder()
	// engine.ServeHTTP(w, req)
	// result := w.Result() // nolint:bodyclose

	// require.Equal(t, tc.ExpectStatus, result.StatusCode)
	// for k, v := range tc.ExpectHeaders {
	// 	require.Equal(t, v, result.Header.Get(k))
	// }

	// resData := types.Map{}
	// err = httpkit.ParseResponse(ctx, result, &resData)
	// require.NoError(t, err)
	// require.Equal(t, tc.ExpectVerdict, resData.ForceString("verdict"))

	// return resData

	return types.Map{}
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
