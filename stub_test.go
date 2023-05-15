package rio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
