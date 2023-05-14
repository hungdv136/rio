package rio

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio/internal/netkit"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

func TestStub_Match(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("matched", func(t *testing.T) {
		t.Parallel()

		searchTerm := uuid.NewString()
		requestID := uuid.NewString()
		sessionID := uuid.NewString()
		requestURL := "https://api.com/animal/create/" + uuid.NewString() + "?search_term=" + searchTerm
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		require.NoError(t, err)
		req.Header.Add("X-REQUEST-ID", requestID)
		req.AddCookie(&http.Cookie{Name: "SESSION_ID", Value: sessionID})

		stub := NewStub().
			For("GET", Contains("animal/create")).
			WithHeader("X-REQUEST-ID", EqualTo(requestID)).
			WithQuery("search_term", EqualTo(searchTerm)).
			WithCookie("SESSION_ID", EqualTo(sessionID))

		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.True(t, matched)
	})

	t.Run("unmatched", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/animal/create/", nil)
		require.NoError(t, err)

		stub := NewStub().For("GET", Contains("animal/create")).WithCookie("SESSION_ID", EqualTo(uuid.NewString()))
		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.False(t, matched)
	})

	t.Run("regex", func(t *testing.T) {
		t.Parallel()

		requestURL := "https://api.com/animal/123456/create"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		require.NoError(t, err)

		stub := NewStub().For("GET", Regex("animal/[0-9]{3,7}/create"))
		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.True(t, matched)

		stub = NewStub().For("GET", Regex("animal/admin/123/create"))
		matched, err = stub.Match(ctx, req)
		require.NoError(t, err)
		require.False(t, matched)
	})

	t.Run("body_json", func(t *testing.T) {
		t.Parallel()

		randomValue := uuid.NewString()
		data := types.Map{
			"key_1": types.Map{
				"key_2": randomValue,
			},
		}

		requestURL := "https://api.com/animal/create"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(data.ForceJSON()))
		require.NoError(t, err)
		req.Header.Add(HeaderContentType, ContentTypeJSON)

		stub := NewStub().For("POST", Contains("animal/create")).WithRequestBody(BodyJSONPath("$.key_1.key_2", EqualTo(randomValue)))
		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.True(t, matched)

		stub = NewStub().For("POST", Contains("animal/create")).WithRequestBody(BodyJSONPath("$.key_1.key_n", EqualTo(randomValue)))
		matched, err = stub.Match(ctx, req)
		require.NoError(t, err)
		require.False(t, matched)
	})

	t.Run("body_xml", func(t *testing.T) {
		t.Parallel()

		randomValue := uuid.NewString()
		data := types.Map{
			"key_1": types.Map{
				"key_2": randomValue,
			},
		}

		requestURL := "https://api.com/animal/create"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(data.ForceJSON()))
		require.NoError(t, err)
		req.Header.Add(HeaderContentType, ContentTypeJSON)

		stub := NewStub().For("POST", Contains("animal/create")).WithRequestBody(BodyJSONPath("$.key_1.key_2", EqualTo(randomValue)))
		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.True(t, matched)

		stub = NewStub().For("POST", Contains("animal/create")).WithRequestBody(BodyJSONPath("$.key_1.key_n", EqualTo(randomValue)))
		matched, err = stub.Match(ctx, req)
		require.NoError(t, err)
		require.False(t, matched)
	})

	t.Run("multiple_part", func(t *testing.T) {
		t.Parallel()

		metadata := map[string]string{
			"key_1": uuid.NewString(),
			"key_2": uuid.NewString(),
		}
		fileContents, err := os.ReadFile("stub_test.go")
		require.NoError(t, err)

		req, err := netkit.NewUploadRequest(ctx, "/animal/image/upload", fileContents, metadata)
		require.NoError(t, err)

		stub := NewStub().For("POST", Contains("animal/image/upload")).WithRequestBody(MultiPartForm("key_1", EqualTo(metadata["key_1"])))
		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.True(t, matched)

		stub = NewStub().For("POST", Contains("animal/image/upload")).WithRequestBody(MultiPartForm("key_", EqualTo(uuid.NewString())))
		matched, err = stub.Match(ctx, req)
		require.NoError(t, err)
		require.False(t, matched)
	})

	t.Run("body_url_encoded", func(t *testing.T) {
		t.Parallel()

		requestURL := "https://api.com/animal/create"

		body := strings.NewReader(`CustomerID=352461777&CustomerPhone=0988777666&typebank=ATM_ON&DefaultProviderID=a6f07639-83a1-45fd-a0e0-85217ff75de2&ProviderCode=MFI&ProviderName=Doctor+Dong&ServiceCode=TTTG&ServiceID=1&ServiceName=Thanh+to%C3%A1n+tr%E1%BA%A3+g%C3%B3p&ProviderID=a6f07639-83a1-45fd-a0e0-85217ff75de2&ncc=Doctor+Dong&inforbill=Kh%C3%A1ch+h%C3%A0ng+thanh+to%C3%A1n+n%E1%BB%A3+06%2F2022&cardtype=ACB`)
		req, err := http.NewRequestWithContext(ctx, "POST", requestURL, body)
		require.NoError(t, err)
		req.Header.Add(HeaderContentType, ContentTypeForm)

		stub := NewStub().For("POST", Contains("animal/create")).WithRequestBody(URLEncodedBody("CustomerID", EqualTo("352461777")))
		matched, err := stub.Match(ctx, req)
		require.NoError(t, err)
		require.True(t, matched)

		stub = NewStub().For("POST", Contains("animal/create")).WithRequestBody(URLEncodedBody("CustomerID", EqualTo(uuid.NewString())))
		matched, err = stub.Match(ctx, req)
		require.NoError(t, err)
		require.False(t, matched)
	})
}

func TestSelectStubs(t *testing.T) {
	t.Parallel()

	t.Run("without_weight", func(t *testing.T) {
		t.Parallel()

		stubs := []*Stub{
			NewStub().WithID(2),
			NewStub().WithID(3),
			NewStub().WithID(1),
		}

		foundStub := SelectStubs(stubs)
		require.Equal(t, stubs[2], foundStub)
	})

	t.Run("with_weight", func(t *testing.T) {
		t.Parallel()

		stubs := []*Stub{
			NewStub().WithID(3).WithWeight(1),
			NewStub().WithID(2).WithWeight(3),
			NewStub().WithID(1).WithWeight(2),
		}

		foundStub := SelectStubs(stubs)
		require.Equal(t, stubs[2], foundStub)
	})
}

func Benchmark_Match(b *testing.B) {
	ctx := context.Background()
	searchTerm := uuid.NewString()
	requestID := uuid.NewString()
	sessionID := uuid.NewString()
	bodyFieldValue := uuid.NewString()
	requestURL := "https://api.com/animal/create?search_term=" + searchTerm

	stub := NewStub().
		For("POST", Contains("animal/create")).
		WithHeader("X-REQUEST-ID", EqualTo(requestID)).
		WithQuery("search_term", EqualTo(searchTerm)).
		WithCookie("SESSION_ID", EqualTo(sessionID)).
		WithRequestBody(BodyJSONPath("$.key_1", EqualTo(bodyFieldValue)))

	data := types.Map{"key_1": bodyFieldValue}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(data.ForceJSON()))
	require.NoError(b, err)
	req.Header.Add(HeaderContentType, ContentTypeJSON)
	req.Header.Add("X-REQUEST-ID", requestID)
	req.AddCookie(&http.Cookie{Name: "SESSION_ID", Value: sessionID})

	matched := false
	for i := 0; i < b.N; i++ {
		matched, err = stub.Match(ctx, req)
	}

	require.NoError(b, err)
	require.True(b, matched)
}
