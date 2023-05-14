package types

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapGetMap(t *testing.T) {
	t.Parallel()

	m := Map{
		"home_credit": map[string]interface{}{
			"create_otp_request": Map{
				"method": "POST",
				"path":   "/otp_requests/create",
			},
			"create_score_request": Map{
				"method": "POST",
				"path":   "/score_requests/create",
			},
		},
		"trusting_social": Map{
			"get_score_request": Map{
				"method": "GET",
				"path":   "/score_requests/get",
			},
		},
	}

	ts, ok := m.GetMap("trusting_social")
	require.True(t, ok)
	require.NotNil(t, ts)
	hc, ok := m.GetMap("home_credit")
	require.True(t, ok)
	require.NotNil(t, hc)
	cor, ok := hc.GetMap("create_otp_request")
	require.True(t, ok)
	require.NotNil(t, cor)
	str, ok := m.GetMap("verdict")
	require.False(t, ok)
	require.NotNil(t, str)
	require.Empty(t, str)
	none, ok := m.GetMap("none")
	require.False(t, ok)
	require.NotNil(t, none)
	require.Empty(t, none)
}

func TestMapForceMap(t *testing.T) {
	t.Parallel()

	m := Map{
		"home_credit": map[string]interface{}{
			"create_otp_request": Map{
				"method": "POST",
				"path":   "/otp_requests/create",
			},
			"create_score_request": Map{
				"method": "POST",
				"path":   "/score_requests/create",
			},
		},
		"trusting_social": Map{
			"get_score_request": Map{
				"method": "GET",
				"path":   "/score_requests/get",
			},
		},
	}

	ts := m.ForceMap("trusting_social")
	require.NotNil(t, ts)
	hc := m.ForceMap("home_credit")
	require.NotNil(t, hc)
	cor := hc.ForceMap("create_otp_request")
	require.NotNil(t, cor)
	str := m.ForceMap("verdict")
	require.NotNil(t, str)
	require.Empty(t, str)
	none := m.ForceMap("none")
	require.NotNil(t, none)
	require.Empty(t, none)
}

func TestMapGetString(t *testing.T) {
	t.Parallel()
	m := Map{"x": false, "y": "true", "z": "zero", "a": 1, "b": "false", "c": true}
	testCases := []struct {
		Name  string
		Param string
		OK    bool
		Value string
	}{
		{"false", "x", false, ""},
		{"string_true", "y", true, "true"},
		{"string", "z", true, "zero"},
		{"int", "a", false, ""},
		{"string_false", "b", true, "false"},
		{"true", "c", false, ""},
		{"missing", "missing_key", false, ""},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			value, ok := m.GetString(tc.Param)
			require.Equal(t, tc.OK, ok)
			require.Equal(t, tc.Value, value)
		})
	}
}

func TestMapForceString(t *testing.T) {
	t.Parallel()

	m := Map{"x": false, "y": "true", "z": "zero", "a": 1, "b": "false", "c": true}
	testCases := []struct {
		Name  string
		Param string
		Value string
	}{
		{"false", "x", ""},
		{"string_true", "y", "true"},
		{"string", "z", "zero"},
		{"int", "a", ""},
		{"string_false", "b", "false"},
		{"true", "c", ""},
		{"missing", "missing_key", ""},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			value := m.ForceString(tc.Param)
			require.Equal(t, tc.Value, value)
		})
	}
}

func TestCreateMapFromReader(t *testing.T) {
	t.Parallel()

	t.Run("nil reader", func(t *testing.T) {
		t.Parallel()

		m, err := CreateMapFromReader(nil)
		require.Nil(t, err)
		require.Empty(t, m)
	})

	t.Run("empty reader", func(t *testing.T) {
		t.Parallel()

		reader := bytes.NewReader([]byte{})
		m, err := CreateMapFromReader(reader)
		require.Nil(t, err)
		require.Empty(t, m)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		reader := strings.NewReader(`"x": 1`)
		m, err := CreateMapFromReader(reader)
		require.Error(t, err)
		require.Empty(t, m)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		reader := strings.NewReader("")
		m, err := CreateMapFromReader(reader)
		require.Nil(t, err)
		require.Empty(t, m)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		reader := strings.NewReader(`{"x": 1}`)
		m, err := CreateMapFromReader(reader)
		require.Nil(t, err)
		require.NotEmpty(t, m)
	})
}

func TestCreateMapFromStruct(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		type Fruit struct {
			Name       string `json:"name"`
			WeightGram int    `json:"weight_gram"`
		}
		apple := Fruit{Name: "apple", WeightGram: 300}
		mapple, err := CreateMapFromStruct(apple)
		require.Nil(t, err)
		require.NotNil(t, mapple)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		m, err := CreateMapFromStruct([]string{"an", "array"})
		require.Error(t, err)
		require.Empty(t, m)
	})
}
