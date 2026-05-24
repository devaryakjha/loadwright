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

// WSMessage describes a single WebSocket message in a multi-message sequence.
type WSMessage struct {
	Send   string    `yaml:"send"`
	Type   string    `yaml:"type,omitempty"`
	Expect *WSExpect `yaml:"expect,omitempty"`
	Delay  Duration  `yaml:"delay,omitempty"`
}

// WSExpect describes the expected response for a single WebSocket message.
type WSExpect struct {
	Contains string   `yaml:"contains,omitempty"`
	Timeout  Duration `yaml:"timeout,omitempty"`
}

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
	Name          string            `yaml:"name"`
	Protocol      string            `yaml:"protocol,omitempty"`
	Method        string            `yaml:"method"`
	Path          string            `yaml:"path"`
	Headers       map[string]string `yaml:"headers,omitempty"`
	Query         map[string]string `yaml:"query,omitempty"`
	Body          any               `yaml:"body,omitempty"`
	BodyJSON      any               `yaml:"body_json,omitempty"`
	BodyText      string            `yaml:"body_text,omitempty"`
	BodyForm      map[string]string `yaml:"body_form,omitempty"`
	BodyMultipart []MultipartPart   `yaml:"body_multipart,omitempty"`
	Auth          Auth              `yaml:"auth,omitempty"`
	Timeout       Duration          `yaml:"timeout,omitempty"`
	Expect        Expect            `yaml:"expect"`
	WebSocket     WebSocket         `yaml:"websocket,omitempty"`
}

type MultipartPart struct {
	Name        string  `yaml:"name"`
	Value       *string `yaml:"value,omitempty"`
	File        string  `yaml:"file,omitempty"`
	ContentType string  `yaml:"content_type,omitempty"`
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

type WebSocket struct {
	URL            string            `yaml:"url,omitempty"`
	Message        string            `yaml:"message,omitempty"`
	ExpectContains string            `yaml:"expect_contains,omitempty"`
	Timeout        Duration          `yaml:"timeout,omitempty"`
	Messages       []WSMessage       `yaml:"messages,omitempty"`
	Subprotocol    string            `yaml:"subprotocol,omitempty"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	CloseTimeout   Duration          `yaml:"close_timeout,omitempty"`
}

func (w WebSocket) IsZero() bool {
	return w.URL == "" && w.Message == "" && w.ExpectContains == "" && !w.Timeout.Set &&
		len(w.Messages) == 0 && w.Subprotocol == "" && len(w.Headers) == 0 && !w.CloseTimeout.Set
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
		validationErrors = append(validationErrors, errors.New("target must be an absolute http, https, ws, or wss URL"))
	} else if parsedTarget.Scheme != "http" && parsedTarget.Scheme != "https" && parsedTarget.Scheme != "ws" && parsedTarget.Scheme != "wss" {
		validationErrors = append(validationErrors, errors.New("target must use http, https, ws, or wss"))
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
		if s.Requests[index].Protocol == "websocket" && s.Requests[index].WebSocket.URL == "" {
			if parsedTarget.Scheme != "ws" && parsedTarget.Scheme != "wss" {
				validationErrors = append(validationErrors, fmt.Errorf("requests[%d].websocket.url is required when target is not ws or wss", index))
			}
		}
		if s.Requests[index].Protocol != "websocket" && (parsedTarget.Scheme == "ws" || parsedTarget.Scheme == "wss") {
			validationErrors = append(validationErrors, fmt.Errorf("requests[%d] is an HTTP request but target uses scheme %q; use an http or https target for HTTP requests", index, parsedTarget.Scheme))
		}
		if !s.Requests[index].Timeout.Set && s.Defaults.Timeout.Set {
			s.Requests[index].Timeout = s.Defaults.Timeout
		}
	}
	return errors.Join(validationErrors...)
}

func (r *Request) NormalizeAndValidate(index int) error {
	r.Protocol = strings.ToLower(strings.TrimSpace(r.Protocol))
	if r.Protocol == "" {
		r.Protocol = "http"
	}
	if r.Protocol == "ws" || r.Protocol == "websocket" {
		r.Protocol = "websocket"
		return r.normalizeAndValidateWebSocket(index)
	}
	if r.Protocol != "http" {
		return fmt.Errorf("requests[%d].protocol is not supported: %s", index, r.Protocol)
	}
	return r.normalizeAndValidateHTTP(index)
}

func (r *Request) normalizeAndValidateHTTP(index int) error {
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

func (r *Request) normalizeAndValidateWebSocket(index int) error {
	r.Method = ""
	r.Path = strings.TrimSpace(r.Path)
	if r.Path != "" && !strings.HasPrefix(r.Path, "/") {
		return fmt.Errorf("requests[%d].path must start with /", index)
	}
	if strings.TrimSpace(r.Name) == "" {
		if r.Path != "" {
			r.Name = "WS " + r.Path
		} else {
			r.Name = "WS request"
		}
	}
	if !r.Auth.IsZero() {
		return fmt.Errorf("requests[%d].auth is not supported for websocket requests", index)
	}
	hasHTTPOnlyBody := r.Body != nil || r.BodyJSON != nil || strings.TrimSpace(r.BodyText) != "" || len(r.BodyForm) > 0 || len(r.BodyMultipart) > 0
	if len(r.Headers) > 0 || len(r.Query) > 0 || hasHTTPOnlyBody || r.Expect.Status > 0 {
		return fmt.Errorf("requests[%d] websocket requests only support websocket.* fields plus optional path and timeout", index)
	}
	r.WebSocket.URL = strings.TrimSpace(r.WebSocket.URL)
	if r.WebSocket.URL != "" {
		parsedURL, err := url.Parse(r.WebSocket.URL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf("requests[%d].websocket.url must be an absolute ws or wss URL", index)
		}
		if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
			return fmt.Errorf("requests[%d].websocket.url must use ws or wss", index)
		}
	}
	if !r.WebSocket.Timeout.Set && r.Timeout.Set {
		r.WebSocket.Timeout = r.Timeout
	}

	// Reject mixing legacy single-message fields with the new messages list.
	hasLegacy := r.WebSocket.Message != "" || r.WebSocket.ExpectContains != ""
	hasMessages := len(r.WebSocket.Messages) > 0
	if hasLegacy && hasMessages {
		return fmt.Errorf("requests[%d].websocket cannot mix legacy message/expect_contains with messages[]", index)
	}

	// Normalize legacy fields into a single-element Messages slice.
	if hasLegacy {
		msg := WSMessage{
			Send: r.WebSocket.Message,
			Type: "text",
		}
		if r.WebSocket.ExpectContains != "" {
			msg.Expect = &WSExpect{Contains: r.WebSocket.ExpectContains}
		}
		r.WebSocket.Messages = []WSMessage{msg}
	}

	// Validate each message in the sequence.
	for i := range r.WebSocket.Messages {
		m := &r.WebSocket.Messages[i]
		m.Send = strings.TrimSpace(m.Send)
		if m.Send == "" {
			return fmt.Errorf("requests[%d].websocket.messages[%d].send is required", index, i)
		}
		m.Type = strings.ToLower(strings.TrimSpace(m.Type))
		if m.Type == "" {
			m.Type = "text"
		}
		if m.Type != "text" && m.Type != "binary" {
			return fmt.Errorf("requests[%d].websocket.messages[%d].type must be text or binary", index, i)
		}
		if m.Type == "binary" {
			if _, err := base64.StdEncoding.DecodeString(m.Send); err != nil {
				return fmt.Errorf("requests[%d].websocket.messages[%d].send must be valid base64 for binary type", index, i)
			}
		}
		if m.Expect != nil && !m.Expect.Timeout.Set && r.WebSocket.Timeout.Set {
			m.Expect.Timeout = r.WebSocket.Timeout
		}
	}

	// Subprotocol is optional free-text.
	r.WebSocket.Subprotocol = strings.TrimSpace(r.WebSocket.Subprotocol)

	// Close timeout defaults to connection timeout.
	if !r.WebSocket.CloseTimeout.Set && r.WebSocket.Timeout.Set {
		r.WebSocket.CloseTimeout = r.WebSocket.Timeout
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
	if len(r.BodyMultipart) > 0 {
		fields = append(fields, "body_multipart")
		for partIndex := range r.BodyMultipart {
			part := &r.BodyMultipart[partIndex]
			part.Name = strings.TrimSpace(part.Name)
			part.File = strings.TrimSpace(part.File)
			part.ContentType = strings.TrimSpace(part.ContentType)
			if part.Name == "" {
				return fmt.Errorf("requests[%d].body_multipart[%d].name is required", index, partIndex)
			}
			if part.Value == nil && part.File == "" {
				return fmt.Errorf("requests[%d].body_multipart[%d] must set value or file", index, partIndex)
			}
			if part.Value != nil && part.File != "" {
				return fmt.Errorf("requests[%d].body_multipart[%d] must set only one of value or file", index, partIndex)
			}
			if part.Value != nil && part.ContentType != "" {
				return fmt.Errorf("requests[%d].body_multipart[%d].content_type is only supported for file parts", index, partIndex)
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
		if s.Requests[index].Protocol != "http" {
			continue
		}
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
	out.BodyMultipart, err = renderMultipartParts(request.BodyMultipart, vars)
	if err != nil {
		return Request{}, fmt.Errorf("requests[%d].body_multipart: %w", index, err)
	}
	out.Auth, err = renderAuth(request.Auth, vars, fmt.Sprintf("requests[%d].auth", index))
	if err != nil {
		return Request{}, err
	}
	out.WebSocket, err = renderWebSocket(request.WebSocket, vars, index)
	if err != nil {
		return Request{}, err
	}
	return out, nil
}

func renderMultipartParts(parts []MultipartPart, vars map[string]string) ([]MultipartPart, error) {
	if len(parts) == 0 {
		return nil, nil
	}
	out := make([]MultipartPart, len(parts))
	for index, part := range parts {
		rendered := part
		var err error
		if rendered.Name, err = renderString(part.Name, vars); err != nil {
			return nil, fmt.Errorf("[%d].name: %w", index, err)
		}
		if part.Value != nil {
			value, err := renderString(*part.Value, vars)
			if err != nil {
				return nil, fmt.Errorf("[%d].value: %w", index, err)
			}
			rendered.Value = &value
		}
		if rendered.File, err = renderString(part.File, vars); err != nil {
			return nil, fmt.Errorf("[%d].file: %w", index, err)
		}
		if rendered.ContentType, err = renderString(part.ContentType, vars); err != nil {
			return nil, fmt.Errorf("[%d].content_type: %w", index, err)
		}
		out[index] = rendered
	}
	return out, nil
}

func renderWebSocket(value WebSocket, vars map[string]string, index int) (WebSocket, error) {
	var err error
	out := value
	if out.URL, err = renderString(value.URL, vars); err != nil {
		return WebSocket{}, fmt.Errorf("requests[%d].websocket.url: %w", index, err)
	}
	if out.Message, err = renderString(value.Message, vars); err != nil {
		return WebSocket{}, fmt.Errorf("requests[%d].websocket.message: %w", index, err)
	}
	if out.ExpectContains, err = renderString(value.ExpectContains, vars); err != nil {
		return WebSocket{}, fmt.Errorf("requests[%d].websocket.expect_contains: %w", index, err)
	}
	if out.Subprotocol, err = renderString(value.Subprotocol, vars); err != nil {
		return WebSocket{}, fmt.Errorf("requests[%d].websocket.subprotocol: %w", index, err)
	}
	if len(value.Headers) > 0 {
		out.Headers, err = renderStringMap(value.Headers, vars)
		if err != nil {
			return WebSocket{}, fmt.Errorf("requests[%d].websocket.headers: %w", index, err)
		}
	}
	if len(value.Messages) > 0 {
		out.Messages = make([]WSMessage, len(value.Messages))
		for i, msg := range value.Messages {
			rendered := msg
			if rendered.Send, err = renderString(msg.Send, vars); err != nil {
				return WebSocket{}, fmt.Errorf("requests[%d].websocket.messages[%d].send: %w", index, i, err)
			}
			if msg.Expect != nil {
				expect := *msg.Expect
				if expect.Contains, err = renderString(msg.Expect.Contains, vars); err != nil {
					return WebSocket{}, fmt.Errorf("requests[%d].websocket.messages[%d].expect.contains: %w", index, i, err)
				}
				rendered.Expect = &expect
			}
			out.Messages[i] = rendered
		}
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
