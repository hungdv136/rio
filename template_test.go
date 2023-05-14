package rio

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTemplate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fileData, err := os.ReadFile("testdata/template.yaml")
	require.NoError(t, err)

	template := Template{}
	require.NoError(t, yaml.Unmarshal(fileData, &template))

	requestBody := types.Map{"naked_value": uuid.NewString()}
	request, err := http.NewRequestWithContext(ctx, "POST", "ts.com", strings.NewReader(requestBody.ForceJSON()))
	require.NoError(t, err)

	expectedCookie := &http.Cookie{Name: "SESSION_ID", Value: uuid.NewString()}
	request.AddCookie(expectedCookie)

	request.Header.Add("X-REQUEST-ID", uuid.NewString())

	data := &TemplateData{Request: request}
	res, err := template.Execute(ctx, data)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)

	require.Len(t, res.Cookies, 1)
	require.Equal(t, expectedCookie.Name, res.Cookies[0].Name)
	require.Equal(t, expectedCookie.Value, res.Cookies[0].Value)

	require.Len(t, res.Headers, 1)
	require.Equal(t, request.Header.Get("X-REQUEST-ID"), res.Headers["X-REQUEST-ID"])

	m, err := types.CreateMapFromReader(strings.NewReader(res.Body))
	require.NoError(t, err)
	require.NotEmpty(t, m.ForceString("encrypted_value"))
}
