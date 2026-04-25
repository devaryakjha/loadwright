package spec

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Duration struct {
	Seconds int
	Set     bool
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var raw any
	if err := value.Decode(&raw); err != nil {
		return err
	}
	seconds, err := ParseDuration(raw)
	if err != nil {
		return err
	}
	d.Seconds = seconds
	d.Set = true
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	if !d.Set {
		return nil, nil
	}
	return fmt.Sprintf("%ds", d.Seconds), nil
}

func (d Duration) IsZero() bool {
	return !d.Set
}

func ParseDuration(raw any) (int, error) {
	switch value := raw.(type) {
	case int:
		if value <= 0 {
			return 0, errors.New("duration must be greater than 0")
		}
		return value, nil
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			return 0, errors.New("duration cannot be empty")
		}
		multiplier := 1
		number := trimmed
		switch trimmed[len(trimmed)-1] {
		case 's':
			number = trimmed[:len(trimmed)-1]
		case 'm':
			number = trimmed[:len(trimmed)-1]
			multiplier = 60
		case 'h':
			number = trimmed[:len(trimmed)-1]
			multiplier = 3600
		}
		parsed, err := strconv.Atoi(number)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("duration must look like 30s, 5m, 1h, or an integer second value")
		}
		return parsed * multiplier, nil
	default:
		return 0, fmt.Errorf("duration must be a string or integer")
	}
}

type Spec struct {
	Name       string             `yaml:"name"`
	Target     string             `yaml:"target"`
	Variables  map[string]string  `yaml:"variables,omitempty"`
	Defaults   Defaults           `yaml:"defaults,omitempty"`
	Auth       Auth               `yaml:"auth,omitempty"`
	Data       map[string]DataSet `yaml:"data,omitempty"`
	Load       Load               `yaml:"load"`
	Requests   []Request          `yaml:"requests"`
	Thresholds Thresholds         `yaml:"thresholds,omitempty"`
}

type Defaults struct {
	Timeout Duration `yaml:"timeout,omitempty"`
}

func (d Defaults) IsZero() bool {
	return d.Timeout.IsZero()
}

type Load struct {
	Users    int      `yaml:"users"`
	RampUp   Duration `yaml:"ramp_up,omitempty"`
	Duration Duration `yaml:"duration,omitempty"`
	Loops    *int     `yaml:"loops,omitempty"`
}

type DataSet struct {
	File       string   `yaml:"file"`
	Variables  []string `yaml:"variables,omitempty"`
	Recycle    *bool    `yaml:"recycle,omitempty"`
	StopThread *bool    `yaml:"stop_thread,omitempty"`
	Sharing    string   `yaml:"sharing,omitempty"`
}

type Request struct {
	Name     string            `yaml:"name"`
	Method   string            `yaml:"method"`
	Path     string            `yaml:"path"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Query    map[string]string `yaml:"query,omitempty"`
	Body     any               `yaml:"body,omitempty"`
	BodyJSON any               `yaml:"body_json,omitempty"`
	BodyText string            `yaml:"body_text,omitempty"`
	BodyForm map[string]string `yaml:"body_form,omitempty"`
	Auth     Auth              `yaml:"auth,omitempty"`
	Timeout  Duration          `yaml:"timeout,omitempty"`
	Expect   Expect            `yaml:"expect"`
}

type Auth struct {
	Type     string `yaml:"type,omitempty"`
	Token    string `yaml:"token,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func (a Auth) IsZero() bool {
	return a.Type == "" && a.Token == "" && a.Username == "" && a.Password == ""
}

type Expect struct {
	Status int `yaml:"status"`
}

type Thresholds struct {
	ErrorRateLT *float64 `yaml:"error_rate_lt,omitempty"`
	P95MsLT     *float64 `yaml:"p95_ms_lt,omitempty"`
	AvgMsLT     *float64 `yaml:"avg_ms_lt,omitempty"`
}

func LoadFile(path string) (*Spec, error) {
	parsed, err := LoadFileUnresolved(path)
	if err != nil {
		return nil, err
	}
	if err := parsed.NormalizeAndValidate(WithBaseDir(filepath.Dir(path))); err != nil {
		return nil, err
	}
	return parsed, nil
}

func LoadFileUnresolved(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	var parsed Spec
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &parsed, nil
}

type NormalizeOption func(*normalizeOptions)

type normalizeOptions struct {
	baseDir string
}

func WithBaseDir(path string) NormalizeOption {
	return func(options *normalizeOptions) {
		options.baseDir = path
	}
}

func (s *Spec) NormalizeAndValidate(opts ...NormalizeOption) error {
	options := normalizeOptions{baseDir: "."}
	for _, opt := range opts {
		opt(&options)
	}
	var validationErrors []error
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		validationErrors = append(validationErrors, errors.New("name is required"))
	}
	s.Target = strings.TrimRight(strings.TrimSpace(s.Target), "/")
	parsedTarget, err := url.Parse(s.Target)
	if err != nil || parsedTarget.Scheme == "" || parsedTarget.Host == "" {
		validationErrors = append(validationErrors, errors.New("target must be an absolute http or https URL"))
	} else if parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https" {
		validationErrors = append(validationErrors, errors.New("target must use http or https"))
	}
	if s.Variables == nil {
		s.Variables = map[string]string{}
	}
	if err := s.Auth.NormalizeAndValidate("auth"); err != nil {
		validationErrors = append(validationErrors, err)
	}
	if s.Data == nil {
		s.Data = map[string]DataSet{}
	}
	for name, dataSet := range s.Data {
		if err := dataSet.NormalizeAndValidate(name, options.baseDir); err != nil {
			validationErrors = append(validationErrors, err)
			continue
		}
		s.Data[name] = dataSet
	}
	if s.Load.Users == 0 {
		s.Load.Users = 1
	}
	if s.Load.Users < 0 {
		validationErrors = append(validationErrors, errors.New("load.users must be greater than 0"))
	}
	if !s.Load.RampUp.Set {
		s.Load.RampUp = Duration{Seconds: 1, Set: true}
	}
	if s.Load.Loops == nil && !s.Load.Duration.Set {
		defaultLoops := 1
		s.Load.Loops = &defaultLoops
	}
	if s.Load.Loops != nil && *s.Load.Loops <= 0 {
		validationErrors = append(validationErrors, errors.New("load.loops must be greater than 0"))
	}
	if len(s.Requests) == 0 {
		validationErrors = append(validationErrors, errors.New("requests must contain at least one request"))
	}
	for index := range s.Requests {
		if err := s.Requests[index].NormalizeAndValidate(index); err != nil {
			validationErrors = append(validationErrors, err)
			continue
		}
		if !s.Requests[index].Timeout.Set && s.Defaults.Timeout.Set {
			s.Requests[index].Timeout = s.Defaults.Timeout
		}
	}
	return errors.Join(validationErrors...)
}

func (r *Request) NormalizeAndValidate(index int) error {
	r.Method = strings.ToUpper(strings.TrimSpace(r.Method))
	if r.Method == "" {
		r.Method = "GET"
	}
	switch r.Method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		return fmt.Errorf("requests[%d].method is not supported: %s", index, r.Method)
	}
	r.Path = strings.TrimSpace(r.Path)
	if !strings.HasPrefix(r.Path, "/") {
		return fmt.Errorf("requests[%d].path must start with /", index)
	}
	if strings.TrimSpace(r.Name) == "" {
		r.Name = r.Method + " " + r.Path
	}
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	if r.Query == nil {
		r.Query = map[string]string{}
	}
	if r.BodyForm == nil {
		r.BodyForm = map[string]string{}
	}
	if err := r.validateBodyFields(index); err != nil {
		return err
	}
	if err := r.Auth.NormalizeAndValidate(fmt.Sprintf("requests[%d].auth", index)); err != nil {
		return err
	}
	return nil
}

func (r *Request) validateBodyFields(index int) error {
	var fields []string
	if r.Body != nil {
		fields = append(fields, "body")
	}
	if r.BodyJSON != nil {
		fields = append(fields, "body_json")
	}
	if strings.TrimSpace(r.BodyText) != "" {
		fields = append(fields, "body_text")
	}
	if len(r.BodyForm) > 0 {
		fields = append(fields, "body_form")
		for key := range r.BodyForm {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf("requests[%d].body_form contains an empty field name", index)
			}
		}
	}
	if len(fields) > 1 {
		return fmt.Errorf("requests[%d] must set only one body field: %s", index, strings.Join(fields, ", "))
	}
	return nil
}

func (d *DataSet) NormalizeAndValidate(name string, baseDir string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("data source name is required")
	}
	d.File = strings.TrimSpace(d.File)
	if d.File == "" {
		return fmt.Errorf("data.%s.file is required", name)
	}
	if d.Sharing == "" {
		d.Sharing = "all"
	}
	switch d.Sharing {
	case "all", "thread", "group":
	default:
		return fmt.Errorf("data.%s.sharing must be one of: all, thread, group", name)
	}
	if len(d.Variables) == 0 {
		headers, err := ReadCSVHeader(filepath.Join(baseDir, d.File))
		if err != nil {
			return fmt.Errorf("data.%s.file: %w", name, err)
		}
		d.Variables = headers
	}
	for index, variable := range d.Variables {
		variable = strings.TrimSpace(variable)
		if variable == "" {
			return fmt.Errorf("data.%s.variables[%d] is empty", name, index)
		}
		d.Variables[index] = variable
	}
	return nil
}

func (a *Auth) NormalizeAndValidate(path string) error {
	a.Type = strings.ToLower(strings.TrimSpace(a.Type))
	if a.Type == "" {
		return nil
	}
	switch a.Type {
	case "bearer":
		if strings.TrimSpace(a.Token) == "" {
			return fmt.Errorf("%s.token is required for bearer auth", path)
		}
	case "basic":
		if strings.TrimSpace(a.Username) == "" {
			return fmt.Errorf("%s.username is required for basic auth", path)
		}
	default:
		return fmt.Errorf("%s.type is not supported: %s", path, a.Type)
	}
	return nil
}

func (s *Spec) Resolve(env map[string]string, opts ...NormalizeOption) (*Spec, error) {
	if env == nil {
		env = map[string]string{}
	}
	resolved := *s
	resolved.Variables = copyMap(s.Variables)
	for key, value := range resolved.Variables {
		rendered, err := renderEnvRefs(value, env)
		if err != nil {
			return nil, fmt.Errorf("variables.%s: %w", key, err)
		}
		resolved.Variables[key] = rendered
	}

	vars := copyMap(env)
	for key, value := range resolved.Variables {
		vars[key] = value
	}

	var err error
	if resolved.Name, err = renderString(s.Name, vars); err != nil {
		return nil, fmt.Errorf("name: %w", err)
	}
	if resolved.Target, err = renderString(s.Target, vars); err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}
	if resolved.Auth, err = renderAuth(s.Auth, vars, "auth"); err != nil {
		return nil, err
	}
	resolved.Requests = make([]Request, len(s.Requests))
	for index, request := range s.Requests {
		resolvedRequest, err := renderRequest(request, vars, index)
		if err != nil {
			return nil, err
		}
		resolved.Requests[index] = resolvedRequest
	}
	if err := resolved.NormalizeAndValidate(opts...); err != nil {
		return nil, err
	}
	resolved.applyAuthHeaders()
	return &resolved, nil
}

func (s *Spec) applyAuthHeaders() {
	for index := range s.Requests {
		auth := s.Auth
		if s.Requests[index].Auth.Type != "" {
			auth = s.Requests[index].Auth
		}
		if auth.Type == "" || hasHeader(s.Requests[index].Headers, "authorization") {
			continue
		}
		switch auth.Type {
		case "bearer":
			s.Requests[index].Headers["Authorization"] = "Bearer " + auth.Token
		case "basic":
			raw := auth.Username + ":" + auth.Password
			s.Requests[index].Headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
		}
	}
}

func renderRequest(request Request, vars map[string]string, index int) (Request, error) {
	var err error
	out := request
	if out.Name, err = renderString(request.Name, vars); err != nil {
		return Request{}, fmt.Errorf("requests[%d].name: %w", index, err)
	}
	if out.Path, err = renderString(request.Path, vars); err != nil {
		return Request{}, fmt.Errorf("requests[%d].path: %w", index, err)
	}
	out.Headers, err = renderStringMap(request.Headers, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].headers: %w", index, err)
	}
	out.Query, err = renderStringMap(request.Query, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].query: %w", index, err)
	}
	out.Body, err = renderAny(request.Body, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].body: %w", index, err)
	}
	out.BodyJSON, err = renderAny(request.BodyJSON, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].body_json: %w", index, err)
	}
	out.BodyText, err = renderString(request.BodyText, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].body_text: %w", index, err)
	}
	out.BodyForm, err = renderStringMap(request.BodyForm, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].body_form: %w", index, err)
	}
	out.Auth, err = renderAuth(request.Auth, vars, fmt.Sprintf("requests[%d].auth", index))
	if err != nil {
		return Request{}, err
	}
	return out, nil
}

func renderAuth(auth Auth, vars map[string]string, path string) (Auth, error) {
	var err error
	out := auth
	if out.Token, err = renderString(auth.Token, vars); err != nil {
		return Auth{}, fmt.Errorf("%s.token: %w", path, err)
	}
	if out.Username, err = renderString(auth.Username, vars); err != nil {
		return Auth{}, fmt.Errorf("%s.username: %w", path, err)
	}
	if out.Password, err = renderString(auth.Password, vars); err != nil {
		return Auth{}, fmt.Errorf("%s.password: %w", path, err)
	}
	return out, nil
}

var templatePattern = regexp.MustCompile(`\{\{[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*\}\}`)
var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func renderString(value string, vars map[string]string) (string, error) {
	rendered := value
	var missing string
	rendered = templatePattern.ReplaceAllStringFunc(rendered, func(match string) string {
		parts := templatePattern.FindStringSubmatch(match)
		key := parts[1]
		value, ok := vars[key]
		if !ok {
			missing = key
			return match
		}
		return value
	})
	if missing != "" {
		return "", fmt.Errorf("missing variable %q", missing)
	}
	return rendered, nil
}

func renderEnvRefs(value string, env map[string]string) (string, error) {
	var missing string
	rendered := envPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := envPattern.FindStringSubmatch(match)
		key := parts[1]
		value, ok := env[key]
		if !ok {
			value, ok = os.LookupEnv(key)
		}
		if !ok {
			missing = key
			return match
		}
		return value
	})
	if missing != "" {
		return "", fmt.Errorf("missing environment value %q", missing)
	}
	return rendered, nil
}

func renderStringMap(values map[string]string, vars map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range values {
		renderedKey, err := renderString(key, vars)
		if err != nil {
			return nil, err
		}
		renderedValue, err := renderString(value, vars)
		if err != nil {
			return nil, err
		}
		out[renderedKey] = renderedValue
	}
	return out, nil
}

func renderAny(value any, vars map[string]string) (any, error) {
	switch typed := value.(type) {
	case string:
		return renderString(typed, vars)
	case map[string]any:
		out := map[string]any{}
		for key, item := range typed {
			renderedKey, err := renderString(key, vars)
			if err != nil {
				return nil, err
			}
			renderedValue, err := renderAny(item, vars)
			if err != nil {
				return nil, err
			}
			out[renderedKey] = renderedValue
		}
		return out, nil
	case map[any]any:
		out := map[string]any{}
		for key, item := range typed {
			renderedKey, err := renderString(fmt.Sprintf("%v", key), vars)
			if err != nil {
				return nil, err
			}
			renderedValue, err := renderAny(item, vars)
			if err != nil {
				return nil, err
			}
			out[renderedKey] = renderedValue
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			renderedValue, err := renderAny(item, vars)
			if err != nil {
				return nil, err
			}
			out[index] = renderedValue
		}
		return out, nil
	default:
		return value, nil
	}
}

func copyMap(values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		out[key] = value
	}
	return out
}

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}
