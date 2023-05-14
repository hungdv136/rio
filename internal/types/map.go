package types

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/util"
)

// Map is a map from string to interface
type Map map[string]interface{}

// CreateMapFromStruct creates map from struct
func CreateMapFromStruct(obj interface{}) (Map, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return CreateMapFromReader(bytes.NewReader(b))
}

// CreateMapFromReader creates map from a JSON reader
func CreateMapFromReader(reader io.Reader) (Map, error) {
	m := Map{}

	if reader == nil {
		return m, nil
	}

	// numbers are represented as string instead of float64
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()

	err := decoder.Decode(&m)
	if err == io.EOF {
		return m, nil
	}

	return m, err
}

// ForceString returns value of key as a string, ignores error
func (m Map) ForceString(key string) string {
	val, ok := m.GetString(key)
	if !ok {
		log.Warn(context.TODO(), "value of", key, "is not a string")
	}
	return val
}

// GetString returns value of a key as string
func (m Map) GetString(key string) (string, bool) {
	s, ok := m[key].(string)
	return s, ok
}

// GetArrayString returns value of key as an array of string
func (m Map) GetArrayString(key string) ([]string, bool) {
	a, ok := m[key].([]string)
	if ok {
		return a, true
	}

	c, ok := m[key].([]interface{})
	if !ok {
		return nil, false
	}

	return util.ToArrayString(c)
}

// JSON converts map to JSON string
func (m Map) JSON() ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// ForceJSON converts map to JSON string
func (m Map) ForceJSON() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}

	return string(b)
}

// GetMap returns value of key as map
func (m Map) GetMap(key string) (Map, bool) {
	v, ok := m[key].(map[string]interface{})
	if ok {
		return Map(v), true
	}
	u, ok := m[key].(Map)
	if ok {
		return u, true
	}
	return Map{}, ok
}

// ForceMap returns value of key as a map, ignores error
func (m Map) ForceMap(key string) Map {
	val, ok := m.GetMap(key)
	if !ok {
		log.Warn(context.TODO(), "value of", key, "is not a map")
	}
	return val
}

// Scan sqlx JSON scan method
func (m *Map) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, &m)
	case string:
		return json.Unmarshal([]byte(v), &m)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Value sqlx JSON value method
func (m Map) Value() (driver.Value, error) {
	return json.Marshal(m)
}
