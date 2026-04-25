package har

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devaryakjha/loadwright/internal/spec"
)

type Options struct {
	BaseURL string
	Name    string
}

type Result struct {
	Spec     *spec.Spec
	Warnings []string
}

type Archive struct {
	Log Log `json:"log"`
}

type Log struct {
	Version string  `json:"version"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Request Request `json:"request"`
}

type Request struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	Headers     []NameValue `json:"headers"`
	QueryString []NameValue `json:"queryString"`
	Cookies     []NameValue `json:"cookies"`
	PostData    *PostData   `json:"postData"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type PostData struct {
	MimeType string      `json:"mimeType"`
	Text     string      `json:"text"`
	Encoding string      `json:"encoding"`
	Params   []PostParam `json:"params"`
}

type PostParam struct {
	Name        string `json:"name"`
	Value       any    `json:"value"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
}

func ImportFile(path string, options Options) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read HAR file: %w", err)
	}
	var archive Archive
	if err := json.Unmarshal(data, &archive); err != nil {
		return Result{}, fmt.Errorf("parse HAR JSON: %w", err)
	}
	if strings.TrimSpace(options.Name) == "" {
		options.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return Import(archive, options)
}

func Import(archive Archive, options Options) (Result, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "har-import"
	}
	loops := 1
	errorRate := 1.0
	p95 := 3000.0
	out := &spec.Spec{
		Name:      toKebab(name),
		Target:    strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
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

	var warnings []string
	var firstTarget string
	for index, entry := range archive.Log.Entries {
		request, target, requestWarnings, ok := requestFromHAR(index, entry.Request)
		warnings = append(warnings, requestWarnings...)
		if !ok {
			continue
		}
		if firstTarget == "" && target != "" {
			firstTarget = target
		}
		if out.Target == "" && target != "" && firstTarget != "" && target != firstTarget {
			warnings = append(warnings, fmt.Sprintf("%s uses target %s; imported path will run against %s", request.Name, target, firstTarget))
		}
		if out.Target != "" && target != "" && target != out.Target {
			warnings = append(warnings, fmt.Sprintf("%s uses target %s; imported path will run against %s", request.Name, target, out.Target))
		}
		out.Requests = append(out.Requests, request)
	}
	if len(out.Requests) == 0 {
		return Result{}, fmt.Errorf("HAR file has no supported requests")
	}
	if out.Target == "" {
		out.Target = firstTarget
	}
	if out.Target == "" {
		return Result{}, fmt.Errorf("HAR file has no absolute request URLs")
	}
	if _, err := out.Resolve(nil); err != nil {
		return Result{}, err
	}
	return Result{Spec: out, Warnings: dedupeWarnings(warnings)}, nil
}

func requestFromHAR(index int, harRequest Request) (spec.Request, string, []string, bool) {
	var warnings []string
	method := strings.ToUpper(strings.TrimSpace(harRequest.Method))
	if method == "" {
		method = "GET"
	}
	name := requestName(method, harRequest.URL)
	if !supportedMethod(method) {
		return spec.Request{}, "", []string{fmt.Sprintf("%s uses unsupported method %s and was skipped", name, method)}, false
	}
	target, requestPath, query, err := splitURL(harRequest.URL)
	if err != nil {
		return spec.Request{}, "", []string{fmt.Sprintf("entry %d has unsupported URL %q and was skipped", index, harRequest.URL)}, false
	}
	for _, item := range harRequest.QueryString {
		key := strings.TrimSpace(item.Name)
		if key == "" {
			continue
		}
		query[key] = stringify(item.Value)
	}
	headers, headerWarnings := headersFromHAR(name, harRequest.Headers)
	warnings = append(warnings, headerWarnings...)
	if len(harRequest.Cookies) > 0 {
		warnings = append(warnings, fmt.Sprintf("%s has cookies; cookies are not imported", name))
	}
	body, bodyWarnings := bodyFromHAR(name, harRequest.PostData, headers)
	warnings = append(warnings, bodyWarnings...)

	return spec.Request{
		Name:     name,
		Method:   method,
		Path:     requestPath,
		Headers:  headers,
		Query:    query,
		Body:     body.Legacy,
		BodyJSON: body.JSON,
		BodyText: body.Text,
		BodyForm: body.Form,
		Expect:   spec.Expect{Status: 200},
	}, target, warnings, true
}

func splitURL(raw string) (target string, requestPath string, query map[string]string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", nil, fmt.Errorf("URL must be absolute")
	}
	target = parsed.Scheme + "://" + parsed.Host
	requestPath = parsed.EscapedPath()
	if requestPath == "" {
		requestPath = "/"
	}
	cleaned := path.Clean("/" + strings.TrimLeft(requestPath, "/"))
	if cleaned == "." {
		cleaned = "/"
	}
	query = map[string]string{}
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}
	return strings.TrimRight(target, "/"), cleaned, query, nil
}

func headersFromHAR(requestName string, headers []NameValue) (map[string]string, []string) {
	out := map[string]string{}
	var warnings []string
	for _, header := range headers {
		key := strings.TrimSpace(header.Name)
		if key == "" {
			continue
		}
		switch {
		case strings.EqualFold(key, "host"),
			strings.EqualFold(key, "content-length"),
			strings.EqualFold(key, "cookie"):
			if strings.EqualFold(key, "cookie") {
				warnings = append(warnings, fmt.Sprintf("%s has cookie header; cookies are not imported", requestName))
			}
			continue
		case strings.EqualFold(key, "authorization"):
			warnings = append(warnings, fmt.Sprintf("%s has authorization header; imported as a static header", requestName))
		}
		out[key] = stringify(header.Value)
	}
	return out, warnings
}

type requestBody struct {
	Legacy any
	JSON   any
	Text   string
	Form   map[string]string
}

func bodyFromHAR(requestName string, postData *PostData, headers map[string]string) (requestBody, []string) {
	if postData == nil {
		return requestBody{}, nil
	}
	if strings.TrimSpace(postData.Encoding) != "" && !strings.EqualFold(postData.Encoding, "utf-8") {
		return requestBody{}, []string{fmt.Sprintf("%s request body uses %s encoding and was skipped", requestName, postData.Encoding)}
	}
	if len(postData.Params) > 0 {
		for _, param := range postData.Params {
			if strings.TrimSpace(param.FileName) != "" {
				return requestBody{}, []string{fmt.Sprintf("%s has file upload form data; body was skipped", requestName)}
			}
		}
		return requestBody{Form: paramsBody(postData.Params)}, nil
	}
	text := strings.TrimSpace(postData.Text)
	if text == "" {
		return requestBody{}, nil
	}
	if shouldParseJSON(text, postData.MimeType, headers) {
		var parsed any
		if err := json.Unmarshal([]byte(text), &parsed); err == nil {
			if !hasHeader(headers, "content-type") {
				headers["Content-Type"] = "application/json"
			}
			return requestBody{JSON: parsed}, nil
		}
		return requestBody{Text: text}, []string{fmt.Sprintf("%s body looked like JSON but could not be parsed; imported as string", requestName)}
	}
	return requestBody{Text: text}, nil
}

func paramsBody(params []PostParam) map[string]string {
	out := map[string]string{}
	for _, param := range params {
		key := strings.TrimSpace(param.Name)
		if key == "" {
			continue
		}
		out[key] = stringify(param.Value)
	}
	return out
}

func shouldParseJSON(text string, mimeType string, headers map[string]string) bool {
	if strings.Contains(strings.ToLower(mimeType), "json") {
		return true
	}
	for key, value := range headers {
		if strings.EqualFold(key, "content-type") && strings.Contains(strings.ToLower(value), "json") {
			return true
		}
	}
	return strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")
}

func requestName(method string, rawURL string) string {
	_, requestPath, _, err := splitURL(rawURL)
	if err != nil || requestPath == "" {
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
