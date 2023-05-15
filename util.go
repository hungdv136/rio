package rio

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Try to get length of object
func getLen(x interface{}) (length int, ok bool) {
	v := reflect.ValueOf(x)
	defer func() {
		if err := recover(); err != nil {
			ok = false
		}
	}()

	return v.Len(), true
}

// Gets whether the specified object is considered empty or not.
func isEmpty(object interface{}) bool {
	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0

	// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)

	// for all other types, compare against the zero value
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

// Try loop over the list check if the list includes the element.
func containsElement(list interface{}, element interface{}) (ok, found bool) {
	listValue := reflect.ValueOf(list)
	elementValue := reflect.ValueOf(element)
	defer func() {
		if e := recover(); e != nil {
			ok = false
			found = false
		}
	}()

	if reflect.TypeOf(list).Kind() == reflect.String {
		return true, strings.Contains(listValue.String(), elementValue.String())
	}

	if reflect.TypeOf(list).Kind() == reflect.Map {
		mapKeys := listValue.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if objectsAreEqual(mapKeys[i].Interface(), element) {
				return true, true
			}
		}
		return true, false
	}

	for i := 0; i < listValue.Len(); i++ {
		if objectsAreEqual(listValue.Index(i).Interface(), element) {
			return true, true
		}
	}

	return true, false
}

// Determines if two objects are considered equal.
// This function does no assertion of any kind.
func objectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	if exp, ok := expected.([]byte); ok {
		act, ok := actual.([]byte)
		if !ok {
			return false
		} else if exp == nil || act == nil {
			return exp == nil && act == nil
		}
		return bytes.Equal(exp, act)
	}

	return reflect.DeepEqual(expected, actual)
}

func getFloat64(value interface{}) (float64, bool) {
	f64, ok := value.(float64)
	if ok {
		return f64, ok
	}

	f, ok := value.(float32)
	if ok {
		return float64(f), ok
	}

	i64, ok := value.(int64)
	if ok {
		return float64(i64), ok
	}

	i, ok := value.(int)
	if ok {
		return float64(i), ok
	}

	n, ok := value.(json.Number)
	if ok {
		value, err := n.Float64()
		if err == nil {
			return value, true
		}
		return 0, false
	}

	s, ok := value.(string)
	if ok {
		v, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return v, true
		}
		return 0, false
	}

	return 0, false
}

func getInt64(value interface{}) (int64, bool) {
	if i64, ok := value.(int64); ok {
		return i64, ok
	}

	if i, ok := value.(int); ok {
		return int64(i), ok
	}

	if n, ok := value.(json.Number); ok {
		v, err := strconv.ParseInt(n.String(), 10, 64)
		if err == nil {
			return v, true
		}
		return 0, false
	}

	if f64, ok := value.(float64); ok {
		return int64(f64), ok
	}

	if f32, ok := value.(float32); ok {
		return int64(f32), ok
	}

	if s, ok := value.(string); ok {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			return v, true
		}
		return 0, false
	}

	return 0, false
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	r := make(map[string]string, len(m))
	for k, v := range m {
		r[k] = v
	}

	return r
}

// A request body can be read multiple times, it should not be closed until the whole process is completed
func readRequestBody(r *http.Request) io.Reader {
	bodyBuf := bytes.NewBuffer(make([]byte, 0))
	reader := io.TeeReader(r.Body, bodyBuf)
	r.Body = io.NopCloser(bodyBuf)
	return reader
}
