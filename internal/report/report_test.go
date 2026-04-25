package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devaryakjha/loadwright/internal/spec"
)

func TestParseJTLEvaluatesThresholds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.jtl")
	jtl := `timeStamp,elapsed,label,responseCode,success
1,100,GET /health,200,true
2,200,GET /health,200,true
3,1000,POST /login,500,false
`
	if err := os.WriteFile(path, []byte(jtl), 0o644); err != nil {
		t.Fatal(err)
	}
	errorRate := 50.0
	p95 := 900.0
	summary, err := ParseJTL(path, spec.Thresholds{
		ErrorRateLT: &errorRate,
		P95MsLT:     &p95,
	})
	if err != nil {
		t.Fatalf("ParseJTL() error = %v", err)
	}
	if summary.TotalSamples != 3 || summary.Failed != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.Passed() {
		t.Fatalf("expected p95 threshold to fail")
	}
}

func TestParseJTLAllSuccessPassesThresholds(t *testing.T) {
	path := writeJTL(t, `timeStamp,elapsed,label,responseCode,success
1,100,GET /health,200,true
2,200,GET /health,200,true
3,300,POST /login,200,true
`)
	errorRate := 1.0
	p95 := 500.0
	avg := 250.0
	summary, err := ParseJTL(path, spec.Thresholds{
		ErrorRateLT: &errorRate,
		P95MsLT:     &p95,
		AvgMsLT:     &avg,
	})
	if err != nil {
		t.Fatalf("ParseJTL() error = %v", err)
	}
	if !summary.Passed() {
		t.Fatalf("expected thresholds to pass: %+v", summary.Thresholds)
	}
	if len(summary.Endpoints) != 2 {
		t.Fatalf("expected two endpoints: %+v", summary.Endpoints)
	}
}

func TestParseJTLEmptyFile(t *testing.T) {
	path := writeJTL(t, "")
	summary, err := ParseJTL(path, spec.Thresholds{})
	if err != nil {
		t.Fatalf("ParseJTL() error = %v", err)
	}
	if summary.TotalSamples != 0 || len(summary.Endpoints) != 0 {
		t.Fatalf("unexpected empty summary: %+v", summary)
	}
}

func TestParseJTLMalformedCSV(t *testing.T) {
	path := writeJTL(t, "timeStamp,elapsed,label\n\"unterminated")
	if _, err := ParseJTL(path, spec.Thresholds{}); err == nil {
		t.Fatalf("expected malformed CSV error")
	}
}

func TestRenderOutputsContainThresholdStatus(t *testing.T) {
	summary := &Summary{
		TotalSamples: 2,
		Successful:   2,
		AverageMS:    100,
		P95MS:        150,
		Endpoints: map[string]Endpoint{
			"GET /health": {Count: 2, AverageMS: 100, P95MS: 150},
		},
		Thresholds: []ThresholdResult{{
			Name:   "p95_ms_lt",
			Limit:  200,
			Actual: 150,
			Passed: true,
		}},
	}
	if got := RenderMarkdown(summary); !strings.Contains(got, "p95_ms_lt: PASS") {
		t.Fatalf("markdown missing threshold pass: %s", got)
	}
	if got := RenderMarkdown(summary); !strings.Contains(got, "## Endpoints") ||
		!strings.Contains(got, "| GET /health | 2 | 0 | 0.00% | 100.00 ms | 150.00 ms |") {
		t.Fatalf("markdown missing endpoint table: %s", got)
	}
	if got := RenderHTML(summary); !strings.Contains(got, "p95_ms_lt") ||
		!strings.Contains(got, "PASS") ||
		!strings.Contains(got, "GET /health") ||
		!strings.Contains(got, "Endpoints") {
		t.Fatalf("html missing threshold table: %s", got)
	}
	if got := RenderJUnit(summary); !strings.Contains(got, `tests="2"`) ||
		!strings.Contains(got, `name="p95_ms_lt"`) ||
		strings.Contains(got, "<failure") {
		t.Fatalf("junit missing passing threshold case: %s", got)
	}
}

func TestRenderEndpointOrderingAndEscaping(t *testing.T) {
	summary := &Summary{
		TotalSamples: 6,
		Successful:   4,
		Failed:       2,
		Endpoints: map[string]Endpoint{
			"GET /fast":                  {Count: 2, Failed: 0, AverageMS: 50, P95MS: 60},
			"GET /slow":                  {Count: 2, Failed: 0, AverageMS: 300, P95MS: 900},
			"POST /danger?<script>|pipe": {Count: 2, Failed: 2, AverageMS: 200, P95MS: 250},
		},
	}
	htmlReport := RenderHTML(summary)
	failingIndex := strings.Index(htmlReport, "POST /danger?&lt;script&gt;|pipe")
	slowIndex := strings.Index(htmlReport, "GET /slow")
	fastIndex := strings.Index(htmlReport, "GET /fast")
	if failingIndex < 0 || slowIndex < 0 || fastIndex < 0 {
		t.Fatalf("html missing endpoints: %s", htmlReport)
	}
	if !(failingIndex < slowIndex && slowIndex < fastIndex) {
		t.Fatalf("endpoints not sorted by triage priority: %s", htmlReport)
	}
	if strings.Contains(htmlReport, "<script>") {
		t.Fatalf("html did not escape endpoint name: %s", htmlReport)
	}
	markdown := RenderMarkdown(summary)
	if !strings.Contains(markdown, "POST /danger?<script>\\|pipe") {
		t.Fatalf("markdown did not escape pipe in endpoint name: %s", markdown)
	}
}

func TestRenderEmptyReportSections(t *testing.T) {
	summary := &Summary{Endpoints: map[string]Endpoint{}}
	htmlReport := RenderHTML(summary)
	for _, want := range []string{"No thresholds configured.", "No endpoint samples found."} {
		if !strings.Contains(htmlReport, want) {
			t.Fatalf("html missing %q: %s", want, htmlReport)
		}
	}
}

func TestRenderJUnitContainsFailures(t *testing.T) {
	summary := &Summary{
		TotalSamples: 2,
		Successful:   1,
		Failed:       1,
		ErrorRate:    50,
		AverageMS:    500,
		P95MS:        950,
		Thresholds: []ThresholdResult{{
			Name:   "error_rate_lt",
			Limit:  1,
			Actual: 50,
			Passed: false,
		}},
	}
	got := RenderJUnit(summary)
	for _, want := range []string{
		`tests="2"`,
		`failures="2"`,
		`type="sample_failure"`,
		`type="threshold_failure"`,
		`error_rate_lt failed`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("junit = %s, want substring %q", got, want)
		}
	}
}

func TestWriteAllCreatesArtifacts(t *testing.T) {
	dir := t.TempDir()
	summary := &Summary{TotalSamples: 1, Successful: 1, Endpoints: map[string]Endpoint{}}
	if err := WriteAll(summary, dir); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}
	for _, name := range []string{"summary.json", "summary.md", "index.html", "junit.xml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
}

func TestLoadSummaryFileAndRenderComparison(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	candidatePath := filepath.Join(dir, "candidate.json")
	writeSummaryJSON(t, baselinePath, Summary{
		TotalSamples: 10,
		Successful:   9,
		Failed:       1,
		ErrorRate:    10,
		AverageMS:    100,
		P95MS:        200,
		P99MS:        250,
		Endpoints: map[string]Endpoint{
			"GET /stable":  {Count: 5, Failed: 0, AverageMS: 80, P95MS: 100},
			"GET /removed": {Count: 5, Failed: 1, AverageMS: 120, P95MS: 200},
		},
	})
	writeSummaryJSON(t, candidatePath, Summary{
		TotalSamples: 12,
		Successful:   10,
		Failed:       2,
		ErrorRate:    16.67,
		AverageMS:    150,
		P95MS:        260,
		P99MS:        300,
		Endpoints: map[string]Endpoint{
			"GET /stable":           {Count: 6, Failed: 1, AverageMS: 130, P95MS: 240},
			"POST /added|expensive": {Count: 6, Failed: 1, AverageMS: 170, P95MS: 280},
		},
	})
	baseline, err := LoadSummaryFile(baselinePath)
	if err != nil {
		t.Fatalf("LoadSummaryFile(baseline) error = %v", err)
	}
	candidate, err := LoadSummaryFile(candidatePath)
	if err != nil {
		t.Fatalf("LoadSummaryFile(candidate) error = %v", err)
	}
	markdown := RenderComparisonMarkdown(CompareSummaries(baseline, candidate))
	for _, want := range []string{
		"| Total samples | 10 | 12 | +2 |",
		"| Error rate | 10.00% | 16.67% | +6.67% |",
		"| GET /stable | changed | 0 | 1 | +16.67% | +50.00 ms | +140.00 ms |",
		"POST /added\\|expensive",
		"| GET /removed | removed | 1 | 0 | -20.00% | -120.00 ms | -200.00 ms |",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("comparison missing %q:\n%s", want, markdown)
		}
	}
	if strings.Index(markdown, "GET /stable") > strings.Index(markdown, "POST /added\\|expensive") {
		t.Fatalf("changed endpoints should sort before added endpoints:\n%s", markdown)
	}
}

func TestRenderComparisonWithoutEndpoints(t *testing.T) {
	markdown := RenderComparisonMarkdown(CompareSummaries(&Summary{}, &Summary{}))
	if !strings.Contains(markdown, "No endpoint data found.") {
		t.Fatalf("comparison missing empty endpoint message:\n%s", markdown)
	}
}

func writeJTL(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "results.jtl")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeSummaryJSON(t *testing.T, path string, summary Summary) {
	t.Helper()
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
