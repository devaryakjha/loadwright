package har

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportHARBasics(t *testing.T) {
	result, err := ImportFile(writeHAR(t, checkoutHAR), Options{})
	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}
	imported := result.Spec
	if imported.Name != "checkout" {
		t.Fatalf("name = %q", imported.Name)
	}
	if imported.Target != "https://api.example.com" {
		t.Fatalf("target = %q", imported.Target)
	}
	if len(imported.Requests) != 2 {
		t.Fatalf("requests = %d", len(imported.Requests))
	}
	list := imported.Requests[0]
	if list.Name != "GET /v1/users" || list.Method != "GET" || list.Path != "/v1/users" {
		t.Fatalf("unexpected list request: %+v", list)
	}
	if list.Query["limit"] != "10" || list.Query["active"] != "true" {
		t.Fatalf("query = %+v", list.Query)
	}
	if list.Headers["Accept"] != "application/json" {
		t.Fatalf("headers = %+v", list.Headers)
	}
	create := imported.Requests[1]
	if create.Name != "POST /v1/users" || create.Method != "POST" || create.Path != "/v1/users" {
		t.Fatalf("unexpected create request: %+v", create)
	}
	body := create.BodyJSON.(map[string]any)
	if body["name"] != "Ada" || body["role"] != "admin" {
		t.Fatalf("body_json = %+v", body)
	}
	if create.Headers["Content-Type"] != "application/json" {
		t.Fatalf("headers = %+v", create.Headers)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
}

func TestImportBaseURLOverrideAndWarnings(t *testing.T) {
	result, err := Import(Archive{Log: Log{Version: "1.2", Entries: []Entry{
		{Request: Request{
			Method: "GET",
			URL:    "https://prod.example.com/health",
			Headers: []NameValue{
				{Name: "Cookie", Value: "session=secret"},
				{Name: "Authorization", Value: "Bearer token"},
			},
			Cookies: []NameValue{{Name: "session", Value: "secret"}},
		}},
		{Request: Request{
			Method: "POST",
			URL:    "https://other.example.com/upload",
			PostData: &PostData{
				Encoding: "base64",
				Text:     "AAAA",
			},
		}},
	}}}, Options{Name: "Warnings", BaseURL: "https://staging.example.com"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Spec.Target != "https://staging.example.com" {
		t.Fatalf("target = %q", result.Spec.Target)
	}
	joined := strings.Join(result.Warnings, "\n")
	for _, expected := range []string{
		"GET /health has authorization header; imported as a static header",
		"GET /health has cookie header; cookies are not imported",
		"GET /health has cookies; cookies are not imported",
		"POST /upload request body uses base64 encoding and was skipped",
		"POST /upload uses target https://other.example.com; imported path will run against https://staging.example.com",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing warning %q in %+v", expected, result.Warnings)
		}
	}
	if _, ok := result.Spec.Requests[0].Headers["Cookie"]; ok {
		t.Fatalf("cookie header should not be imported: %+v", result.Spec.Requests[0].Headers)
	}
}

func TestImportFormParamsAsFormBody(t *testing.T) {
	result, err := Import(Archive{Log: Log{Entries: []Entry{{Request: Request{
		Method: "POST",
		URL:    "https://api.example.com/login",
		PostData: &PostData{Params: []PostParam{
			{Name: "username", Value: "demo"},
			{Name: "password", Value: "secret"},
		}},
	}}}}}, Options{Name: "Form"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	bodyForm := result.Spec.Requests[0].BodyForm
	if bodyForm["username"] != "demo" || bodyForm["password"] != "secret" {
		t.Fatalf("body_form = %+v", bodyForm)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
}

func TestImportRejectsNoSupportedRequests(t *testing.T) {
	_, err := Import(Archive{Log: Log{Entries: []Entry{{Request: Request{
		Method: "TRACE",
		URL:    "https://api.example.com/trace",
	}}}}}, Options{Name: "Empty"})
	if err == nil {
		t.Fatalf("expected no supported requests error")
	}
}

func TestImportRejectsRelativeURLs(t *testing.T) {
	_, err := Import(Archive{Log: Log{Entries: []Entry{{Request: Request{
		Method: "GET",
		URL:    "/relative",
	}}}}}, Options{Name: "Relative"})
	if err == nil {
		t.Fatalf("expected no absolute URL error")
	}
}

func writeHAR(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "checkout.har")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const checkoutHAR = `{
  "log": {
    "version": "1.2",
    "creator": {"name": "Loadwright test", "version": "1.0"},
    "entries": [
      {
        "request": {
          "method": "GET",
          "url": "https://api.example.com/v1/users?limit=10",
          "headers": [
            {"name": "Accept", "value": "application/json"},
            {"name": "Host", "value": "api.example.com"}
          ],
          "queryString": [
            {"name": "limit", "value": "10"},
            {"name": "active", "value": true}
          ]
        }
      },
      {
        "request": {
          "method": "POST",
          "url": "https://api.example.com/v1/users",
          "headers": [
            {"name": "Content-Type", "value": "application/json"},
            {"name": "Content-Length", "value": "29"}
          ],
          "postData": {
            "mimeType": "application/json",
            "text": "{\"name\":\"Ada\",\"role\":\"admin\"}"
          }
        }
      }
    ]
  }
}`
