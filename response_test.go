package rio

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMustToJSON(t *testing.T) {
	t.Parallel()

	c := &Cookie{
		Name:      uuid.NewString(),
		Value:     uuid.NewString(),
		ExpiredAt: time.Now(),
	}

	contentType, d := MustToJSON(c)
	require.Equal(t, ContentTypeJSON, contentType)
	require.NotEmpty(t, d)

	uc := Cookie{}
	err := json.Unmarshal(d, &uc)
	require.NoError(t, err)
	require.Equal(t, c.Name, uc.Name)
	require.Equal(t, c.Value, uc.Value)
}
