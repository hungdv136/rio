package rio

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandler_rewritePath(t *testing.T) {
	t.Parallel()

	namespace := uuid.NewString()
	tests := [...]struct {
		namespace string
		input     string
		want      string
	}{
		{input: "/echo/animal/create", want: "/animal/create"},
		{input: "/" + namespace + "/echo/animal/get", want: "/animal/get", namespace: namespace},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			h := &Handler{basePath: "/echo/", namespace: tc.namespace}
			got := h.rewritePath(tc.input)
			require.Equal(t, tc.want, got)
		})
	}
}
