package rio

import (
	"context"
	"fmt"

	"github.com/hungdv136/rio/internal/log"
)

var AllSupportedOperators = []OperatorName{
	OpContaining,
	OpNotContaining,
	OpRegex,
	OpEqualTo,
	OpStartWith,
	OpEndWith,
	OpLength,
	OpEmpty,
	OpNotEmpty,
}

// Defines common operator name
// Remember to add to AllSupportedOperators
const (
	OpContaining    OperatorName = "contains"
	OpNotContaining OperatorName = "not_contains"
	OpRegex         OperatorName = "regex"
	OpEqualTo       OperatorName = "equal_to"
	OpStartWith     OperatorName = "start_with"
	OpEndWith       OperatorName = "end_with"
	OpLength        OperatorName = "length"
	OpEmpty         OperatorName = "empty"
	OpNotEmpty      OperatorName = "not_empty"
)

// OperatorName is alias for operator name
type OperatorName string

// CreateOperator is alias for creating an operator
type CreateOperator func() Operator

// Operator defines operator name and expected value
type Operator struct {
	// OperatorName is the name of operator which is one of the following values
	//  - "contains"
	//  - "not_contains"
	//  - "regex"
	//  - "equal_to"
	//  - "start_with"
	//  - "end_with"
	//  - "length"
	//  - "empty"
	//  - "not_empty"
	Name OperatorName `json:"name" yaml:"name"`

	// Value the expected value, which will be compared with value from incoming request
	Value interface{} `json:"value" yaml:"value"`
}

// String returns string
func (o Operator) String() string {
	return fmt.Sprintf("Operator Name: %s - Type: %T", o.Name, o.Value)
}

func (o Operator) IsValid() bool {
	for _, name := range AllSupportedOperators {
		if o.Name == name {
			return true
		}
	}

	return false
}

// FieldOperator defines operator with field name
type FieldOperator struct {
	// FieldName is header name, cookie name or parameter name
	FieldName string   `json:"field_name" yaml:"field_name"`
	Operator  Operator `json:"operator" yaml:"operator"`
}

// String returns string
func (o FieldOperator) String() string {
	return fmt.Sprintf("Field Name: %s - Operator Name: %s - Type: %T", o.FieldName, o.Operator.Name, o.Operator.Value)
}

// Contains checks actual value should contain given value in parameter
func Contains(v interface{}) CreateOperator {
	return func() Operator {
		return Operator{Name: OpContaining, Value: v}
	}
}

// NotContains checks actual value should not contain given value in parameter
func NotContains(v interface{}) CreateOperator {
	return func() Operator {
		return Operator{Name: OpNotContaining, Value: v}
	}
}

// EqualTo checks actual value should equal to given value in parameter
// Engine will convert actual value to same type with predefined parameter
func EqualTo(v interface{}) CreateOperator {
	return func() Operator {
		return Operator{Name: OpEqualTo, Value: v}
	}
}

// Regex checks actual value should match with given regex in parameter
func Regex(v string) CreateOperator {
	return func() Operator {
		return Operator{Name: OpRegex, Value: v}
	}
}

// StartWith sets the start with matching logic
// This operator is applied for string only
func StartWith(v string) CreateOperator {
	return func() Operator {
		return Operator{Name: OpStartWith, Value: v}
	}
}

// EndWith sets the start with matching logic
// This operator is applied for string only
func EndWith(v string) CreateOperator {
	return func() Operator {
		return Operator{Name: OpEndWith, Value: v}
	}
}

// Empty checks object is empty
func Empty() CreateOperator {
	return func() Operator {
		return Operator{Name: OpEmpty}
	}
}

// NotEmpty checks object is not empty
func NotEmpty() CreateOperator {
	return func() Operator {
		return Operator{Name: OpNotEmpty}
	}
}

// Length check length operator
// Supported data types: map, string and array of map, int, string, float, ...
func Length(v int) CreateOperator {
	return func() Operator {
		return Operator{Name: OpLength, Value: v}
	}
}

// BodyOperator define operator for matching body
type BodyOperator struct {
	// The content type of the request body which is one of the following values
	//  - "application/json"
	//  - "text/xml"
	//  - "text/html"
	//  - "text/plain"
	//  - "multipart/form-data"
	//  - "application/x-www-form-urlencoded"
	ContentType string `json:"content_type" yaml:"content_type"`

	Operator Operator `json:"operator" yaml:"operator"`

	// KeyPath is json or xml path
	// Refer to this document for json path syntax https://goessner.net/articles/JsonPath/
	KeyPath string `json:"key_path" yaml:"key_path"`
}

// CreateBodyOperator is alias function for creating a body operator
type CreateBodyOperator func() BodyOperator

// BodyJSONPath matches request body by the json path
// Refer to this document for json path syntax https://goessner.net/articles/JsonPath/
func BodyJSONPath(jsonPath string, createOperator CreateOperator) CreateBodyOperator {
	return func() BodyOperator {
		return BodyOperator{
			Operator:    createOperator(),
			ContentType: ContentTypeJSON,
			KeyPath:     jsonPath,
		}
	}
}

// MultiPartForm to verify form value in multiple parts request
func MultiPartForm(key string, createOperator CreateOperator) CreateBodyOperator {
	return func() BodyOperator {
		return BodyOperator{
			Operator:    createOperator(),
			ContentType: ContentTypeMultipart,
			KeyPath:     key,
		}
	}
}

// URLEncodedBody to verify form value in url encoded request
func URLEncodedBody(key string, createOperator CreateOperator) CreateBodyOperator {
	return func() BodyOperator {
		return BodyOperator{
			Operator:    createOperator(),
			ContentType: ContentTypeForm,
			KeyPath:     key,
		}
	}
}

func validateOp(ctx context.Context, ops ...Operator) error {
	for _, o := range ops {
		if !o.IsValid() {
			err := fmt.Errorf("unsupported operator name %s", o.Name)
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}

func validateFieldOps(ctx context.Context, ops ...FieldOperator) error {
	for _, o := range ops {
		if err := validateOp(ctx, o.Operator); err != nil {
			return err
		}

		if len(o.FieldName) == 0 {
			err := fmt.Errorf("missing field name %s", o.FieldName)
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}

func validateBodyOps(ctx context.Context, ops ...BodyOperator) error {
	for _, o := range ops {
		if err := validateOp(ctx, o.Operator); err != nil {
			return err
		}

		if len(o.KeyPath) == 0 {
			err := fmt.Errorf("missing key path %s", o.KeyPath)
			log.Error(ctx, err)
			return err
		}

		if len(o.ContentType) == 0 {
			err := fmt.Errorf("missing content type %s", o.ContentType)
			log.Error(ctx, err)
			return err
		}
	}

	return nil
}
