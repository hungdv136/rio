package rio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/types"
	"gopkg.in/yaml.v3"
)

const (
	SchemaTypeJSON SchemaType = "json"
	SchemaTypeYAML SchemaType = "yaml"
)

type SchemaType string

// Template defines setting for response template
type Template struct {
	// Supported json and yaml. Default value is yaml
	ScriptSchemaType SchemaType `json:"script_schema_type" yaml:"script_schema_type"`

	// Script is content of template file. See ResponseScript for the detail structure
	Script string `json:"script" yaml:"script"`
}

// ResponseScript represents for http response script
type ResponseScript struct {
	StatusCode int               `json:"status_code,omitempty" yaml:"status_code"`
	Body       string            `json:"body,omitempty" yaml:"body"`
	Cookies    []Cookie          `json:"cookies,omitempty" yaml:"cookies"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers"`
	Error      *ResponseError    `json:"error,omitempty" yaml:"error"`
}

func (s *ResponseScript) AssignTo(r *Response) {
	if s.StatusCode > 0 {
		r.StatusCode = s.StatusCode
	}

	if len(s.Body) > 0 {
		r.Body = []byte(s.Body)
	}

	r.Cookies = append(r.Cookies, s.Cookies...)

	if s.Error != nil {
		r.Error = s.Error
	}

	for k, v := range s.Headers {
		r.Header[k] = v
	}
}

// TemplateData holds all available data for feeding to template
// Either HTTP or GRPC is supported
type TemplateData struct {
	// Request is the incoming request
	// Which can be accessed from template as {{ .Request.<FielName> }}
	Request *http.Request `json:"request,omitempty" yaml:"request"`

	// Grpc is grpc request
	// Which can be accessed from template as {{ .Grpc.<FielName> }}
	Grpc *GrpcRequest `json:"grpc,omitempty" yaml:"grpc"`

	parsedBody types.Map
}

// JSONBody parses request body (or request parameter in grpc) to json
// Panic if error for more convenient when using with template
func (t *TemplateData) JSONBody() map[string]interface{} {
	if t.Grpc != nil {
		return t.Grpc.InputData
	}

	if t.parsedBody != nil {
		return t.parsedBody
	}

	dataMap := map[string]interface{}{}
	decoder := json.NewDecoder(readRequestBody(t.Request))
	decoder.UseNumber()
	if err := decoder.Decode(&dataMap); err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}

	t.parsedBody = dataMap
	return dataMap
}

// Execute executes the template. Only go-template is supported at the moment
// For supported function in Go template, see http://masterminds.github.io/sprig/
func (t *Template) Execute(ctx context.Context, data *TemplateData) (*ResponseScript, error) {
	script := template.New("Response Template").Funcs(sprig.TxtFuncMap())
	script, err := script.Parse(t.Script)
	if err != nil {
		log.Error(ctx, "cannot parse script", err, t.Script)
		return nil, err
	}

	doc := bytes.Buffer{}
	if err := script.Execute(&doc, data); err != nil {
		log.Error(ctx, "cannot execute template", err)
		return nil, err
	}

	if t.ScriptSchemaType == SchemaTypeJSON {
		return parseResponseScriptFromJSON(ctx, doc.Bytes())
	}

	return parseResponseScriptFromYaml(ctx, doc.Bytes())
}

func parseResponseScriptFromJSON(ctx context.Context, data []byte) (*ResponseScript, error) {
	r := ResponseScript{}
	if err := json.Unmarshal(data, &r); err != nil {
		log.Error(ctx, err, string(data))
		return nil, err
	}

	return &r, nil
}

func parseResponseScriptFromYaml(ctx context.Context, data []byte) (*ResponseScript, error) {
	r := ResponseScript{}
	if err := yaml.Unmarshal(data, &r); err != nil {
		log.Error(ctx, "cannot parse yaml", err, string(data))
		return nil, err
	}

	return &r, nil
}
