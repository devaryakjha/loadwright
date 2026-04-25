package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseCompileArgsAcceptsFlagsAfterSpec(t *testing.T) {
	specPath, output, envFile, err := parseCompileArgs([]string{"spec.yaml", "-o", "tests/spec.jmx", "--env-file", ".env.test"})
	if err != nil {
		t.Fatalf("parseCompileArgs() error = %v", err)
	}
	if specPath != "spec.yaml" || output != "tests/spec.jmx" || envFile != ".env.test" {
		t.Fatalf("unexpected args: spec=%q output=%q env=%q", specPath, output, envFile)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(version) code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Loadwright") {
		t.Fatalf("version output = %q", stdout.String())
	}
}

func TestParseCompileArgsErrors(t *testing.T) {
	if _, _, _, err := parseCompileArgs([]string{"spec.yaml", "-o"}); err == nil {
		t.Fatalf("expected missing output value error")
	}
	if _, _, _, err := parseCompileArgs([]string{}); err == nil {
		t.Fatalf("expected missing spec error")
	}
}

func TestParseValidateArgs(t *testing.T) {
	specPath, envFile, err := parseValidateArgs([]string{"spec.yaml", "--env-file=.env.test"})
	if err != nil {
		t.Fatalf("parseValidateArgs() error = %v", err)
	}
	if specPath != "spec.yaml" || envFile != ".env.test" {
		t.Fatalf("unexpected args: spec=%q env=%q", specPath, envFile)
	}
}

func TestParseValidateArgsErrors(t *testing.T) {
	if _, _, err := parseValidateArgs([]string{"spec.yaml", "--env-file"}); err == nil {
		t.Fatalf("expected missing env-file value error")
	}
	if _, _, err := parseValidateArgs([]string{}); err == nil {
		t.Fatalf("expected missing spec error")
	}
}

func TestParseRunArgsAcceptsInterspersedFlags(t *testing.T) {
	input, outputDir, envFile, ci, image, err := parseRunArgs([]string{"spec.yaml", "--ci", "--out-dir=results/test", "--env-file=.env.test", "--image", "jmeter:test"})
	if err != nil {
		t.Fatalf("parseRunArgs() error = %v", err)
	}
	if input != "spec.yaml" || outputDir != "results/test" || envFile != ".env.test" || !ci || image != "jmeter:test" {
		t.Fatalf("unexpected args: input=%q outputDir=%q env=%q ci=%v image=%q", input, outputDir, envFile, ci, image)
	}
}

func TestParseRunArgsErrors(t *testing.T) {
	if _, _, _, _, _, err := parseRunArgs([]string{"spec.yaml", "--out-dir"}); err == nil {
		t.Fatalf("expected missing out-dir value error")
	}
	if _, _, _, _, _, err := parseRunArgs([]string{"spec.yaml", "--image"}); err == nil {
		t.Fatalf("expected missing image value error")
	}
	if _, _, _, _, _, err := parseRunArgs([]string{}); err == nil {
		t.Fatalf("expected missing input error")
	}
}

func TestParseReportArgs(t *testing.T) {
	jtlPath, outputDir, thresholds, ci, err := parseReportArgs([]string{
		"results.jtl",
		"--out-dir=reports",
		"--error-rate-lt", "1",
		"--p95-ms-lt=500",
		"--avg-ms-lt", "250",
		"--ci",
	})
	if err != nil {
		t.Fatalf("parseReportArgs() error = %v", err)
	}
	if jtlPath != "results.jtl" || outputDir != "reports" || !ci {
		t.Fatalf("unexpected args: jtl=%q out=%q ci=%v", jtlPath, outputDir, ci)
	}
	if thresholds.ErrorRateLT == nil || *thresholds.ErrorRateLT != 1 {
		t.Fatalf("unexpected error threshold: %+v", thresholds.ErrorRateLT)
	}
	if thresholds.P95MsLT == nil || *thresholds.P95MsLT != 500 {
		t.Fatalf("unexpected p95 threshold: %+v", thresholds.P95MsLT)
	}
	if thresholds.AvgMsLT == nil || *thresholds.AvgMsLT != 250 {
		t.Fatalf("unexpected avg threshold: %+v", thresholds.AvgMsLT)
	}
}

func TestParseReportArgsErrors(t *testing.T) {
	if _, _, _, _, err := parseReportArgs([]string{}); err == nil {
		t.Fatalf("expected missing JTL path error")
	}
	if _, _, _, _, err := parseReportArgs([]string{"results.jtl", "--p95-ms-lt"}); err == nil {
		t.Fatalf("expected missing threshold value error")
	}
	if _, _, _, _, err := parseReportArgs([]string{"results.jtl", "--avg-ms-lt=-1"}); err == nil {
		t.Fatalf("expected negative threshold error")
	}
}

func TestParseCompareArgs(t *testing.T) {
	baseline, candidate, output, err := parseCompareArgs([]string{"baseline.json", "candidate.json", "--out=comparison.md"})
	if err != nil {
		t.Fatalf("parseCompareArgs() error = %v", err)
	}
	if baseline != "baseline.json" || candidate != "candidate.json" || output != "comparison.md" {
		t.Fatalf("unexpected args: baseline=%q candidate=%q output=%q", baseline, candidate, output)
	}
}

func TestParseCompareArgsErrors(t *testing.T) {
	if _, _, _, err := parseCompareArgs([]string{"baseline.json"}); err == nil {
		t.Fatalf("expected missing candidate error")
	}
	if _, _, _, err := parseCompareArgs([]string{"baseline.json", "candidate.json", "--out"}); err == nil {
		t.Fatalf("expected missing output value error")
	}
}

func TestParseDoctorArgs(t *testing.T) {
	deep, image, err := parseDoctorArgs([]string{"--deep", "--image=custom:jmeter"})
	if err != nil {
		t.Fatalf("parseDoctorArgs() error = %v", err)
	}
	if !deep || image != "custom:jmeter" {
		t.Fatalf("unexpected args: deep=%v image=%q", deep, image)
	}
}

func TestParseDoctorArgsRejectsUnknown(t *testing.T) {
	if _, _, err := parseDoctorArgs([]string{"--unknown"}); err == nil {
		t.Fatalf("expected unknown doctor option error")
	}
}

func TestParseImportOpenAPIArgs(t *testing.T) {
	input, output, baseURL, err := parseImportOpenAPIArgs([]string{"openapi.yaml", "-o", "loadwright.yaml", "--base-url=https://staging.example.com"})
	if err != nil {
		t.Fatalf("parseImportOpenAPIArgs() error = %v", err)
	}
	if input != "openapi.yaml" || output != "loadwright.yaml" || baseURL != "https://staging.example.com" {
		t.Fatalf("unexpected args: input=%q output=%q baseURL=%q", input, output, baseURL)
	}
}

func TestParseImportPostmanArgs(t *testing.T) {
	input, output, baseURL, err := parseImportPostmanArgs([]string{"collection.json", "--out=loadwright.yaml", "--base-url", "https://staging.example.com"})
	if err != nil {
		t.Fatalf("parseImportPostmanArgs() error = %v", err)
	}
	if input != "collection.json" || output != "loadwright.yaml" || baseURL != "https://staging.example.com" {
		t.Fatalf("unexpected args: input=%q output=%q baseURL=%q", input, output, baseURL)
	}
}

func TestParseImportHARArgs(t *testing.T) {
	input, output, baseURL, err := parseImportHARArgs([]string{"capture.har", "-o", "loadwright.yaml", "--base-url=https://staging.example.com"})
	if err != nil {
		t.Fatalf("parseImportHARArgs() error = %v", err)
	}
	if input != "capture.har" || output != "loadwright.yaml" || baseURL != "https://staging.example.com" {
		t.Fatalf("unexpected args: input=%q output=%q baseURL=%q", input, output, baseURL)
	}
}

func TestParseImportOpenAPIArgsErrors(t *testing.T) {
	if _, _, _, err := parseImportOpenAPIArgs([]string{"openapi.yaml", "--base-url"}); err == nil {
		t.Fatalf("expected missing base-url value error")
	}
	if _, _, _, err := parseImportOpenAPIArgs([]string{}); err == nil {
		t.Fatalf("expected missing OpenAPI input error")
	}
}

func TestRunInitCreatesStarterSpec(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"init"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(init) code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "loadwright.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: example-api") {
		t.Fatalf("unexpected starter spec: %s", data)
	}
}

func TestRunCompileCreatesJMX(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specPath := filepath.Join(dir, "spec.yaml")
	specYAML := `name: compile-me
target: https://example.com
requests:
  - path: /health
`
	if err := os.WriteFile(specPath, []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"compile", specPath, "-o", "out/test.jmx"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(compile) code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "out", "test.jmx"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `testname="compile-me"`) {
		t.Fatalf("unexpected JMX: %s", data)
	}
}

func TestRunCompileUsesEnvFile(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specPath := filepath.Join(dir, "spec.yaml")
	envPath := filepath.Join(dir, ".env.test")
	specYAML := `name: env-compile
target: https://example.com
variables:
  token: ${API_TOKEN}
auth:
  type: bearer
  token: "{{token}}"
requests:
  - path: /secure
`
	if err := os.WriteFile(specPath, []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("API_TOKEN=abc123\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"compile", specPath, "--env-file", envPath, "-o", "out/test.jmx"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(compile) code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "out", "test.jmx"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Bearer abc123") {
		t.Fatalf("compiled JMX missing resolved bearer token: %s", data)
	}
}

func TestRunValidateAcceptsResolvedSpec(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	specYAML := `name: validate-me
target: https://example.com
variables:
  token: ${API_TOKEN}
auth:
  type: bearer
  token: "{{token}}"
requests:
  - path: /secure
`
	if err := os.WriteFile("spec.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env.test", []byte("API_TOKEN=abc123\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate", "spec.yaml", "--env-file", ".env.test"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(validate) code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid spec: validate-me (1 request)") {
		t.Fatalf("unexpected validate output: %s", stdout.String())
	}
}

func TestRunValidateRejectsInvalidSpec(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if err := os.WriteFile("bad.yaml", []byte("name: bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate", "bad.yaml"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "invalid spec:") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got %s", stdout.String())
	}
}

func TestRunImportOpenAPICreatesSpec(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	openAPI := `openapi: 3.0.3
info:
  title: Import Test
servers:
  - url: https://api.example.com
paths:
  /health:
    get:
      responses:
        "200": {}
`
	if err := os.WriteFile("openapi.yaml", []byte(openAPI), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"import", "openapi", "openapi.yaml", "-o", "loadwright.yaml"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(import openapi) code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile("loadwright.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "target: https://api.example.com") {
		t.Fatalf("unexpected imported spec: %s", data)
	}
}

func TestRunImportPostmanCreatesSpecAndWarnings(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	collection := `{
  "info": {"name": "CLI Import"},
  "variable": [{"key": "base_url", "value": "https://api.example.com"}],
  "item": [
    {
      "name": "Create user",
      "request": {
        "method": "POST",
        "header": [{"key": "Content-Type", "value": "application/json"}],
        "body": {"mode": "raw", "raw": "{\"name\":\"Ada\"}", "options": {"raw": {"language": "json"}}},
        "url": "{{base_url}}/users"
      }
    },
    {
      "name": "Upload",
      "request": {
        "method": "POST",
        "body": {"mode": "formdata"},
        "url": "{{base_url}}/upload"
      }
    }
  ]
}`
	if err := os.WriteFile("collection.json", []byte(collection), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"import", "postman", "collection.json", "-o", "loadwright.yaml"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(import postman) code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile("loadwright.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "target: '{{base_url}}'") ||
		!strings.Contains(string(data), "name: Create user") ||
		!strings.Contains(string(data), "name: Ada") {
		t.Fatalf("unexpected imported spec: %s", data)
	}
	if !strings.Contains(stderr.String(), `warning: Upload: request body mode "formdata" is not imported yet`) {
		t.Fatalf("expected warning, got stderr=%s", stderr.String())
	}
}

func TestRunImportHARCreatesSpecAndWarnings(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	capture := `{
  "log": {
    "version": "1.2",
    "entries": [
      {
        "request": {
          "method": "GET",
          "url": "https://api.example.com/users?limit=10",
          "headers": [{"name": "Accept", "value": "application/json"}],
          "queryString": [{"name": "active", "value": true}]
        }
      },
      {
        "request": {
          "method": "POST",
          "url": "https://api.example.com/users",
          "headers": [{"name": "Content-Type", "value": "application/json"}],
          "postData": {
            "mimeType": "application/json",
            "text": "{\"name\":\"Ada\"}"
          }
        }
      },
      {
        "request": {
          "method": "GET",
          "url": "https://other.example.com/health"
        }
      }
    ]
  }
}`
	if err := os.WriteFile("capture.har", []byte(capture), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"import", "har", "capture.har", "-o", "loadwright.yaml"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(import har) code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile("loadwright.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "target: https://api.example.com") ||
		!strings.Contains(string(data), "name: GET /users") ||
		!strings.Contains(string(data), "name: Ada") {
		t.Fatalf("unexpected imported spec: %s", data)
	}
	if !strings.Contains(stderr.String(), "warning: GET /health uses target https://other.example.com; imported path will run against https://api.example.com") {
		t.Fatalf("expected warning, got stderr=%s", stderr.String())
	}
}

func TestRunImportRejectsUnsupportedSource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"import", "insomnia", "export.json"}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "unsupported import source") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunReportCreatesArtifacts(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	jtl := `timeStamp,elapsed,label,responseCode,success
1,100,GET /health,200,true
2,200,GET /health,200,true
`
	if err := os.WriteFile("results.jtl", []byte(jtl), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"report", "results.jtl", "--out-dir", "report-out", "--error-rate-lt=1", "--p95-ms-lt=500", "--ci"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(report) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, name := range []string{"summary.json", "summary.md", "index.html", "junit.xml"} {
		if _, err := os.Stat(filepath.Join("report-out", name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	if !strings.Contains(stdout.String(), filepath.Join("report-out", "index.html")) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunReportFailsCIOnThresholds(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	jtl := `timeStamp,elapsed,label,responseCode,success
1,100,GET /health,200,true
2,1000,GET /health,500,false
`
	if err := os.WriteFile("results.jtl", []byte(jtl), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"report", "results.jtl", "--out-dir", "report-out", "--error-rate-lt", "1", "--ci"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "thresholds failed") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join("report-out", "summary.json")); err != nil {
		t.Fatalf("expected report artifacts despite failed thresholds: %v", err)
	}
}

func TestRunCompareWritesMarkdown(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	baseline := `{
  "total_samples": 2,
  "successful": 2,
  "failed": 0,
  "error_rate": 0,
  "average_ms": 100,
  "p95_ms": 150,
  "p99_ms": 180,
  "endpoints": {
    "GET /health": {"count": 2, "failed": 0, "average_ms": 100, "p95_ms": 150}
  }
}`
	candidate := `{
  "total_samples": 2,
  "successful": 1,
  "failed": 1,
  "error_rate": 50,
  "average_ms": 250,
  "p95_ms": 400,
  "p99_ms": 450,
  "endpoints": {
    "GET /health": {"count": 2, "failed": 1, "average_ms": 250, "p95_ms": 400}
  }
}`
	if err := os.WriteFile("baseline.json", []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("candidate.json", []byte(candidate), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"compare", "baseline.json", "candidate.json", "--out", "comparison.md"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(compare) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile("comparison.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Loadwright Comparison") ||
		!strings.Contains(string(data), "| Failed samples | 0 | 1 | +1 |") ||
		!strings.Contains(string(data), "| GET /health | changed | 0 | 1 | +50.00% | +150.00 ms | +250.00 ms |") {
		t.Fatalf("unexpected comparison:\n%s", data)
	}
	if !strings.Contains(stdout.String(), "wrote comparison.md") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunComparePrintsMarkdown(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if err := os.WriteFile("baseline.json", []byte(`{"total_samples":1,"endpoints":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("candidate.json", []byte(`{"total_samples":2,"endpoints":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"compare", "baseline.json", "candidate.json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(compare) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "| Total samples | 1 | 2 | +1 |") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunCompileRejectsInvalidSpec(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if err := os.WriteFile("bad.yaml", []byte("name: bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"compile", "bad.yaml"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "invalid spec:") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "- target must be an absolute http or https URL") ||
		!strings.Contains(stderr.String(), "- requests must contain at least one request") {
		t.Fatalf("expected grouped validation errors, got stderr=%s", stderr.String())
	}
}

func TestRunSpecCreatesReportsWithDockerShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker shim uses POSIX shell")
	}
	dir := t.TempDir()
	chdir(t, dir)
	installDockerShim(t, dir)
	specYAML := `name: shim-run
target: https://example.com
load:
  users: 2
  loops: 1
requests:
  - name: health
    method: GET
    path: /health
    expect:
      status: 200
thresholds:
  error_rate_lt: 1
  p95_ms_lt: 500
`
	if err := os.WriteFile("spec.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "spec.yaml", "--out-dir", "results/smoke", "--ci"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(run) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, name := range []string{"shim-run.jmx", "results.jtl", "summary.json", "summary.md", "index.html", "junit.xml", "run.json"} {
		if _, err := os.Stat(filepath.Join("results", "smoke", name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	summary, err := os.ReadFile(filepath.Join("results", "smoke", "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), `"total_samples": 1`) ||
		!strings.Contains(string(summary), `"failed": 0`) {
		t.Fatalf("unexpected summary: %s", summary)
	}
	manifest := readRunManifest(t, filepath.Join("results", "smoke", "run.json"))
	if manifest.RunID != "smoke" || manifest.Input != "spec.yaml" || manifest.InputType != "yaml" || !manifest.GeneratedJMX {
		t.Fatalf("unexpected run manifest: %+v", manifest)
	}
	if manifest.JMX != filepath.ToSlash(filepath.Join("results", "smoke", "shim-run.jmx")) {
		t.Fatalf("unexpected JMX path: %+v", manifest)
	}
	if manifest.Image != "justb4/jmeter:latest" || !manifest.CI {
		t.Fatalf("unexpected run flags: %+v", manifest)
	}
	if manifest.Artifacts.ReportHTML != filepath.ToSlash(filepath.Join("results", "smoke", "index.html")) ||
		manifest.Artifacts.ResultsJTL != filepath.ToSlash(filepath.Join("results", "smoke", "results.jtl")) {
		t.Fatalf("unexpected artifacts: %+v", manifest.Artifacts)
	}
	if manifest.StartedAt == "" || manifest.FinishedAt == "" {
		t.Fatalf("expected timestamps: %+v", manifest)
	}
}

func TestRunSpecDefaultOutputCreatesLatestPointer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker shim uses POSIX shell")
	}
	dir := t.TempDir()
	chdir(t, dir)
	installDockerShim(t, dir)
	specYAML := `name: latest-run
target: https://example.com
requests:
  - name: health
    path: /health
`
	if err := os.WriteFile("spec.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "spec.yaml"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(run) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	metadata := readLatestRun(t, filepath.Join("results", "latest.json"))
	if metadata.RunID == "" {
		t.Fatalf("expected run_id in latest metadata: %+v", metadata)
	}
	if metadata.RunDir != filepath.ToSlash(filepath.Join("results", metadata.RunID)) {
		t.Fatalf("unexpected run_dir: %+v", metadata)
	}
	if metadata.Report != filepath.ToSlash(filepath.Join("results", metadata.RunID, "index.html")) {
		t.Fatalf("unexpected report path: %+v", metadata)
	}
	if _, err := os.Stat(filepath.FromSlash(metadata.Report)); err != nil {
		t.Fatalf("expected latest report path to exist: %v", err)
	}
	manifest := readRunManifest(t, filepath.Join("results", metadata.RunID, "run.json"))
	if manifest.RunID != metadata.RunID || manifest.Artifacts.ReportHTML != metadata.Report {
		t.Fatalf("latest metadata and run manifest disagree: latest=%+v run=%+v", metadata, manifest)
	}
}

func TestRunSpecExplicitOutputDoesNotCreateLatestPointer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker shim uses POSIX shell")
	}
	dir := t.TempDir()
	chdir(t, dir)
	installDockerShim(t, dir)
	specYAML := `name: explicit-run
target: https://example.com
requests:
  - path: /health
`
	if err := os.WriteFile("spec.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "spec.yaml", "--out-dir", "results/manual"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(run) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join("results", "latest.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no latest pointer for explicit out-dir, err=%v", err)
	}
}

func TestRunSpecThresholdFailureStillCreatesLatestPointer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker shim uses POSIX shell")
	}
	dir := t.TempDir()
	chdir(t, dir)
	installDockerShim(t, dir)
	specYAML := `name: failing-latest-run
target: https://example.com
requests:
  - name: health
    path: /health
thresholds:
  p95_ms_lt: 1
`
	if err := os.WriteFile("spec.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "spec.yaml", "--ci"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "thresholds failed") {
		t.Fatalf("Run(run) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	metadata := readLatestRun(t, filepath.Join("results", "latest.json"))
	if metadata.Report == "" {
		t.Fatalf("expected latest metadata despite threshold failure: %+v", metadata)
	}
	if _, err := os.Stat(filepath.FromSlash(metadata.Report)); err != nil {
		t.Fatalf("expected latest report path to exist: %v", err)
	}
}

func TestRunExistingJMXCreatesRunManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker shim uses POSIX shell")
	}
	dir := t.TempDir()
	chdir(t, dir)
	installDockerShim(t, dir)
	if err := os.WriteFile("existing.jmx", []byte("<jmeterTestPlan/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "existing.jmx", "--out-dir", "results/jmx-smoke", "--image", "custom:jmeter"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(run existing JMX) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	manifest := readRunManifest(t, filepath.Join("results", "jmx-smoke", "run.json"))
	if manifest.Input != "existing.jmx" || manifest.InputType != "jmx" || manifest.GeneratedJMX {
		t.Fatalf("unexpected JMX manifest: %+v", manifest)
	}
	if manifest.JMX != "existing.jmx" || manifest.Image != "custom:jmeter" || manifest.CI {
		t.Fatalf("unexpected JMX manifest fields: %+v", manifest)
	}
}

func TestRunManifestDoesNotIncludeEnvValues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake docker shim uses POSIX shell")
	}
	dir := t.TempDir()
	chdir(t, dir)
	installDockerShim(t, dir)
	specYAML := `name: secret-run
target: https://example.com
variables:
  token: ${API_TOKEN}
auth:
  type: bearer
  token: "{{token}}"
requests:
  - path: /secure
`
	if err := os.WriteFile("spec.yaml", []byte(specYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env.test", []byte("API_TOKEN=super-secret-token\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "spec.yaml", "--env-file", ".env.test", "--out-dir", "results/secret"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(run) code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join("results", "secret", "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-token") || strings.Contains(string(data), "API_TOKEN") || strings.Contains(string(data), ".env.test") {
		t.Fatalf("run manifest leaked env data: %s", data)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func readLatestRun(t *testing.T, path string) latestRun {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var metadata latestRun
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	return metadata
}

func readRunManifest(t *testing.T, path string) runManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest runManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func installDockerShim(t *testing.T, dir string) {
	t.Helper()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
set -eu
workdir=""
jtl=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -v)
      workdir="${2%:/work}"
      shift 2
      ;;
    -l)
      jtl="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
if [ -z "$workdir" ] || [ -z "$jtl" ]; then
  echo "fake docker expected -v and -l" >&2
  exit 1
fi
case "$jtl" in
  /work/*) output="$workdir/${jtl#/work/}" ;;
  *) output="$jtl" ;;
esac
mkdir -p "$(dirname "$output")"
cat > "$output" <<'JTL'
timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
1,120,health,200,OK,thread-1,text,true,64,64,1,1,https://example.com/health,100,0,20
JTL
`
	dockerPath := filepath.Join(binDir, "docker")
	if err := os.WriteFile(dockerPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	previousPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+previousPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Setenv("PATH", previousPath); err != nil {
			t.Fatalf("restore PATH: %v", err)
		}
	})
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"wat"}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}
