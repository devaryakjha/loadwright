package report

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/devaryakjha/loadwright/internal/spec"
)

type Summary struct {
	TotalSamples int                 `json:"total_samples"`
	Successful   int                 `json:"successful"`
	Failed       int                 `json:"failed"`
	ErrorRate    float64             `json:"error_rate"`
	AverageMS    float64             `json:"average_ms"`
	MinMS        float64             `json:"min_ms"`
	MaxMS        float64             `json:"max_ms"`
	P50MS        float64             `json:"p50_ms"`
	P90MS        float64             `json:"p90_ms"`
	P95MS        float64             `json:"p95_ms"`
	P99MS        float64             `json:"p99_ms"`
	Endpoints    map[string]Endpoint `json:"endpoints"`
	Thresholds   []ThresholdResult   `json:"thresholds"`
}

type Endpoint struct {
	Count     int     `json:"count"`
	Failed    int     `json:"failed"`
	AverageMS float64 `json:"average_ms"`
	P95MS     float64 `json:"p95_ms"`
}

type ThresholdResult struct {
	Name   string  `json:"name"`
	Limit  float64 `json:"limit"`
	Actual float64 `json:"actual"`
	Passed bool    `json:"passed"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	ClassName string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

func ParseJTL(path string, thresholds spec.Thresholds) (*Summary, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &Summary{Endpoints: map[string]Endpoint{}}, nil
	}

	header := map[string]int{}
	for i, name := range rows[0] {
		header[name] = i
	}

	var elapsedValues []float64
	totalElapsed := 0.0
	minElapsed := math.Inf(1)
	maxElapsed := 0.0
	endpointValues := map[string][]float64{}
	endpointFailed := map[string]int{}
	summary := &Summary{Endpoints: map[string]Endpoint{}}

	for _, row := range rows[1:] {
		elapsed := fieldFloat(row, header, "elapsed")
		label := field(row, header, "label")
		if label == "" {
			label = "unknown"
		}
		success := strings.EqualFold(field(row, header, "success"), "true")
		summary.TotalSamples++
		if success {
			summary.Successful++
		} else {
			summary.Failed++
			endpointFailed[label]++
		}
		elapsedValues = append(elapsedValues, elapsed)
		endpointValues[label] = append(endpointValues[label], elapsed)
		totalElapsed += elapsed
		if elapsed < minElapsed {
			minElapsed = elapsed
		}
		if elapsed > maxElapsed {
			maxElapsed = elapsed
		}
	}

	if summary.TotalSamples == 0 {
		return summary, nil
	}
	sort.Float64s(elapsedValues)
	summary.ErrorRate = float64(summary.Failed) / float64(summary.TotalSamples) * 100
	summary.AverageMS = totalElapsed / float64(summary.TotalSamples)
	summary.MinMS = minElapsed
	summary.MaxMS = maxElapsed
	summary.P50MS = percentile(elapsedValues, 50)
	summary.P90MS = percentile(elapsedValues, 90)
	summary.P95MS = percentile(elapsedValues, 95)
	summary.P99MS = percentile(elapsedValues, 99)

	for label, values := range endpointValues {
		sort.Float64s(values)
		total := 0.0
		for _, value := range values {
			total += value
		}
		summary.Endpoints[label] = Endpoint{
			Count:     len(values),
			Failed:    endpointFailed[label],
			AverageMS: total / float64(len(values)),
			P95MS:     percentile(values, 95),
		}
	}

	summary.Thresholds = EvaluateThresholds(summary, thresholds)
	return summary, nil
}

func EvaluateThresholds(summary *Summary, thresholds spec.Thresholds) []ThresholdResult {
	var results []ThresholdResult
	if thresholds.ErrorRateLT != nil {
		results = append(results, ThresholdResult{
			Name: "error_rate_lt", Limit: *thresholds.ErrorRateLT, Actual: summary.ErrorRate,
			Passed: summary.ErrorRate < *thresholds.ErrorRateLT,
		})
	}
	if thresholds.P95MsLT != nil {
		results = append(results, ThresholdResult{
			Name: "p95_ms_lt", Limit: *thresholds.P95MsLT, Actual: summary.P95MS,
			Passed: summary.P95MS < *thresholds.P95MsLT,
		})
	}
	if thresholds.AvgMsLT != nil {
		results = append(results, ThresholdResult{
			Name: "avg_ms_lt", Limit: *thresholds.AvgMsLT, Actual: summary.AverageMS,
			Passed: summary.AverageMS < *thresholds.AvgMsLT,
		})
	}
	return results
}

func (s *Summary) Passed() bool {
	for _, threshold := range s.Thresholds {
		if !threshold.Passed {
			return false
		}
	}
	return true
}

func WriteAll(summary *Summary, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "summary.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "summary.md"), []byte(RenderMarkdown(summary)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "index.html"), []byte(RenderHTML(summary)), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "junit.xml"), []byte(RenderJUnit(summary)), 0o644)
}

func RenderMarkdown(s *Summary) string {
	var b strings.Builder
	b.WriteString("# JMeter Test Summary\n\n")
	status := "PASS"
	if !s.Passed() {
		status = "FAIL"
	}
	fmt.Fprintf(&b, "- Status: %s\n", status)
	fmt.Fprintf(&b, "- Total samples: %d\n", s.TotalSamples)
	fmt.Fprintf(&b, "- Successful: %d\n", s.Successful)
	fmt.Fprintf(&b, "- Failed: %d\n", s.Failed)
	fmt.Fprintf(&b, "- Error rate: %.2f%%\n", s.ErrorRate)
	fmt.Fprintf(&b, "- Average: %.2f ms\n", s.AverageMS)
	fmt.Fprintf(&b, "- p95: %.2f ms\n", s.P95MS)
	fmt.Fprintf(&b, "- p99: %.2f ms\n", s.P99MS)
	if len(s.Thresholds) > 0 {
		b.WriteString("\n## Thresholds\n\n")
		for _, threshold := range s.Thresholds {
			status := "PASS"
			if !threshold.Passed {
				status = "FAIL"
			}
			fmt.Fprintf(&b, "- %s: %s actual %.2f limit %.2f\n", threshold.Name, status, threshold.Actual, threshold.Limit)
		}
	}
	if len(s.Endpoints) > 0 {
		b.WriteString("\n## Endpoints\n\n")
		b.WriteString("| Endpoint | Samples | Failed | Error rate | Average | p95 |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: |\n")
		for _, row := range endpointRows(s) {
			fmt.Fprintf(&b, "| %s | %d | %d | %.2f%% | %.2f ms | %.2f ms |\n",
				escapeMarkdownCell(row.Name), row.Endpoint.Count, row.Endpoint.Failed, endpointErrorRate(row.Endpoint), row.Endpoint.AverageMS, row.Endpoint.P95MS)
		}
	}
	return b.String()
}

func RenderHTML(s *Summary) string {
	status := "PASS"
	if !s.Passed() {
		status = "FAIL"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Loadwright report</title>
  <style>
    :root { color-scheme: light; --ink: #17202a; --muted: #5f6b7a; --line: #d8dee4; --panel: #f6f8fa; --pass: #166534; --fail: #991b1b; }
    body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; color: var(--ink); background: #ffffff; }
    main { max-width: 1120px; margin: 0 auto; padding: 32px 24px 48px; }
    header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 24px; }
    h1 { font-size: 28px; line-height: 1.2; margin: 0; }
    h2 { font-size: 18px; margin: 28px 0 12px; }
    .badge { border-radius: 999px; font-size: 13px; font-weight: 700; padding: 4px 10px; border: 1px solid currentColor; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(156px, 1fr)); gap: 12px; }
    .metric { border: 1px solid var(--line); border-radius: 8px; padding: 14px; background: var(--panel); }
    .metric strong { display: block; font-size: 26px; line-height: 1.1; margin-bottom: 4px; }
    .metric span { color: var(--muted); font-size: 13px; }
    .pass { color: var(--pass); }
    .fail { color: var(--fail); }
    .empty { color: var(--muted); border: 1px solid var(--line); border-radius: 8px; padding: 14px; }
    table { width: 100%%; border-collapse: collapse; margin-top: 8px; font-size: 14px; }
    th, td { border-bottom: 1px solid var(--line); padding: 10px 8px; text-align: left; vertical-align: top; }
    th { color: var(--muted); font-weight: 600; background: #ffffff; }
    td.num, th.num { text-align: right; font-variant-numeric: tabular-nums; }
    .endpoint { max-width: 460px; overflow-wrap: anywhere; }
    @media (max-width: 640px) {
      main { padding: 24px 14px 40px; }
      header { display: block; }
      header .badge { display: inline-block; margin-top: 12px; }
      table { display: block; overflow-x: auto; white-space: nowrap; }
      .endpoint { min-width: 220px; }
    }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>Loadwright report</h1>
      <span class="badge %s">%s</span>
    </header>
    <section aria-label="Overview">
      <div class="grid">
        <div class="metric"><strong>%d</strong><span>Total samples</span></div>
        <div class="metric"><strong>%d</strong><span>Failed</span></div>
        <div class="metric"><strong>%.2f%%</strong><span>Error rate</span></div>
        <div class="metric"><strong>%.2f ms</strong><span>Average</span></div>
        <div class="metric"><strong>%.2f ms</strong><span>p95</span></div>
        <div class="metric"><strong>%.2f ms</strong><span>p99</span></div>
      </div>
    </section>
    %s
    %s
  </main>
</body>
</html>
`, strings.ToLower(status), status, s.TotalSamples, s.Failed, s.ErrorRate, s.AverageMS, s.P95MS, s.P99MS, renderThresholdTable(s), renderEndpointTable(s))
}

func RenderJUnit(s *Summary) string {
	cases := []junitTestCase{junitSamplesCase(s)}
	for _, threshold := range s.Thresholds {
		cases = append(cases, junitThresholdCase(threshold))
	}
	failures := 0
	for _, testCase := range cases {
		if testCase.Failure != nil {
			failures++
		}
	}
	suite := junitTestSuite{
		Name:      "Loadwright",
		Tests:     len(cases),
		Failures:  failures,
		TestCases: cases,
	}
	data, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return ""
	}
	return xml.Header + string(data) + "\n"
}

func junitSamplesCase(s *Summary) junitTestCase {
	testCase := junitTestCase{
		ClassName: "loadwright.samples",
		Name:      "samples",
		Time:      fmt.Sprintf("%.3f", s.AverageMS/1000),
		SystemOut: fmt.Sprintf("total=%d successful=%d failed=%d error_rate=%.2f%% p95_ms=%.2f", s.TotalSamples, s.Successful, s.Failed, s.ErrorRate, s.P95MS),
	}
	if s.Failed > 0 {
		message := fmt.Sprintf("%d of %d samples failed", s.Failed, s.TotalSamples)
		testCase.Failure = &junitFailure{
			Message: message,
			Type:    "sample_failure",
			Body:    message,
		}
	}
	return testCase
}

func junitThresholdCase(threshold ThresholdResult) junitTestCase {
	testCase := junitTestCase{
		ClassName: "loadwright.thresholds",
		Name:      threshold.Name,
		Time:      "0",
		SystemOut: fmt.Sprintf("actual=%.2f limit=%.2f", threshold.Actual, threshold.Limit),
	}
	if !threshold.Passed {
		message := fmt.Sprintf("%s failed: actual %.2f must be less than %.2f", threshold.Name, threshold.Actual, threshold.Limit)
		testCase.Failure = &junitFailure{
			Message: message,
			Type:    "threshold_failure",
			Body:    message,
		}
	}
	return testCase
}

func renderThresholdTable(s *Summary) string {
	if len(s.Thresholds) == 0 {
		return `<section aria-labelledby="thresholds"><h2 id="thresholds">Thresholds</h2><p class="empty">No thresholds configured.</p></section>`
	}
	var b strings.Builder
	b.WriteString(`<section aria-labelledby="thresholds"><h2 id="thresholds">Thresholds</h2><table><thead><tr><th>Name</th><th class="num">Actual</th><th class="num">Limit</th><th>Status</th></tr></thead><tbody>`)
	for _, threshold := range s.Thresholds {
		status := "PASS"
		className := "pass"
		if !threshold.Passed {
			status = "FAIL"
			className = "fail"
		}
		fmt.Fprintf(&b, `<tr><td>%s</td><td class="num">%.2f</td><td class="num">%.2f</td><td class="%s">%s</td></tr>`, html.EscapeString(threshold.Name), threshold.Actual, threshold.Limit, className, status)
	}
	b.WriteString("</tbody></table></section>")
	return b.String()
}

func renderEndpointTable(s *Summary) string {
	if len(s.Endpoints) == 0 {
		return `<section aria-labelledby="endpoints"><h2 id="endpoints">Endpoints</h2><p class="empty">No endpoint samples found.</p></section>`
	}
	var b strings.Builder
	b.WriteString(`<section aria-labelledby="endpoints"><h2 id="endpoints">Endpoints</h2><table><thead><tr><th>Endpoint</th><th class="num">Samples</th><th class="num">Failed</th><th class="num">Error rate</th><th class="num">Average</th><th class="num">p95</th></tr></thead><tbody>`)
	for _, row := range endpointRows(s) {
		fmt.Fprintf(&b, `<tr><td class="endpoint">%s</td><td class="num">%d</td><td class="num">%d</td><td class="num">%.2f%%</td><td class="num">%.2f ms</td><td class="num">%.2f ms</td></tr>`,
			html.EscapeString(row.Name), row.Endpoint.Count, row.Endpoint.Failed, endpointErrorRate(row.Endpoint), row.Endpoint.AverageMS, row.Endpoint.P95MS)
	}
	b.WriteString("</tbody></table></section>")
	return b.String()
}

type endpointRow struct {
	Name     string
	Endpoint Endpoint
}

func endpointRows(s *Summary) []endpointRow {
	rows := make([]endpointRow, 0, len(s.Endpoints))
	for name, endpoint := range s.Endpoints {
		rows = append(rows, endpointRow{Name: name, Endpoint: endpoint})
	}
	sort.Slice(rows, func(i, j int) bool {
		left := rows[i].Endpoint
		right := rows[j].Endpoint
		if left.Failed != right.Failed {
			return left.Failed > right.Failed
		}
		if left.P95MS != right.P95MS {
			return left.P95MS > right.P95MS
		}
		if left.AverageMS != right.AverageMS {
			return left.AverageMS > right.AverageMS
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func endpointErrorRate(endpoint Endpoint) float64 {
	if endpoint.Count == 0 {
		return 0
	}
	return float64(endpoint.Failed) / float64(endpoint.Count) * 100
}

func escapeMarkdownCell(value string) string {
	return strings.ReplaceAll(value, "|", `\|`)
}

func percentile(sorted []float64, pct float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := (pct / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sorted[lower]
	}
	weight := rank - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func field(row []string, header map[string]int, name string) string {
	index, ok := header[name]
	if !ok || index >= len(row) {
		return ""
	}
	return row[index]
}

func fieldFloat(row []string, header map[string]int, name string) float64 {
	parsed, _ := strconv.ParseFloat(field(row, header, name), 64)
	return parsed
}
