package postman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportCollectionBasics(t *testing.T) {
	result, err := ImportFile(writeCollection(t, postmanCollection), Options{})
	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}
	imported := result.Spec
	if imported.Name != "checkout-api" {
		t.Fatalf("name = %q", imported.Name)
	}
	if imported.Target != "{{base_url}}" {
		t.Fatalf("target = %q", imported.Target)
	}
	if imported.Variables["base_url"] != "https://api.example.com" {
		t.Fatalf("base_url variable = %q", imported.Variables["base_url"])
	}
	if imported.Auth.Type != "bearer" || imported.Auth.Token != "{{token}}" {
		t.Fatalf("auth = %+v", imported.Auth)
	}
	if len(imported.Requests) != 2 {
		t.Fatalf("requests = %d", len(imported.Requests))
	}
	list := imported.Requests[0]
	if list.Name != "Users / List users" || list.Method != "GET" || list.Path != "/v1/users" {
		t.Fatalf("unexpected list request: %+v", list)
	}
	if list.Query["limit"] != "10" || list.Query["active"] != "true" {
		t.Fatalf("query = %+v", list.Query)
	}
	if list.Headers["X-Trace"] != "{{trace_id}}" {
		t.Fatalf("headers = %+v", list.Headers)
	}
	create := imported.Requests[1]
	if create.Method != "POST" || create.Path != "/v1/users" {
		t.Fatalf("unexpected create request: %+v", create)
	}
	body := create.Body.(map[string]any)
	if body["name"] != "Ada" || body["role"] != "admin" {
		t.Fatalf("body = %+v", body)
	}
	if create.Headers["Content-Type"] != "application/json" {
		t.Fatalf("headers = %+v", create.Headers)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
}

func TestImportBaseURLOverrideAndStringRequest(t *testing.T) {
	result, err := Import(Collection{
		Info: Info{Name: "String URL"},
		Items: []Item{{
			Name: "Health",
			Request: Request{
				Set:    true,
				Method: "GET",
				URL:    URL{Raw: "https://prod.example.com/health?ready=true"},
			},
		}},
	}, Options{BaseURL: "https://staging.example.com"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Spec.Target != "https://staging.example.com" {
		t.Fatalf("target = %q", result.Spec.Target)
	}
	request := result.Spec.Requests[0]
	if request.Path != "/health" || request.Query["ready"] != "true" {
		t.Fatalf("request = %+v", request)
	}
}

func TestImportRequestLevelBasicAuth(t *testing.T) {
	result, err := Import(Collection{
		Info: Info{Name: "Auth"},
		Items: []Item{{
			Name: "Secure",
			Request: Request{
				Set:    true,
				Method: "GET",
				URL:    URL{Raw: "https://api.example.com/secure"},
				Auth: &Auth{Type: "basic", Basic: []KeyValue{
					{Key: "username", Value: "{{username}}"},
					{Key: "password", Value: "{{password}}"},
				}},
			},
		}},
		Variables: []Variable{
			{Key: "username", Value: "demo"},
			{Key: "password", Value: "secret"},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	auth := result.Spec.Requests[0].Auth
	if auth.Type != "basic" || auth.Username != "{{username}}" || auth.Password != "{{password}}" {
		t.Fatalf("auth = %+v", auth)
	}
}

func TestImportWarnsForUnsupportedFeatures(t *testing.T) {
	result, err := Import(Collection{
		Info: Info{Name: "Unsupported"},
		Auth: &Auth{Type: "digest"},
		Items: []Item{
			{
				Name: "GraphQL",
				Request: Request{
					Set:    true,
					Method: "POST",
					URL:    URL{Raw: "https://api.example.com/graphql"},
					Body:   Body{Mode: "graphql"},
				},
			},
			{
				Name: "Trace",
				Request: Request{
					Set:    true,
					Method: "TRACE",
					URL:    URL{Raw: "https://api.example.com/trace"},
				},
			},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	joined := strings.Join(result.Warnings, "\n")
	if !strings.Contains(joined, `auth type "digest" is not imported yet`) ||
		!strings.Contains(joined, `request body mode "graphql" is not imported yet`) ||
		!strings.Contains(joined, "Trace uses unsupported method TRACE and was skipped") {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
	if len(result.Spec.Requests) != 1 || result.Spec.Requests[0].Name != "GraphQL" {
		t.Fatalf("requests = %+v", result.Spec.Requests)
	}
}

func TestImportRejectsNoSupportedRequests(t *testing.T) {
	_, err := Import(Collection{
		Info: Info{Name: "Empty"},
		Items: []Item{{
			Name: "Trace",
			Request: Request{
				Set:    true,
				Method: "TRACE",
				URL:    URL{Raw: "https://api.example.com/trace"},
			},
		}},
	}, Options{})
	if err == nil {
		t.Fatalf("expected no supported requests error")
	}
}

func writeCollection(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "collection.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const postmanCollection = `{
  "info": {
    "name": "Checkout API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    {"key": "base_url", "value": "https://api.example.com"},
    {"key": "token", "value": "demo-token"},
    {"key": "trace_id", "value": "trace-123"}
  ],
  "auth": {
    "type": "bearer",
    "bearer": [{"key": "token", "value": "{{token}}"}]
  },
  "item": [
    {
      "name": "Users",
      "item": [
        {
          "name": "List users",
          "request": {
            "method": "GET",
            "header": [
              {"key": "X-Trace", "value": "{{trace_id}}"},
              {"key": "X-Disabled", "value": "nope", "disabled": true}
            ],
            "url": {
              "raw": "{{base_url}}/v1/users?limit=10",
              "host": ["{{base_url}}"],
              "path": ["v1", "users"],
              "query": [
                {"key": "limit", "value": "10"},
                {"key": "active", "value": true}
              ]
            }
          }
        },
        {
          "name": "Create user",
          "request": {
            "method": "POST",
            "header": [{"key": "Content-Type", "value": "application/json"}],
            "body": {
              "mode": "raw",
              "raw": "{\"name\":\"Ada\",\"role\":\"admin\"}",
              "options": {"raw": {"language": "json"}}
            },
            "url": "{{base_url}}/v1/users"
          }
        }
      ]
    }
  ]
}`
