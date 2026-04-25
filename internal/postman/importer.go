package postman

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/devaryakjha/loadwright/internal/spec"
)

type Options struct {
	BaseURL string
}

type Result struct {
	Spec     *spec.Spec
	Warnings []string
}

type Collection struct {
	Info      Info       `json:"info"`
	Variables []Variable `json:"variable"`
	Items     []Item     `json:"item"`
	Auth      *Auth      `json:"auth"`
}

type Info struct {
	Name string `json:"name"`
}

type Variable struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type Item struct {
	Name    string  `json:"name"`
	Request Request `json:"request"`
	Items   []Item  `json:"item"`
}

type Request struct {
	Set     bool
	Method  string   `json:"method"`
	Headers []Header `json:"header"`
	URL     URL      `json:"url"`
	Body    Body     `json:"body"`
	Auth    *Auth    `json:"auth"`
}

func (r *Request) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		r.Set = true
		r.Method = "GET"
		r.URL = URL{Raw: raw}
		return nil
	}
	type request Request
	var parsed request
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	*r = Request(parsed)
	r.Set = true
	return nil
}

type Header struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Disabled bool   `json:"disabled"`
}

type URL struct {
	Raw      string       `json:"raw"`
	Protocol string       `json:"protocol"`
	Host     StringList   `json:"host"`
	Path     StringList   `json:"path"`
	Query    []QueryParam `json:"query"`
}

func (u *URL) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		u.Raw = raw
		return nil
	}
	type urlAlias URL
	var parsed urlAlias
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	*u = URL(parsed)
	return nil
}

type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	var parts []string
	if err := json.Unmarshal(data, &parts); err == nil {
		*s = parts
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single == "" {
			*s = nil
		} else {
			*s = []string{single}
		}
		return nil
	}
	return fmt.Errorf("expected string or string list")
}

type QueryParam struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Disabled bool   `json:"disabled"`
}

type Body struct {
	Mode    string      `json:"mode"`
	Raw     string      `json:"raw"`
	Options BodyOptions `json:"options"`
}

type BodyOptions struct {
	Raw RawBodyOptions `json:"raw"`
}

type RawBodyOptions struct {
	Language string `json:"language"`
}

type Auth struct {
	Type   string     `json:"type"`
	Bearer []KeyValue `json:"bearer"`
	Basic  []KeyValue `json:"basic"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func ImportFile(path string, options Options) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read Postman collection: %w", err)
	}
	var collection Collection
	if err := json.Unmarshal(data, &collection); err != nil {
		return Result{}, fmt.Errorf("parse Postman collection JSON: %w", err)
	}
	return Import(collection, options)
}

func Import(collection Collection, options Options) (Result, error) {
	name := strings.TrimSpace(collection.Info.Name)
	if name == "" {
		name = "postman-import"
	}
	loops := 1
	errorRate := 1.0
	p95 := 3000.0
	out := &spec.Spec{
		Name:      toKebab(name),
		Target:    strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		Variables: variablesFromCollection(collection.Variables),
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

	var warnings []string
	if collection.Auth != nil {
		auth, warning := authFromPostman(collection.Auth)
		if warning != "" {
			warnings = append(warnings, warning)
		}
		out.Auth = auth
	}

	var firstTarget string
	collectRequests(collection.Items, nil, collection.Auth, out, &firstTarget, &warnings)
	if out.Target == "" {
		out.Target = firstTarget
	}
	if out.Target == "" {
		out.Target = "{{base_url}}"
		if _, exists := out.Variables["base_url"]; !exists {
			out.Variables["base_url"] = "https://example.com"
		}
		warnings = append(warnings, "no request URL target found; using {{base_url}} with https://example.com placeholder")
	}
	ensureTargetVariables(out.Target, out.Variables)
	if len(out.Requests) == 0 {
		return Result{}, fmt.Errorf("Postman collection has no supported requests")
	}
	if _, err := out.Resolve(nil); err != nil {
		return Result{}, err
	}
	return Result{Spec: out, Warnings: dedupeWarnings(warnings)}, nil
}

func collectRequests(items []Item, folders []string, collectionAuth *Auth, out *spec.Spec, firstTarget *string, warnings *[]string) {
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		nextFolders := folders
		if len(item.Items) > 0 {
			if name != "" {
				nextFolders = append(append([]string{}, folders...), name)
			}
			collectRequests(item.Items, nextFolders, collectionAuth, out, firstTarget, warnings)
			continue
		}
		if !item.Request.Set {
			continue
		}
		request, target, requestWarnings, ok := requestFromPostman(item, folders, collectionAuth)
		*warnings = append(*warnings, requestWarnings...)
		if !ok {
			continue
		}
		if *firstTarget == "" && target != "" {
			*firstTarget = target
		}
		if *firstTarget != "" && target != "" && target != *firstTarget {
			*warnings = append(*warnings, fmt.Sprintf("%s uses target %s; imported path will run against %s", request.Name, target, *firstTarget))
		}
		out.Requests = append(out.Requests, request)
	}
}

func requestFromPostman(item Item, folders []string, collectionAuth *Auth) (spec.Request, string, []string, bool) {
	var warnings []string
	method := strings.ToUpper(strings.TrimSpace(item.Request.Method))
	if method == "" {
		method = "GET"
	}
	if !supportedMethod(method) {
		name := requestName(item.Name, folders, method, "")
		return spec.Request{}, "", []string{fmt.Sprintf("%s uses unsupported method %s and was skipped", name, method)}, false
	}
	target, requestPath, query := splitURL(item.Request.URL)
	if requestPath == "" {
		requestPath = "/"
	}
	headers := headersFromPostman(item.Request.Headers)
	body, bodyWarnings := bodyFromPostman(item.Request.Body, headers)
	requestName := requestName(item.Name, folders, method, requestPath)
	for _, warning := range bodyWarnings {
		warnings = append(warnings, fmt.Sprintf("%s: %s", requestName, warning))
	}

	authSource := item.Request.Auth
	if authSource == nil {
		authSource = collectionAuth
	}
	requestAuth := spec.Auth{}
	if authSource != nil && authSource != collectionAuth {
		auth, warning := authFromPostman(authSource)
		requestAuth = auth
		if warning != "" {
			warnings = append(warnings, fmt.Sprintf("%s: %s", requestName, warning))
		}
	}

	return spec.Request{
		Name:    requestName,
		Method:  method,
		Path:    requestPath,
		Headers: headers,
		Query:   query,
		Body:    body,
		Auth:    requestAuth,
		Expect:  spec.Expect{Status: 200},
	}, target, warnings, true
}

func splitURL(postmanURL URL) (target string, requestPath string, query map[string]string) {
	target, requestPath, query = splitRawURL(postmanURL.Raw)
	if len(postmanURL.Host) > 0 {
		host := strings.Join(postmanURL.Host, ".")
		protocol := strings.TrimSpace(postmanURL.Protocol)
		if protocol == "" && len(postmanURL.Host) == 1 && isVariableRef(host) {
			target = host
		} else {
			if protocol == "" {
				protocol = "https"
			}
			target = protocol + "://" + host
		}
	}
	if len(postmanURL.Path) > 0 {
		requestPath = "/" + strings.Join(postmanURL.Path, "/")
	}
	if query == nil {
		query = map[string]string{}
	}
	for _, param := range postmanURL.Query {
		if param.Disabled || strings.TrimSpace(param.Key) == "" {
			continue
		}
		query[param.Key] = stringify(param.Value)
	}
	if requestPath == "" {
		requestPath = "/"
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}
	return strings.TrimRight(target, "/"), requestPath, query
}

func isVariableRef(value string) bool {
	return postmanVariablePattern.MatchString(value) && postmanVariablePattern.ReplaceAllString(value, "") == ""
}

func splitRawURL(raw string) (target string, requestPath string, query map[string]string) {
	raw = strings.TrimSpace(raw)
	query = map[string]string{}
	if raw == "" {
		return "", "", query
	}
	if strings.Contains(raw, "://") {
		scheme, rest, _ := strings.Cut(raw, "://")
		host, pathQuery := splitHostPath(rest)
		target = scheme + "://" + host
		requestPath, query = splitPathQuery(pathQuery)
		return strings.TrimRight(target, "/"), requestPath, query
	}
	if strings.HasPrefix(raw, "{{") {
		end := strings.Index(raw, "}}")
		if end >= 0 {
			target = raw[:end+2]
			requestPath, query = splitPathQuery(raw[end+2:])
			return target, requestPath, query
		}
	}
	if strings.HasPrefix(raw, "/") {
		requestPath, query = splitPathQuery(raw)
		return "", requestPath, query
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		target = parsed.Scheme + "://" + parsed.Host
		requestPath = parsed.EscapedPath()
		for key, values := range parsed.Query() {
			if len(values) > 0 {
				query[key] = values[0]
			}
		}
		return strings.TrimRight(target, "/"), requestPath, query
	}
	return "", "/" + strings.TrimLeft(raw, "/"), query
}

func splitHostPath(rest string) (host string, pathQuery string) {
	index := strings.IndexAny(rest, "/?")
	if index < 0 {
		return rest, ""
	}
	if rest[index] == '?' {
		return rest[:index], "/" + rest[index:]
	}
	return rest[:index], rest[index:]
}

func splitPathQuery(pathQuery string) (string, map[string]string) {
	query := map[string]string{}
	if pathQuery == "" {
		return "/", query
	}
	if strings.HasPrefix(pathQuery, "?") {
		pathQuery = "/" + pathQuery
	}
	requestPath, rawQuery, hasQuery := strings.Cut(pathQuery, "?")
	if requestPath == "" {
		requestPath = "/"
	}
	if hasQuery {
		values, err := url.ParseQuery(rawQuery)
		if err == nil {
			for key, items := range values {
				if len(items) > 0 {
					query[key] = items[0]
				}
			}
		}
	}
	cleaned := path.Clean("/" + strings.TrimLeft(requestPath, "/"))
	if cleaned == "." {
		cleaned = "/"
	}
	return cleaned, query
}

func headersFromPostman(headers []Header) map[string]string {
	out := map[string]string{}
	for _, header := range headers {
		key := strings.TrimSpace(header.Key)
		if header.Disabled || key == "" {
			continue
		}
		out[key] = stringify(header.Value)
	}
	return out
}

func bodyFromPostman(body Body, headers map[string]string) (any, []string) {
	mode := strings.ToLower(strings.TrimSpace(body.Mode))
	switch mode {
	case "":
		return nil, nil
	case "raw":
		raw := strings.TrimSpace(body.Raw)
		if raw == "" {
			return nil, nil
		}
		if shouldParseJSON(raw, body, headers) {
			var parsed any
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				if !hasHeader(headers, "content-type") {
					headers["Content-Type"] = "application/json"
				}
				return parsed, nil
			}
			return raw, []string{"raw body looked like JSON but could not be parsed; imported as string"}
		}
		return raw, nil
	default:
		return nil, []string{fmt.Sprintf("request body mode %q is not imported yet", mode)}
	}
}

func shouldParseJSON(raw string, body Body, headers map[string]string) bool {
	if strings.EqualFold(body.Options.Raw.Language, "json") {
		return true
	}
	for key, value := range headers {
		if strings.EqualFold(key, "content-type") && strings.Contains(strings.ToLower(value), "json") {
			return true
		}
	}
	return strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[")
}

func authFromPostman(auth *Auth) (spec.Auth, string) {
	if auth == nil {
		return spec.Auth{}, ""
	}
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "":
		return spec.Auth{}, ""
	case "bearer":
		token := keyValue(auth.Bearer, "token")
		if token == "" {
			return spec.Auth{}, "bearer auth is missing token and was skipped"
		}
		return spec.Auth{Type: "bearer", Token: token}, ""
	case "basic":
		username := keyValue(auth.Basic, "username")
		password := keyValue(auth.Basic, "password")
		if username == "" {
			return spec.Auth{}, "basic auth is missing username and was skipped"
		}
		return spec.Auth{Type: "basic", Username: username, Password: password}, ""
	default:
		return spec.Auth{}, fmt.Sprintf("auth type %q is not imported yet", auth.Type)
	}
}

func keyValue(values []KeyValue, key string) string {
	for _, value := range values {
		if strings.EqualFold(value.Key, key) {
			return stringify(value.Value)
		}
	}
	return ""
}

func variablesFromCollection(values []Variable) map[string]string {
	out := map[string]string{}
	for _, variable := range values {
		key := strings.TrimSpace(variable.Key)
		if key == "" {
			continue
		}
		out[key] = stringify(variable.Value)
	}
	return out
}

var postmanVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

func ensureTargetVariables(target string, variables map[string]string) {
	matches := postmanVariablePattern.FindAllStringSubmatch(target, -1)
	for _, match := range matches {
		key := match[1]
		if _, exists := variables[key]; exists {
			continue
		}
		if target == match[0] {
			variables[key] = "https://example.com"
		} else {
			variables[key] = "example.com"
		}
	}
}

func requestName(itemName string, folders []string, method string, requestPath string) string {
	parts := make([]string, 0, len(folders)+1)
	for _, folder := range folders {
		if strings.TrimSpace(folder) != "" {
			parts = append(parts, strings.TrimSpace(folder))
		}
	}
	if strings.TrimSpace(itemName) != "" {
		parts = append(parts, strings.TrimSpace(itemName))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " / ")
	}
	if requestPath == "" {
		requestPath = "/"
	}
	return method + " " + requestPath
}

func supportedMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

func hasHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func toKebab(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			previousDash = false
			continue
		}
		if !previousDash && builder.Len() > 0 {
			builder.WriteByte('-')
			previousDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func dedupeWarnings(warnings []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" || seen[warning] {
			continue
		}
		seen[warning] = true
		out = append(out, warning)
	}
	sort.Strings(out)
	return out
}
