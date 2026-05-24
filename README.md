# Loadwright

[![CI](https://github.com/devaryakjha/loadwright/actions/workflows/ci.yml/badge.svg)](https://github.com/devaryakjha/loadwright/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/devaryakjha/loadwright?sort=semver)](https://github.com/devaryakjha/loadwright/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/devaryakjha/loadwright)](go.mod)
[![License](https://img.shields.io/github/license/devaryakjha/loadwright)](LICENSE)

Docker-first, spec-driven JMeter automation.

Loadwright turns readable YAML specs into portable JMeter `.jmx` test plans, runs them through Dockerized JMeter, and emits JSON, Markdown, HTML, and JUnit reports for local development and CI.

It is not a new load-testing engine. It is a small automation layer that keeps JMeter compatibility while making common API load-test workflows easier to review, run, and ship.

## Project Status

Loadwright is at `v0.2.0`. It is usable for HTTP API, WebSocket API, and CI smoke/performance checks, but the public API and YAML spec may still evolve before `v1.0.0`.

The current development scope is intentionally focused: HTTP requests, WebSocket requests, JSON/text/urlencoded/multipart bodies, Dockerized JMeter execution, OpenAPI/Postman/HAR bootstrapping, CSV data, thresholds, and reports. Automated plugin management, distributed runners, and AI-assisted workflows are planned later.

## Why This Exists

JMeter is powerful, but the day-to-day workflow can be awkward: JMX XML, local Java/JMeter setup, Docker wiring, plugin handling, reports, and CI thresholds. Loadwright keeps JMeter compatibility while giving teams a small CLI and reviewable specs.

Use Loadwright when you want:

- a readable YAML source of truth for load tests
- Dockerized JMeter runs without local JMeter setup
- JSON, Markdown, HTML, and JUnit summaries
- CI pass/fail thresholds
- OpenAPI-to-spec bootstrapping for simple API tests
- Postman-collection-to-spec bootstrapping for common API workflows
- HAR-to-spec bootstrapping from browser/API traffic captures
- future optional AI assistance without depending on AI for normal runs

Current `v0.2.0` scope: API load tests (HTTP and WebSocket). See [docs/limitations.md](docs/limitations.md) for known limits.

## Install

Install the latest release binary from [GitHub Releases](https://github.com/devaryakjha/loadwright/releases), or use Go:

```bash
go install github.com/devaryakjha/loadwright/cmd/loadwright@v0.2.0
```

From a source checkout:

```bash
go build -o bin/loadwright ./cmd/loadwright
```

Docker is required for `loadwright run` and `loadwright doctor --deep`. It is not required for `init`, `validate`, `compile`, `import`, or `report`.

## Quickstart

Check the CLI and local prerequisites:

```bash
loadwright version
loadwright doctor
```

Create a starter spec, then validate and compile it without starting Docker:

```bash
loadwright init
loadwright validate loadwright.yaml
loadwright compile loadwright.yaml
```

Run the starter spec through Dockerized JMeter:

```bash
loadwright doctor --deep
loadwright run loadwright.yaml --ci
```

Reports are written to `results/<run-id>/`:

- `results.jtl`
- `summary.json`
- `summary.md`
- `index.html`
- `junit.xml`
- `run.json`

Default runs also update `results/latest.json` so the newest report can be found without copying the timestamped run ID.

From a source checkout, you can also run the included examples:

```bash
loadwright validate examples/api/basic.yaml
loadwright run examples/api/basic.yaml --ci
```

## Example Spec

```yaml
name: httpbin-basic
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
thresholds:
  error_rate_lt: 1
  p95_ms_lt: 3000
```

See [docs/spec-reference.md](docs/spec-reference.md) for the current spec format.

More docs:

- [Getting started](docs/getting-started.md)
- [Install](docs/install.md)
- [Examples](docs/examples.md)
- [Troubleshooting](docs/troubleshooting.md)
- [OpenAPI import](docs/openapi-import.md)
- [Postman import](docs/postman-import.md)
- [HAR import](docs/har-import.md)
- [Data sources](docs/data-sources.md)
- [CI](docs/ci.md)
- [Reports](docs/reports.md)
- [Testing](docs/testing.md)
- [Release checklist](docs/release.md)
- [Compatibility](docs/compatibility.md)
- [Limitations](docs/limitations.md)

## Commands

```bash
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
```

`doctor --deep` runs the configured JMeter Docker image and verifies that JMeter starts.

For WebSocket specs, build the bundled plugin image first, then pass it with `--image`:

```bash
docker build -t loadwright/jmeter-websocket:latest -f docker/jmeter/Dockerfile .
bin/loadwright run examples/api/websocket-multi.yaml --ci --image loadwright/jmeter-websocket:latest
```

The `docker/jmeter/Dockerfile` extends `justb4/jmeter:latest` (the same base used for all HTTP runs) and adds the [WebSocket Samplers](https://github.com/ptrd/jmeter-websocket-samplers) plugin.

## Roadmap

See [ROADMAP.md](ROADMAP.md). The short version:

- make the deterministic Go CLI excellent first
- add broader import support next
- add WebSocket/plugin automation
- add optional AI later for generating, explaining, and improving specs

## Development

```bash
go test ./...
go vet ./...
```

## Releases

Tagged releases are built with GoReleaser and publish cross-platform binaries plus checksums. See [docs/install.md](docs/install.md).

## Credits

The initial experimental prototype that led to this project was built by [Jaykumar Gori](https://github.com/jaykumargori). The public OSS implementation is the Go CLI in this repository.

## License

Apache-2.0.
