package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
)

const DefaultJMeterImage = "justb4/jmeter@sha256:088ac52b759a198a5afa5ae13d0a6306e9f2017d71ad140ff57427f6930406f7"

type ErrorKind string

const (
	ErrorDockerUnavailable ErrorKind = "docker unavailable"
	ErrorImagePull         ErrorKind = "image pull failed"
	ErrorJMeterStartup     ErrorKind = "jmeter startup failed"
	ErrorTestExecution     ErrorKind = "test execution failed"
)

type RuntimeError struct {
	Kind     ErrorKind
	Image    string
	Cause    error
	Output   string
	Recovery string
}

func (e *RuntimeError) Error() string {
	if e == nil {
		return ""
	}
	message := string(e.Kind)
	if e.Image != "" {
		message += " for image " + e.Image
	}
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *RuntimeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type RunOptions struct {
	Image      string
	WorkDir    string
	JMXPath    string
	ResultsDir string
	JTLName    string
}

type DoctorOptions struct {
	Image   string
	Deep    bool
	WorkDir string
}

func Doctor(options DoctorOptions) []Check {
	if options.Image == "" {
		options.Image = DefaultJMeterImage
	}
	if options.WorkDir == "" {
		options.WorkDir = "."
	}
	checks := []Check{
		checkCommand("docker", "Docker CLI"),
		checkDocker(),
		checkWritableDir(filepath.Join(options.WorkDir, "tests"), "tests directory"),
		checkWritableDir(filepath.Join(options.WorkDir, "results"), "results directory"),
		checkImage(options.Image),
	}
	if options.Deep {
		checks = append(checks, checkJMeterRuntime(options.Image))
	}
	return checks
}

type Check struct {
	Name    string
	Passed  bool
	Message string
}

func RunJMeter(options RunOptions) error {
	if options.Image == "" {
		options.Image = DefaultJMeterImage
	}
	if options.JTLName == "" {
		options.JTLName = "results.jtl"
	}
	absWorkDir, err := filepath.Abs(options.WorkDir)
	if err != nil {
		return err
	}
	jmxAbs, err := filepath.Abs(options.JMXPath)
	if err != nil {
		return err
	}
	resultsAbs, err := filepath.Abs(options.ResultsDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(resultsAbs, 0o755); err != nil {
		return err
	}
	relJMX, err := filepath.Rel(absWorkDir, jmxAbs)
	if err != nil || relJMX == "." || relJMX == ".." || strings.HasPrefix(relJMX, ".."+string(filepath.Separator)) {
		return errors.New("JMX file must be inside the working directory")
	}
	if err := requireDockerDaemon(options.Image); err != nil {
		return err
	}
	if err := ensureImage(options.Image); err != nil {
		return err
	}
	if err := probeJMeterStartup(options.Image); err != nil {
		return err
	}

	args := []string{
		"run", "--rm",
		"-v", absWorkDir + ":/work",
		"-w", "/work",
		options.Image,
		"-n",
		"-t", filepath.ToSlash(relJMX),
		"-l", "/work/" + filepath.ToSlash(filepath.Join(mustRel(absWorkDir, resultsAbs), options.JTLName)),
	}
	cmd := exec.Command("docker", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()
		return &RuntimeError{
			Kind:     ErrorTestExecution,
			Image:    options.Image,
			Cause:    err,
			Output:   output,
			Recovery: testExecutionRecovery(output),
		}
	}
	return nil
}

func checkCommand(command string, name string) Check {
	if _, err := exec.LookPath(command); err != nil {
		return Check{Name: name, Passed: false, Message: fmt.Sprintf("%s not found in PATH", command)}
	}
	return Check{Name: name, Passed: true, Message: "found"}
}

func checkDocker() Check {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := "Docker CLI is installed, but the Docker daemon is not reachable. Start Docker Desktop or your Docker service, then run `docker version` and retry."
		if detail := strings.TrimSpace(string(output)); detail != "" {
			message += " Docker said: " + oneLine(detail)
		}
		return Check{Name: "Docker daemon", Passed: false, Message: message}
	}
	return Check{Name: "Docker daemon", Passed: true, Message: "server " + stringTrim(output)}
}

func checkWritableDir(path string, name string) Check {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return Check{Name: name, Passed: false, Message: err.Error()}
	}
	probe, err := os.CreateTemp(path, ".loadwright-doctor-*")
	if err != nil {
		return Check{Name: name, Passed: false, Message: "not writable: " + err.Error()}
	}
	probePath := probe.Name()
	_ = probe.Close()
	_ = os.Remove(probePath)
	return Check{Name: name, Passed: true, Message: "writable"}
}

func checkImage(image string) Check {
	cmd := exec.Command("docker", "image", "inspect", image, "--format", "{{.Architecture}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		pull := exec.Command("docker", "pull", image)
		pullOutput, pullErr := pull.CombinedOutput()
		if pullErr != nil {
			message := fmt.Sprintf("could not pull %s. Check network access, registry auth, and the image tag.", image)
			if detail := strings.TrimSpace(string(pullOutput)); detail != "" {
				message += " Docker said: " + oneLine(detail)
			}
			return Check{Name: "JMeter image", Passed: false, Message: message}
		}
		return Check{Name: "JMeter image", Passed: true, Message: image + " pulled successfully"}
	}
	arch := stringTrim(output)
	message := image + " available"
	if arch != "" && arch != goruntime.GOARCH {
		message += fmt.Sprintf(" (%s image on %s host; Docker may use emulation)", arch, goruntime.GOARCH)
	}
	return Check{Name: "JMeter image", Passed: true, Message: message}
}

func checkJMeterRuntime(image string) Check {
	output, err := runJMeterVersion(image)
	if err != nil {
		message := fmt.Sprintf("Docker started %s but JMeter did not report a version.", image)
		if detail := strings.TrimSpace(output); detail != "" {
			message += " Output: " + oneLine(detail)
		}
		return Check{Name: "JMeter runtime", Passed: false, Message: message}
	}
	version := firstVersionLine(output)
	if version == "" {
		version = "started successfully"
	}
	return Check{Name: "JMeter runtime", Passed: true, Message: version}
}

func requireDockerDaemon(image string) error {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return &RuntimeError{
		Kind:     ErrorDockerUnavailable,
		Image:    image,
		Cause:    err,
		Output:   string(output),
		Recovery: "Start Docker Desktop or your Docker service, confirm `docker version` shows a Server version, then retry `loadwright run`.",
	}
}

func ensureImage(image string) error {
	inspect := exec.Command("docker", "image", "inspect", image, "--format", "{{.Id}}")
	if err := inspect.Run(); err == nil {
		return nil
	}
	pull := exec.Command("docker", "pull", image)
	output, err := pull.CombinedOutput()
	if err == nil {
		return nil
	}
	return &RuntimeError{
		Kind:     ErrorImagePull,
		Image:    image,
		Cause:    err,
		Output:   string(output),
		Recovery: "Check your network, registry credentials, and image tag. You can override the runtime with `loadwright run ... --image <image:tag>`.",
	}
}

func probeJMeterStartup(image string) error {
	output, err := runJMeterVersion(image)
	if err == nil && firstVersionLine(output) != "" {
		return nil
	}
	if err == nil {
		err = errors.New("JMeter version was not found in container output")
	}
	return &RuntimeError{
		Kind:     ErrorJMeterStartup,
		Image:    image,
		Cause:    err,
		Output:   output,
		Recovery: "Run `loadwright doctor --deep` for the same image. If this is a custom image, verify it can run `jmeter --version` or `--version` successfully.",
	}
}

func runJMeterVersion(image string) (string, error) {
	cmd := exec.Command("docker", "run", "--rm", image, "--version")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func testExecutionRecovery(output string) string {
	message := "Docker and JMeter started, but the test plan failed during execution. Check the generated jmeter.log in the results directory when available."
	if looksLikeMissingWebSocketPlugin(output) {
		message += " WebSocket specs require an image with the WebSocket Samplers plugin, for example the image built from docker/jmeter/Dockerfile."
	}
	return message
}

func looksLikeMissingWebSocketPlugin(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "cannotresolveclassexception") &&
		strings.Contains(lower, "eu.luminis.jmeter.wssampler")
}

func oneLine(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func firstVersionLine(output string) string {
	versionPattern := regexp.MustCompile(`\b([0-9]+\.[0-9]+(?:\.[0-9]+)?)\b`)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Apache JMeter") || strings.HasPrefix(trimmed, "Version ") {
			return trimmed
		}
		if strings.Contains(trimmed, "____") {
			matches := versionPattern.FindStringSubmatch(trimmed)
			if len(matches) > 1 {
				return "Apache JMeter " + matches[1]
			}
		}
	}
	return ""
}

func mustRel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

func stringTrim(value []byte) string {
	result := string(value)
	for len(result) > 0 && (result[len(result)-1] == '\n' || result[len(result)-1] == '\r' || result[len(result)-1] == ' ') {
		result = result[:len(result)-1]
	}
	return result
}
