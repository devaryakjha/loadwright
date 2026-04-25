package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportYAMLOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(path, []byte(petstoreLiteYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	imported, err := ImportFile(path, Options{})
	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}
	if imported.Name != "petstore-lite" {
		t.Fatalf("name = %q", imported.Name)
	}
	if imported.Target != "https://api.example.com" {
		t.Fatalf("target = %q", imported.Target)
	}
	if len(imported.Requests) != 2 {
		t.Fatalf("requests = %d", len(imported.Requests))
	}
	if imported.Variables["pet_id"] != "example-pet-id" {
		t.Fatalf("pet_id variable = %q", imported.Variables["pet_id"])
	}
	requests := map[string]int{}
	for index, request := range imported.Requests {
		requests[request.Name] = index
	}
	get := imported.Requests[requests["getPet"]]
	if get.Name != "getPet" || get.Path != "/pets/{{pet_id}}" || get.Expect.Status != 200 {
		t.Fatalf("unexpected GET request: %+v", get)
	}
	post := imported.Requests[requests["createPet"]]
	if post.Name != "createPet" || post.Method != "POST" || post.Expect.Status != 201 {
		t.Fatalf("unexpected POST request: %+v", post)
	}
	body := post.Body.(map[string]any)
	if body["name"] != "example" || body["tag"] != "example" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestImportBaseURLOverride(t *testing.T) {
	imported, err := Import(Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Override"},
		Servers: []Server{{URL: "https://prod.example.com"}},
		Paths: map[string]PathItem{
			"/health": {Get: &Operation{Responses: map[string]Response{"200": {}}}},
		},
	}, Options{BaseURL: "https://staging.example.com"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if imported.Target != "https://staging.example.com" {
		t.Fatalf("target = %q", imported.Target)
	}
}

func TestImportJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.json")
	json := `{"openapi":"3.0.3","info":{"title":"JSON API"},"servers":[{"url":"https://json.example.com"}],"paths":{"/health":{"get":{"responses":{"204":{}}}}}}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	imported, err := ImportFile(path, Options{})
	if err != nil {
		t.Fatalf("ImportFile() error = %v", err)
	}
	if imported.Target != "https://json.example.com" || imported.Requests[0].Expect.Status != 204 {
		t.Fatalf("unexpected import: %+v", imported)
	}
}

func TestImportRejectsUnsupportedVersion(t *testing.T) {
	_, err := Import(Document{OpenAPI: "2.0", Paths: map[string]PathItem{"/x": {Get: &Operation{}}}}, Options{})
	if err == nil {
		t.Fatalf("expected unsupported version error")
	}
}

func TestImportRejectsNoPathsAndNoOperations(t *testing.T) {
	if _, err := Import(Document{OpenAPI: "3.0.3"}, Options{}); err == nil {
		t.Fatalf("expected no paths error")
	}
	if _, err := Import(Document{OpenAPI: "3.0.3", Paths: map[string]PathItem{"/x": {}}}, Options{}); err == nil {
		t.Fatalf("expected no operations error")
	}
}

func TestImportExamplesAndSchemaFallbacks(t *testing.T) {
	imported, err := Import(Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Examples"},
		Paths: map[string]PathItem{
			"/items": {Post: &Operation{
				RequestBody: &RequestBody{Content: map[string]MediaType{
					"application/json": {
						Examples: map[string]Example{"one": {Value: map[string]any{"name": "from-example"}}},
					},
				}},
				Responses: map[string]Response{"default": {}},
			}},
			"/flags": {Get: &Operation{Parameters: []Parameter{
				{Name: "enabled", In: "query", Schema: Schema{Type: "boolean"}},
				{Name: "count", In: "query", Schema: Schema{Type: "integer"}},
			}}},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	requests := map[string]int{}
	for index, request := range imported.Requests {
		requests[request.Name] = index
	}
	body := imported.Requests[requests["POST /items"]].Body.(map[string]any)
	if body["name"] != "from-example" {
		t.Fatalf("example body not used: %+v", body)
	}
	flags := imported.Requests[requests["GET /flags"]]
	if flags.Query["enabled"] != "true" || flags.Query["count"] != "1" {
		t.Fatalf("query examples not generated: %+v", flags.Query)
	}
}

func TestImportGlobalBearerSecurity(t *testing.T) {
	imported, err := Import(Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Bearer API"},
		Components: Components{SecuritySchemes: map[string]SecurityScheme{
			"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
		}},
		Security: []SecurityRequirement{{"bearerAuth": {}}},
		Paths: map[string]PathItem{
			"/secure": {Get: &Operation{Responses: map[string]Response{"200": {}}}},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if imported.Auth.Type != "bearer" || imported.Auth.Token != "{{api_token}}" {
		t.Fatalf("auth = %+v", imported.Auth)
	}
	if imported.Variables["api_token"] != "replace-me" {
		t.Fatalf("api_token variable = %q", imported.Variables["api_token"])
	}
}

func TestImportGlobalBasicSecurity(t *testing.T) {
	imported, err := Import(Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "Basic API"},
		Components: Components{SecuritySchemes: map[string]SecurityScheme{
			"basicAuth": {Type: "http", Scheme: "basic"},
		}},
		Security: []SecurityRequirement{{"basicAuth": {}}},
		Paths: map[string]PathItem{
			"/secure": {Get: &Operation{Responses: map[string]Response{"200": {}}}},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if imported.Auth.Type != "basic" || imported.Auth.Username != "{{basic_username}}" || imported.Auth.Password != "{{basic_password}}" {
		t.Fatalf("auth = %+v", imported.Auth)
	}
	if imported.Variables["basic_username"] != "replace-me" || imported.Variables["basic_password"] != "replace-me" {
		t.Fatalf("variables = %+v", imported.Variables)
	}
}

func TestImportIgnoresUnsupportedSecuritySchemes(t *testing.T) {
	imported, err := Import(Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: "API Key API"},
		Components: Components{SecuritySchemes: map[string]SecurityScheme{
			"apiKey": {Type: "apiKey", In: "header", Name: "X-API-Key"},
		}},
		Security: []SecurityRequirement{{"apiKey": {}}},
		Paths: map[string]PathItem{
			"/secure": {Get: &Operation{Responses: map[string]Response{"200": {}}}},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if !imported.Auth.IsZero() {
		t.Fatalf("unsupported auth should not be imported: %+v", imported.Auth)
	}
}

const petstoreLiteYAML = `openapi: 3.0.3
info:
  title: Petstore Lite
servers:
  - url: https://api.example.com
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200": {}
  /pets:
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                tag:
                  type: string
      responses:
        "201": {}
`
