package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/devaryakjha/loadwright/internal/spec"
	"gopkg.in/yaml.v3"
)

type Options struct {
	BaseURL string
}

type Document struct {
	OpenAPI    string                `json:"openapi" yaml:"openapi"`
	Info       Info                  `json:"info" yaml:"info"`
	Servers    []Server              `json:"servers" yaml:"servers"`
	Components Components            `json:"components" yaml:"components"`
	Security   []SecurityRequirement `json:"security" yaml:"security"`
	Paths      map[string]PathItem   `json:"paths" yaml:"paths"`
}

type Info struct {
	Title string `json:"title" yaml:"title"`
}

type Server struct {
	URL string `json:"url" yaml:"url"`
}

type PathItem struct {
	Get     *Operation `json:"get" yaml:"get"`
	Post    *Operation `json:"post" yaml:"post"`
	Put     *Operation `json:"put" yaml:"put"`
	Patch   *Operation `json:"patch" yaml:"patch"`
	Delete  *Operation `json:"delete" yaml:"delete"`
	Head    *Operation `json:"head" yaml:"head"`
	Options *Operation `json:"options" yaml:"options"`
}

type Operation struct {
	OperationID string              `json:"operationId" yaml:"operationId"`
	Summary     string              `json:"summary" yaml:"summary"`
	Parameters  []Parameter         `json:"parameters" yaml:"parameters"`
	RequestBody *RequestBody        `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]Response `json:"responses" yaml:"responses"`
}

type Parameter struct {
	Name    string `json:"name" yaml:"name"`
	In      string `json:"in" yaml:"in"`
	Example any    `json:"example" yaml:"example"`
	Schema  Schema `json:"schema" yaml:"schema"`
}

type RequestBody struct {
	Content map[string]MediaType `json:"content" yaml:"content"`
}

type MediaType struct {
	Example  any                `json:"example" yaml:"example"`
	Examples map[string]Example `json:"examples" yaml:"examples"`
	Schema   Schema             `json:"schema" yaml:"schema"`
}

type Example struct {
	Value any `json:"value" yaml:"value"`
}

type Response struct{}

type Components struct {
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes" yaml:"securitySchemes"`
}

type SecurityRequirement map[string][]string

type SecurityScheme struct {
	Type         string `json:"type" yaml:"type"`
	Scheme       string `json:"scheme" yaml:"scheme"`
	BearerFormat string `json:"bearerFormat" yaml:"bearerFormat"`
	In           string `json:"in" yaml:"in"`
	Name         string `json:"name" yaml:"name"`
}

type Schema struct {
	Type       string            `json:"type" yaml:"type"`
	Format     string            `json:"format" yaml:"format"`
	Example    any               `json:"example" yaml:"example"`
	Default    any               `json:"default" yaml:"default"`
	Enum       []any             `json:"enum" yaml:"enum"`
	Properties map[string]Schema `json:"properties" yaml:"properties"`
	Items      *Schema           `json:"items" yaml:"items"`
}

func ImportFile(path string, options Options) (*spec.Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI file: %w", err)
	}
	var doc Document
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse OpenAPI JSON: %w", err)
		}
	} else if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse OpenAPI YAML: %w", err)
	}
	return Import(doc, options)
}

func Import(doc Document, options Options) (*spec.Spec, error) {
	if !strings.HasPrefix(doc.OpenAPI, "3.") {
		return nil, fmt.Errorf("only OpenAPI 3.x documents are supported")
	}
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("OpenAPI document has no paths")
	}
	target := strings.TrimRight(options.BaseURL, "/")
	if target == "" && len(doc.Servers) > 0 {
		target = strings.TrimRight(doc.Servers[0].URL, "/")
	}
	if target == "" {
		target = "https://example.com"
	}
	title := strings.TrimSpace(doc.Info.Title)
	if title == "" {
		title = "openapi-import"
	}
	errorRate := 1.0
	p95 := 3000.0
	loops := 1
	out := &spec.Spec{
		Name:      toKebab(title),
		Target:    target,
		Variables: map[string]string{},
		Load: spec.Load{
			Users:  1,
			RampUp: spec.Duration{Seconds: 1, Set: true},
			Loops:  &loops,
		},
		Thresholds: spec.Thresholds{
			ErrorRateLT: &errorRate,
			P95MsLT:     &p95,
		},
	}
	out.Auth = authFromSecurity(doc, out.Variables)

	for _, path := range sortedPathKeys(doc.Paths) {
		item := doc.Paths[path]
		for _, endpoint := range item.operations() {
			request := requestFromOperation(endpoint.method, path, endpoint.operation, out.Variables)
			out.Requests = append(out.Requests, request)
		}
	}
	if len(out.Requests) == 0 {
		return nil, fmt.Errorf("OpenAPI document has no supported operations")
	}
	if err := out.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	return out, nil
}

type endpoint struct {
	method    string
	operation *Operation
}

func (p PathItem) operations() []endpoint {
	candidates := []endpoint{
		{method: "GET", operation: p.Get},
		{method: "POST", operation: p.Post},
		{method: "PUT", operation: p.Put},
		{method: "PATCH", operation: p.Patch},
		{method: "DELETE", operation: p.Delete},
		{method: "HEAD", operation: p.Head},
		{method: "OPTIONS", operation: p.Options},
	}
	var out []endpoint
	for _, candidate := range candidates {
		if candidate.operation != nil {
			out = append(out, candidate)
		}
	}
	return out
}

func requestFromOperation(method string, path string, operation *Operation, variables map[string]string) spec.Request {
	name := strings.TrimSpace(operation.OperationID)
	if name == "" {
		name = method + " " + path
	}
	requestPath := path
	query := map[string]string{}
	for _, parameter := range operation.Parameters {
		value := parameterExample(parameter)
		switch parameter.In {
		case "path":
			variableName := toSnake(parameter.Name)
			if _, exists := variables[variableName]; !exists {
				variables[variableName] = value
			}
			requestPath = strings.ReplaceAll(requestPath, "{"+parameter.Name+"}", "{{"+variableName+"}}")
		case "query":
			query[parameter.Name] = value
		}
	}
	request := spec.Request{
		Name:   name,
		Method: method,
		Path:   requestPath,
		Query:  query,
		Expect: spec.Expect{Status: firstSuccessStatus(operation.Responses)},
	}
	if body, ok := requestBodyExample(operation.RequestBody); ok {
		request.Headers = map[string]string{"content-type": "application/json"}
		request.BodyJSON = body
	}
	return request
}

func authFromSecurity(doc Document, variables map[string]string) spec.Auth {
	for _, requirement := range doc.Security {
		for schemeName := range requirement {
			scheme, ok := doc.Components.SecuritySchemes[schemeName]
			if !ok {
				continue
			}
			auth := authFromSecurityScheme(scheme, variables)
			if !auth.IsZero() {
				return auth
			}
		}
	}
	return spec.Auth{}
}

func authFromSecurityScheme(scheme SecurityScheme, variables map[string]string) spec.Auth {
	if !strings.EqualFold(strings.TrimSpace(scheme.Type), "http") {
		return spec.Auth{}
	}
	switch strings.ToLower(strings.TrimSpace(scheme.Scheme)) {
	case "bearer":
		if _, exists := variables["api_token"]; !exists {
			variables["api_token"] = "replace-me"
		}
		return spec.Auth{Type: "bearer", Token: "{{api_token}}"}
	case "basic":
		if _, exists := variables["basic_username"]; !exists {
			variables["basic_username"] = "replace-me"
		}
		if _, exists := variables["basic_password"]; !exists {
			variables["basic_password"] = "replace-me"
		}
		return spec.Auth{Type: "basic", Username: "{{basic_username}}", Password: "{{basic_password}}"}
	default:
		return spec.Auth{}
	}
}

func parameterExample(parameter Parameter) string {
	if parameter.Example != nil {
		return fmt.Sprintf("%v", parameter.Example)
	}
	if parameter.Schema.Example != nil {
		return fmt.Sprintf("%v", parameter.Schema.Example)
	}
	if parameter.Schema.Default != nil {
		return fmt.Sprintf("%v", parameter.Schema.Default)
	}
	if len(parameter.Schema.Enum) > 0 {
		return fmt.Sprintf("%v", parameter.Schema.Enum[0])
	}
	switch parameter.Schema.Type {
	case "integer", "number":
		return "1"
	case "boolean":
		return "true"
	default:
		if parameter.Name == "id" || strings.HasSuffix(strings.ToLower(parameter.Name), "id") {
			return "example-" + toKebab(parameter.Name)
		}
		return "example-" + toKebab(parameter.Name)
	}
}

func requestBodyExample(body *RequestBody) (any, bool) {
	if body == nil || len(body.Content) == 0 {
		return nil, false
	}
	media, ok := body.Content["application/json"]
	if !ok {
		for contentType, candidate := range body.Content {
			if strings.Contains(contentType, "json") {
				media = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, false
	}
	if media.Example != nil {
		return media.Example, true
	}
	for _, example := range media.Examples {
		if example.Value != nil {
			return example.Value, true
		}
	}
	return schemaExample(media.Schema), true
}

func schemaExample(schema Schema) any {
	if schema.Example != nil {
		return schema.Example
	}
	if schema.Default != nil {
		return schema.Default
	}
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}
	switch schema.Type {
	case "object":
		out := map[string]any{}
		for _, key := range sortedSchemaKeys(schema.Properties) {
			out[key] = schemaExample(schema.Properties[key])
		}
		return out
	case "array":
		if schema.Items == nil {
			return []any{}
		}
		return []any{schemaExample(*schema.Items)}
	case "integer":
		return 1
	case "number":
		return 1.0
	case "boolean":
		return true
	default:
		return "example"
	}
}

func firstSuccessStatus(responses map[string]Response) int {
	if len(responses) == 0 {
		return 200
	}
	var statuses []string
	for status := range responses {
		if strings.HasPrefix(status, "2") {
			statuses = append(statuses, status)
		}
	}
	sort.Strings(statuses)
	if len(statuses) == 0 {
		return 200
	}
	status := 200
	_, _ = fmt.Sscanf(statuses[0], "%d", &status)
	return status
}

func sortedPathKeys(paths map[string]PathItem) []string {
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedSchemaKeys(values map[string]Schema) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func toKebab(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	var previous rune
	for _, r := range value {
		isUpper := r >= 'A' && r <= 'Z'
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isUpper || isLower || isDigit {
			if isUpper && previous != 0 && previous != '-' && !lastDash {
				b.WriteRune('-')
			}
			if isUpper {
				r = r + ('a' - 'A')
			}
			b.WriteRune(r)
			lastDash = false
			previous = r
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
		previous = '-'
	}
	return strings.Trim(b.String(), "-")
}

func toSnake(value string) string {
	return strings.ReplaceAll(toKebab(value), "-", "_")
}
