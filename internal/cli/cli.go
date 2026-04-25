package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/devaryakjha/loadwright/internal/har"
	"github.com/devaryakjha/loadwright/internal/jmx"
	"github.com/devaryakjha/loadwright/internal/openapi"
	"github.com/devaryakjha/loadwright/internal/postman"
	"github.com/devaryakjha/loadwright/internal/report"
	"github.com/devaryakjha/loadwright/internal/runtime"
	"github.com/devaryakjha/loadwright/internal/spec"
	"github.com/devaryakjha/loadwright/internal/version"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	case "version", "--version":
		fmt.Fprintln(stdout, version.String())
		return 0
	case "doctor":
		return doctor(args[1:], stdout, stderr)
	case "init":
		return initSpec(args[1:], stdout, stderr)
	case "validate":
		return validate(args[1:], stdout, stderr)
	case "compile":
		return compile(args[1:], stdout, stderr)
	case "run":
		return run(args[1:], stdout, stderr)
	case "report":
		return reportCommand(args[1:], stdout, stderr)
	case "compare":
		return compareCommand(args[1:], stdout, stderr)
	case "import":
		return importCommand(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `Loadwright: Docker-first, spec-driven JMeter automation

Usage:
  loadwright doctor [--deep] [--image justb4/jmeter:latest]
  loadwright version
  loadwright init [path]
  loadwright import openapi <openapi.yaml|openapi.json> [-o loadwright.yaml] [--base-url https://api.example.com]
  loadwright import postman <collection.json> [-o loadwright.yaml] [--base-url https://api.example.com]
  loadwright import har <capture.har> [-o loadwright.yaml] [--base-url https://api.example.com]
  loadwright validate <spec.yaml> [--env-file .env.test]
  loadwright compile <spec.yaml> [-o tests/name.jmx] [--env-file .env.test]
  loadwright run <spec.yaml|test.jmx> [--out-dir results/run] [--env-file .env.test] [--ci]
  loadwright report <results.jtl> [--out-dir results/report] [--error-rate-lt 1] [--p95-ms-lt 3000] [--avg-ms-lt 1000] [--ci]
  loadwright compare <baseline-summary.json> <candidate-summary.json> [-o comparison.md]

Commands:
  doctor    Check local Docker/JMeter prerequisites
  version   Print version information
  init      Write a starter YAML spec
  import    Convert supported source formats to Loadwright specs
  validate  Validate a YAML spec without compiling or running JMeter
  compile   Compile a YAML spec to JMeter JMX
  run       Run a YAML spec or existing JMX through Dockerized JMeter
  report    Generate reports from an existing JMeter JTL file
  compare   Compare two Loadwright summary.json files`)
}

func doctor(args []string, stdout io.Writer, stderr io.Writer) int {
	deep, image, err := parseDoctorArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	failed := false
	for _, check := range runtime.Doctor(runtime.DoctorOptions{Deep: deep, Image: image}) {
		status := "PASS"
		if !check.Passed {
			status = "FAIL"
			failed = true
		}
		fmt.Fprintf(stdout, "%-14s %s %s\n", status, check.Name, check.Message)
	}
	if failed {
		return 1
	}
	return 0
}

func parseDoctorArgs(args []string) (deep bool, image string, err error) {
	image = runtime.DefaultJMeterImage
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--deep":
			deep = true
		case arg == "--image":
			i++
			if i >= len(args) {
				return false, "", fmt.Errorf("%s requires a value", arg)
			}
			image = args[i]
		case strings.HasPrefix(arg, "--image="):
			image = strings.TrimPrefix(arg, "--image=")
		default:
			return false, "", fmt.Errorf("unknown doctor option: %s", arg)
		}
	}
	return deep, image, nil
}

func initSpec(args []string, stdout io.Writer, stderr io.Writer) int {
	path := "loadwright.yaml"
	if len(args) > 0 {
		path = args[0]
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", path)
		return 1
	}
	if err := os.WriteFile(path, []byte(starterSpec), 0o644); err != nil {
		fmt.Fprintf(stderr, "write starter spec: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created %s\n", path)
	return 0
}

func compile(args []string, stdout io.Writer, stderr io.Writer) int {
	specPath, output, envFile, err := parseCompileArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	loaded, err := loadResolvedSpec(specPath, envFile)
	if err != nil {
		writeSpecError(stderr, err)
		return 1
	}
	out := output
	if out == "" {
		out = filepath.Join("tests", jmx.SafeName(loaded.Name)+".jmx")
	}
	if err := jmx.Compile(loaded, out); err != nil {
		fmt.Fprintf(stderr, "compile failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", out)
	return 0
}

func validate(args []string, stdout io.Writer, stderr io.Writer) int {
	specPath, envFile, err := parseValidateArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	loaded, err := loadResolvedSpec(specPath, envFile)
	if err != nil {
		writeSpecError(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "valid spec: %s (%d request", loaded.Name, len(loaded.Requests))
	if len(loaded.Requests) != 1 {
		fmt.Fprint(stdout, "s")
	}
	fmt.Fprintln(stdout, ")")
	return 0
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	input, outputDir, envFile, ci, image, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	startedAt := time.Now().UTC()
	updateLatest := outputDir == ""
	runID := startedAt.Format("20060102-150405")
	if outputDir == "" {
		outputDir = filepath.Join("results", runID)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "create output dir: %v\n", err)
		return 1
	}

	var thresholds spec.Thresholds
	jmxPath := input
	generatedJMX := false
	workDir := "."
	if isYAML(input) {
		loaded, err := loadResolvedSpec(input, envFile)
		if err != nil {
			writeSpecError(stderr, err)
			return 1
		}
		thresholds = loaded.Thresholds
		jmxPath = filepath.Join(outputDir, jmx.SafeName(loaded.Name)+".jmx")
		if err := jmx.Compile(loaded, jmxPath); err != nil {
			fmt.Fprintf(stderr, "compile failed: %v\n", err)
			return 1
		}
		generatedJMX = true
		workDir = outputDir
		fmt.Fprintf(stdout, "compiled %s\n", jmxPath)
	}

	jtlName := "results.jtl"
	err = runtime.RunJMeter(runtime.RunOptions{
		Image:      image,
		WorkDir:    workDir,
		JMXPath:    jmxPath,
		ResultsDir: outputDir,
		JTLName:    jtlName,
	})
	if err != nil {
		fmt.Fprintf(stderr, "jmeter run failed: %v\n", err)
		return 1
	}

	summary, err := report.ParseJTL(filepath.Join(outputDir, jtlName), thresholds)
	if err != nil {
		fmt.Fprintf(stderr, "parse report failed: %v\n", err)
		return 1
	}
	if err := report.WriteAll(summary, outputDir); err != nil {
		fmt.Fprintf(stderr, "write report failed: %v\n", err)
		return 1
	}
	finishedAt := time.Now().UTC()
	if err := writeRunManifest(runManifest{
		RunID:        filepath.Base(outputDir),
		Input:        filepath.ToSlash(input),
		InputType:    runInputType(input),
		JMX:          filepath.ToSlash(jmxPath),
		GeneratedJMX: generatedJMX,
		Image:        image,
		CI:           ci,
		StartedAt:    startedAt.Format(time.RFC3339),
		FinishedAt:   finishedAt.Format(time.RFC3339),
		Artifacts: runArtifacts{
			ResultsJTL:  filepath.ToSlash(filepath.Join(outputDir, jtlName)),
			SummaryJSON: filepath.ToSlash(filepath.Join(outputDir, "summary.json")),
			SummaryMD:   filepath.ToSlash(filepath.Join(outputDir, "summary.md")),
			ReportHTML:  filepath.ToSlash(filepath.Join(outputDir, "index.html")),
			JUnitXML:    filepath.ToSlash(filepath.Join(outputDir, "junit.xml")),
		},
	}, outputDir); err != nil {
		fmt.Fprintf(stderr, "write run metadata failed: %v\n", err)
		return 1
	}
	if updateLatest {
		if err := writeLatestRun("results", outputDir, finishedAt); err != nil {
			fmt.Fprintf(stderr, "warning: update latest run pointer: %v\n", err)
		}
	}
	fmt.Fprintf(stdout, "report %s\n", filepath.Join(outputDir, "index.html"))
	if ci && !summary.Passed() {
		fmt.Fprintln(stderr, "thresholds failed")
		return 1
	}
	return 0
}

type runManifest struct {
	RunID        string       `json:"run_id"`
	Input        string       `json:"input"`
	InputType    string       `json:"input_type"`
	JMX          string       `json:"jmx"`
	GeneratedJMX bool         `json:"generated_jmx"`
	Image        string       `json:"image"`
	CI           bool         `json:"ci"`
	StartedAt    string       `json:"started_at"`
	FinishedAt   string       `json:"finished_at"`
	Artifacts    runArtifacts `json:"artifacts"`
}

type runArtifacts struct {
	ResultsJTL  string `json:"results_jtl"`
	SummaryJSON string `json:"summary_json"`
	SummaryMD   string `json:"summary_md"`
	ReportHTML  string `json:"report_html"`
	JUnitXML    string `json:"junit_xml"`
}

func writeRunManifest(manifest runManifest, outputDir string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(outputDir, "run.json"), data, 0o644)
}

func runInputType(path string) string {
	if isYAML(path) {
		return "yaml"
	}
	return "jmx"
}

type latestRun struct {
	RunID     string `json:"run_id"`
	RunDir    string `json:"run_dir"`
	Report    string `json:"report"`
	UpdatedAt string `json:"updated_at"`
}

func writeLatestRun(resultsRoot string, outputDir string, updatedAt time.Time) error {
	if err := os.MkdirAll(resultsRoot, 0o755); err != nil {
		return err
	}
	runDir := filepath.Clean(outputDir)
	relativeRunDir, err := filepath.Rel(resultsRoot, runDir)
	if err == nil && !strings.HasPrefix(relativeRunDir, "..") && relativeRunDir != "." {
		runDir = filepath.ToSlash(filepath.Join(resultsRoot, relativeRunDir))
	}
	metadata := latestRun{
		RunID:     filepath.Base(outputDir),
		RunDir:    filepath.ToSlash(runDir),
		Report:    filepath.ToSlash(filepath.Join(runDir, "index.html")),
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	latestPath := filepath.Join(resultsRoot, "latest.json")
	if err := os.WriteFile(latestPath, data, 0o644); err != nil {
		return err
	}
	writeLatestSymlink(resultsRoot, outputDir)
	return nil
}

func writeLatestSymlink(resultsRoot string, outputDir string) {
	latestPath := filepath.Join(resultsRoot, "latest")
	if info, err := os.Lstat(latestPath); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return
		}
		if err := os.Remove(latestPath); err != nil {
			return
		}
	}
	target, err := filepath.Rel(resultsRoot, outputDir)
	if err != nil || strings.HasPrefix(target, "..") || target == "." {
		return
	}
	_ = os.Symlink(target, latestPath)
}

func reportCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	jtlPath, outputDir, thresholds, ci, err := parseReportArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if outputDir == "" {
		outputDir = filepath.Dir(jtlPath)
		if outputDir == "" {
			outputDir = "."
		}
	}
	summary, err := report.ParseJTL(jtlPath, thresholds)
	if err != nil {
		fmt.Fprintf(stderr, "parse report failed: %v\n", err)
		return 1
	}
	if err := report.WriteAll(summary, outputDir); err != nil {
		fmt.Fprintf(stderr, "write report failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "report %s\n", filepath.Join(outputDir, "index.html"))
	if ci && !summary.Passed() {
		fmt.Fprintln(stderr, "thresholds failed")
		return 1
	}
	return 0
}

func compareCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	baselinePath, candidatePath, output, err := parseCompareArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	baseline, err := report.LoadSummaryFile(baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "load baseline summary failed: %v\n", err)
		return 1
	}
	candidate, err := report.LoadSummaryFile(candidatePath)
	if err != nil {
		fmt.Fprintf(stderr, "load candidate summary failed: %v\n", err)
		return 1
	}
	markdown := report.RenderComparisonMarkdown(report.CompareSummaries(baseline, candidate))
	if output == "" {
		fmt.Fprint(stdout, markdown)
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(stderr, "create comparison output dir: %v\n", err)
		return 1
	}
	if err := os.WriteFile(output, []byte(markdown), 0o644); err != nil {
		fmt.Fprintf(stderr, "write comparison failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", output)
	return 0
}

func importCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "import requires a source type")
		return 2
	}
	switch args[0] {
	case "openapi":
		return importOpenAPI(args[1:], stdout, stderr)
	case "postman":
		return importPostman(args[1:], stdout, stderr)
	case "har":
		return importHAR(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unsupported import source: %s\n", args[0])
		return 2
	}
}

func importOpenAPI(args []string, stdout io.Writer, stderr io.Writer) int {
	input, output, baseURL, err := parseImportOpenAPIArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	imported, err := openapi.ImportFile(input, openapi.Options{BaseURL: baseURL})
	if err != nil {
		fmt.Fprintf(stderr, "import failed: %v\n", err)
		return 1
	}
	if output == "" {
		output = "loadwright.yaml"
	}
	if err := spec.WriteFile(imported, output); err != nil {
		fmt.Fprintf(stderr, "write spec failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", output)
	return 0
}

func importPostman(args []string, stdout io.Writer, stderr io.Writer) int {
	input, output, baseURL, err := parseImportPostmanArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	result, err := postman.ImportFile(input, postman.Options{BaseURL: baseURL})
	if err != nil {
		fmt.Fprintf(stderr, "import failed: %v\n", err)
		return 1
	}
	if output == "" {
		output = "loadwright.yaml"
	}
	if err := spec.WriteFile(result.Spec, output); err != nil {
		fmt.Fprintf(stderr, "write spec failed: %v\n", err)
		return 1
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	fmt.Fprintf(stdout, "wrote %s\n", output)
	return 0
}

func importHAR(args []string, stdout io.Writer, stderr io.Writer) int {
	input, output, baseURL, err := parseImportHARArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	result, err := har.ImportFile(input, har.Options{BaseURL: baseURL})
	if err != nil {
		fmt.Fprintf(stderr, "import failed: %v\n", err)
		return 1
	}
	if output == "" {
		output = "loadwright.yaml"
	}
	if err := spec.WriteFile(result.Spec, output); err != nil {
		fmt.Fprintf(stderr, "write spec failed: %v\n", err)
		return 1
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	fmt.Fprintf(stdout, "wrote %s\n", output)
	return 0
}

func loadResolvedSpec(path string, envFile string) (*spec.Spec, error) {
	env, err := spec.LoadEnvFile(envFile)
	if err != nil {
		return nil, err
	}
	loaded, err := spec.LoadFileUnresolved(path)
	if err != nil {
		return nil, err
	}
	return loaded.Resolve(env, spec.WithBaseDir(filepath.Dir(path)))
}

func writeSpecError(w io.Writer, err error) {
	lines := errorLines(err)
	if len(lines) == 0 {
		fmt.Fprintf(w, "invalid spec: %v\n", err)
		return
	}
	fmt.Fprintln(w, "invalid spec:")
	for _, line := range lines {
		fmt.Fprintf(w, "  - %s\n", line)
	}
}

func errorLines(err error) []string {
	if err == nil {
		return nil
	}
	type multiUnwrapper interface {
		Unwrap() []error
	}
	if joined, ok := err.(multiUnwrapper); ok {
		var lines []string
		for _, child := range joined.Unwrap() {
			lines = append(lines, errorLines(child)...)
		}
		return lines
	}
	return []string{err.Error()}
}

func parseCompileArgs(args []string) (specPath string, output string, envFile string, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--out":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("%s requires a value", arg)
			}
			output = args[i]
		case strings.HasPrefix(arg, "-o="):
			output = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "--out="):
			output = strings.TrimPrefix(arg, "--out=")
		case arg == "--env-file":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("%s requires a value", arg)
			}
			envFile = args[i]
		case strings.HasPrefix(arg, "--env-file="):
			envFile = strings.TrimPrefix(arg, "--env-file=")
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return "", "", "", fmt.Errorf("compile requires exactly one spec path")
	}
	return positional[0], output, envFile, nil
}

func parseValidateArgs(args []string) (specPath string, envFile string, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--env-file":
			i++
			if i >= len(args) {
				return "", "", fmt.Errorf("%s requires a value", arg)
			}
			envFile = args[i]
		case strings.HasPrefix(arg, "--env-file="):
			envFile = strings.TrimPrefix(arg, "--env-file=")
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return "", "", fmt.Errorf("validate requires exactly one spec path")
	}
	return positional[0], envFile, nil
}

func parseRunArgs(args []string) (input string, outputDir string, envFile string, ci bool, image string, err error) {
	image = runtime.DefaultJMeterImage
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--ci":
			ci = true
		case arg == "--out-dir":
			i++
			if i >= len(args) {
				return "", "", "", false, "", fmt.Errorf("%s requires a value", arg)
			}
			outputDir = args[i]
		case strings.HasPrefix(arg, "--out-dir="):
			outputDir = strings.TrimPrefix(arg, "--out-dir=")
		case arg == "--image":
			i++
			if i >= len(args) {
				return "", "", "", false, "", fmt.Errorf("%s requires a value", arg)
			}
			image = args[i]
		case strings.HasPrefix(arg, "--image="):
			image = strings.TrimPrefix(arg, "--image=")
		case arg == "--env-file":
			i++
			if i >= len(args) {
				return "", "", "", false, "", fmt.Errorf("%s requires a value", arg)
			}
			envFile = args[i]
		case strings.HasPrefix(arg, "--env-file="):
			envFile = strings.TrimPrefix(arg, "--env-file=")
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return "", "", "", false, "", fmt.Errorf("run requires exactly one spec or JMX path")
	}
	return positional[0], outputDir, envFile, ci, image, nil
}

func parseReportArgs(args []string) (jtlPath string, outputDir string, thresholds spec.Thresholds, ci bool, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--ci":
			ci = true
		case arg == "--out-dir":
			i++
			if i >= len(args) {
				return "", "", thresholds, false, fmt.Errorf("%s requires a value", arg)
			}
			outputDir = args[i]
		case strings.HasPrefix(arg, "--out-dir="):
			outputDir = strings.TrimPrefix(arg, "--out-dir=")
		case arg == "--error-rate-lt":
			i++
			if i >= len(args) {
				return "", "", thresholds, false, fmt.Errorf("%s requires a value", arg)
			}
			value, parseErr := parseThresholdValue(arg, args[i])
			if parseErr != nil {
				return "", "", thresholds, false, parseErr
			}
			thresholds.ErrorRateLT = &value
		case strings.HasPrefix(arg, "--error-rate-lt="):
			value, parseErr := parseThresholdValue("--error-rate-lt", strings.TrimPrefix(arg, "--error-rate-lt="))
			if parseErr != nil {
				return "", "", thresholds, false, parseErr
			}
			thresholds.ErrorRateLT = &value
		case arg == "--p95-ms-lt":
			i++
			if i >= len(args) {
				return "", "", thresholds, false, fmt.Errorf("%s requires a value", arg)
			}
			value, parseErr := parseThresholdValue(arg, args[i])
			if parseErr != nil {
				return "", "", thresholds, false, parseErr
			}
			thresholds.P95MsLT = &value
		case strings.HasPrefix(arg, "--p95-ms-lt="):
			value, parseErr := parseThresholdValue("--p95-ms-lt", strings.TrimPrefix(arg, "--p95-ms-lt="))
			if parseErr != nil {
				return "", "", thresholds, false, parseErr
			}
			thresholds.P95MsLT = &value
		case arg == "--avg-ms-lt":
			i++
			if i >= len(args) {
				return "", "", thresholds, false, fmt.Errorf("%s requires a value", arg)
			}
			value, parseErr := parseThresholdValue(arg, args[i])
			if parseErr != nil {
				return "", "", thresholds, false, parseErr
			}
			thresholds.AvgMsLT = &value
		case strings.HasPrefix(arg, "--avg-ms-lt="):
			value, parseErr := parseThresholdValue("--avg-ms-lt", strings.TrimPrefix(arg, "--avg-ms-lt="))
			if parseErr != nil {
				return "", "", thresholds, false, parseErr
			}
			thresholds.AvgMsLT = &value
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return "", "", thresholds, false, fmt.Errorf("report requires exactly one JTL path")
	}
	return positional[0], outputDir, thresholds, ci, nil
}

func parseCompareArgs(args []string) (baselinePath string, candidatePath string, output string, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--out":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("%s requires a value", arg)
			}
			output = args[i]
		case strings.HasPrefix(arg, "-o="):
			output = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "--out="):
			output = strings.TrimPrefix(arg, "--out=")
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 2 {
		return "", "", "", fmt.Errorf("compare requires exactly two summary paths")
	}
	return positional[0], positional[1], output, nil
}

func parseThresholdValue(flag string, raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative number", flag)
	}
	return value, nil
}

func parseImportOpenAPIArgs(args []string) (input string, output string, baseURL string, err error) {
	return parseImportArgs("openapi", args)
}

func parseImportPostmanArgs(args []string) (input string, output string, baseURL string, err error) {
	return parseImportArgs("postman", args)
}

func parseImportHARArgs(args []string) (input string, output string, baseURL string, err error) {
	return parseImportArgs("har", args)
}

func parseImportArgs(source string, args []string) (input string, output string, baseURL string, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--out":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("%s requires a value", arg)
			}
			output = args[i]
		case strings.HasPrefix(arg, "-o="):
			output = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "--out="):
			output = strings.TrimPrefix(arg, "--out=")
		case arg == "--base-url":
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("%s requires a value", arg)
			}
			baseURL = args[i]
		case strings.HasPrefix(arg, "--base-url="):
			baseURL = strings.TrimPrefix(arg, "--base-url=")
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return "", "", "", fmt.Errorf("import %s requires exactly one input file", source)
	}
	return positional[0], output, baseURL, nil
}

func isYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

const starterSpec = `name: example-api
target: https://httpbin.org
load:
  users: 5
  ramp_up: 10s
  loops: 3
requests:
  - name: get status
    method: GET
    path: /status/200
    expect:
      status: 200
  - name: post payload
    method: POST
    path: /post
    headers:
      content-type: application/json
    body:
      hello: world
    expect:
      status: 200
thresholds:
  error_rate_lt: 1
  p95_ms_lt: 3000
`
