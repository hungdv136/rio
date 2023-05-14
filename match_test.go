package rio

import (
	"context"
	"testing"

	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

func TestExecuteEqualToOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedValue  interface{}
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"matched_int", 12, 12, true, false},
		{"matched_different_int", 14, int64(14), true, false},
		{"unmatched_int", 12, 21, false, false},
		{"matched_float", 12.000, 12.00, true, false},
		{"matched_different_float", 12.000, float64(12.00), true, false},
		{"matched_string", "this is string", "this is string", true, false},
	}

	for _, tc := range testCases {
		matched, err := executeEqualToOperator(ctx, EqualTo(tc.PredefinedValue)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteContainingOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedValue  interface{}
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"matched_string", "this is string", "Hum, this is string from payload data", true, false},
		{"unmatched_string", "this is string 1", "another string", false, false},
		{"unmatched_different_type", 12, "Hum, this is string from payload data", false, false},
	}

	for _, tc := range testCases {
		matched, err := executeContainingOperator(ctx, Contains(tc.PredefinedValue)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteNotContainingOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedValue  interface{}
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"matched_string", "this is string", "Hum, this is string from payload data", false, false},
		{"unmatched_string", "this is string 1", "another string", true, false},
		{"unmatched_different_type", 12, "Hum, this is string from payload data", true, false},
	}

	for _, tc := range testCases {
		matched, err := Match(ctx, NotContains(tc.PredefinedValue)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteStartWithOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedValue  string
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"with_prefix", "this is string", "this is string from payload data", true, false},
		{"unmatched_string", "this is string", "Hello, this is string", false, false},
		{"unsupported_type", "string", 34, false, true},
	}

	for _, tc := range testCases {
		matched, err := executeStartWithOperator(ctx, StartWith(tc.PredefinedValue)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteEndWithOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedValue  string
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"with_prefix", "this is string", "hello, this is string", true, false},
		{"unmatched_string", "this is string", "this is string from data", false, false},
		{"unsupported_type", "string", 34, false, true},
	}

	for _, tc := range testCases {
		matched, err := executeEndWithOperator(ctx, EndWith(tc.PredefinedValue)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteRegexWithOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedValue  string
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"matched", "/animal/[0-9]{3,7}/create", "/animal/123456/create", true, false},
		{"unmatched", "/animal/[0-9]{3,7}/create", "/animal/abcdfg/create", false, false},
		{"unsupported_type", "/animal/[0-9](3,7)/create", 34, false, true},
	}

	for _, tc := range testCases {
		matched, err := executeRegexOperator(ctx, Regex(tc.PredefinedValue)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteLengthOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		PredefinedLen    int
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"string", 6, "string", true, false},
		{"map", 2, types.Map{"key_1": 1, "key_2": 2}, true, false},
		{"array_integer", 3, []int{1, 2, 4}, true, false},
		{"array_float32", 4, []float32{1, 2, 3, 4}, true, false},
		{"array_map", 2, []types.Map{{}, {}}, true, false},
	}

	for _, tc := range testCases {
		matched, err := executeLengthOperator(ctx, Length(tc.PredefinedLen)(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched)
		}
	}
}

func TestExecuteNotEmptyOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"string", "string", true, false},
		{"map", types.Map{"key_1": 1, "key_2": 2}, true, false},
		{"array_integer", []int{1, 2, 4}, true, false},
		{"array_float32", []float32{1, 2, 3, 4}, true, false},
		{"array_map", []types.Map{{}, {}}, true, false},
		{"nil", nil, false, false},
		{"empty_string", "", false, false},
		{"empty_array", []float32{}, false, false},
	}

	for _, tc := range testCases {
		matched, err := Match(ctx, NotEmpty()(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched, tc.Name)
		}
	}
}

func TestExecuteEmptyOperator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testCases := []struct {
		Name             string
		ActualValue      interface{}
		ExpectedResult   bool
		ExpectedHasError bool
	}{
		{"string", "string", false, false},
		{"nil", nil, true, false},
		{"empty_string", "", true, false},
		{"empty_array", []float32{}, true, false},
	}

	for _, tc := range testCases {
		matched, err := Match(ctx, Empty()(), tc.ActualValue)
		if tc.ExpectedHasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedResult, matched, tc.Name)
		}
	}
}

func Benchmark_ExecuteRegexOperator(b *testing.B) {
	ctx := context.Background()
	op := Regex("/animal/[0-9]{3,7}/create")()
	var (
		matched bool
		err     error
	)

	for i := 0; i < b.N; i++ {
		matched, err = executeRegexOperator(ctx, op, "http://api.trustingsocial.com/animal/123456/create")
	}

	require.NoError(b, err)
	require.True(b, matched)
}
