package rio

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/util"
)

var matchingFunctions = map[OperatorName]matchingFunc{
	OpEqualTo:       executeEqualToOperator,
	OpContaining:    executeContainingOperator,
	OpNotContaining: executeNotContainingOperator,
	OpRegex:         executeRegexOperator,
	OpStartWith:     executeStartWithOperator,
	OpEndWith:       executeEndWithOperator,
	OpLength:        executeLengthOperator,
	OpEmpty:         executeEmptyOperator,
	OpNotEmpty:      executeNotEmptyOperator,
}

type matchingFunc func(ctx context.Context, op Operator, value interface{}) (bool, error)

// Match compares input value with predefined operator
func Match(ctx context.Context, op Operator, value interface{}) (bool, error) {
	matchFunc, ok := matchingFunctions[op.Name]
	if !ok {
		err := fmt.Errorf("unsupported operator %s", op.Name)
		log.Error(ctx, err)
		return false, err
	}

	return matchFunc(ctx, op, value)
}

func executeEqualToOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	switch opVal := op.Value.(type) {
	case json.Number:
		vFloat64, err := opVal.Float64()
		if err != nil {
			log.Error(ctx, err)
			return false, err
		}

		if val, ok := getFloat64(value); ok {
			return vFloat64 == val, nil
		}

	case float64, float32:
		opVal64, ok := getFloat64(opVal)
		if !ok {
			return false, nil
		}

		if val, ok := getFloat64(value); ok {
			return val == opVal64, nil
		}

	case int, int16, int64, int32, uint, uint16, uint32, uint64:
		opVal64, ok := getInt64(opVal)
		if !ok {
			return false, nil
		}

		if val, ok := getInt64(value); ok {
			return opVal64 == val, nil
		}

	case string:
		if val, ok := value.(string); ok {
			return opVal == val, nil
		}

		return opVal == util.ToString(value), nil
	}

	return reflect.DeepEqual(op.Value, value), nil
}

func executeContainingOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	ok, found := containsElement(value, op.Value)
	if !ok {
		err := fmt.Errorf("unsupported data type - %T", value)
		log.Error(ctx, err)
		return false, err
	}

	return found, nil
}

func executeNotContainingOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	contain, err := executeContainingOperator(ctx, op, value)
	return !contain, err
}

func executeRegexOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	s, ok := op.Value.(string)
	if !ok {
		err := fmt.Errorf("unsupported data type %s - %T", op.String(), value)
		log.Error(ctx, err)
		return false, err
	}

	actualVal, ok := value.(string)
	if !ok {
		err := fmt.Errorf("incompatible string type %s - %T", op.String(), value)
		log.Error(ctx, err)
		return false, err
	}

	r, err := defaultRegexCompiler.compile(ctx, s)
	if err != nil {
		return false, err
	}

	return r.MatchString(actualVal), nil
}

func executeStartWithOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	s, ok := op.Value.(string)
	if !ok {
		err := fmt.Errorf("unsupported data type %s - %T", op.String(), value)
		log.Error(ctx, err)
		return false, err
	}

	actualVal, ok := value.(string)
	if !ok {
		err := fmt.Errorf("incompatible string type %s - %T", op.String(), value)
		log.Error(ctx, err)
		return false, err
	}

	return strings.HasPrefix(actualVal, s), nil
}

func executeEndWithOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	s, ok := op.Value.(string)
	if !ok {
		err := fmt.Errorf("unsupported data type %s - %T", op.String(), value)
		log.Error(ctx, err)
		return false, err
	}

	actualVal, ok := value.(string)
	if !ok {
		err := fmt.Errorf("incompatible string type %s - %T", op.String(), value)
		log.Error(ctx, err)
		return false, err
	}

	return strings.HasSuffix(actualVal, s), nil
}

func executeLengthOperator(ctx context.Context, op Operator, value interface{}) (bool, error) {
	l, ok := getLen(value)
	if !ok {
		err := fmt.Errorf("unsupported data type %T", value)
		log.Error(ctx, err)
		return false, err
	}

	return l == op.Value.(int), nil
}

func executeEmptyOperator(_ context.Context, _ Operator, value interface{}) (bool, error) {
	return isEmpty(value), nil
}

func executeNotEmptyOperator(_ context.Context, _ Operator, value interface{}) (bool, error) {
	return !isEmpty(value), nil
}
