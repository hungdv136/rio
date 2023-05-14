package rio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	r, err := defaultRegexCompiler.compile(ctx, "^[0-9A-Za-z]{1,12}$")
	require.NoError(t, err)
	require.NotNil(t, r)

	cachedRegex, err := defaultRegexCompiler.compile(ctx, "^[0-9A-Za-z]{1,12}$")
	require.NoError(t, err)
	require.Equal(t, r, cachedRegex)

	otherRegex, err := defaultRegexCompiler.compile(ctx, "^[0-9A-Za-z]{1,1}$")
	require.NoError(t, err)
	require.NotNil(t, otherRegex)
	require.NotEqual(t, r, otherRegex)
}
