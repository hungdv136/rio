package api

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

// func newRequest(t *testing.T, tc *httpkit.TestCase) *http.Request {
// 	ctx := context.Background()
// 	if tc.Method == http.MethodPost {
// 		req, err := http.NewRequestWithContext(ctx, tc.Method, tc.Path, bytes.NewReader([]byte(tc.Params.ForceJSON())))
// 		require.NoError(t, err)
// 		req.Header.Set(HeaderContentType, netkit.ContentTypeJSON)
// 		return req
// 	}

// 	urlValues := url.Values{}
// 	for k, v := range tc.Params {
// 		urlValues.Add(k, v.(string))
// 	}

// 	url := tc.Path
// 	if len(urlValues) > 0 {
// 		url = url + "?" + urlValues.Encode()
// 	}

// 	req, err := http.NewRequestWithContext(ctx, tc.Method, url, nil)
// 	require.NoError(t, err)
// 	return req
// }

func parseResponse(t *testing.T, r io.Reader) types.Map {
	data := types.Map{}
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	err := decoder.Decode(&data)
	require.NoError(t, err)
	return data
}

func parseJSONFileToMap(t *testing.T, filePath string) types.Map {
	rawJSON, err := os.ReadFile(filePath)
	require.NoError(t, err)

	data, err := types.CreateMapFromReader(bytes.NewReader(rawJSON))
	require.NoError(t, err)
	return data
}
