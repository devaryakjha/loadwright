# Loadwright

[![CI](https://github.com/devaryakjha/loadwright/actions/workflows/ci.yml/badge.svg)](https://github.com/devaryakjha/loadwright/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/devaryakjha/loadwright?sort=semver)](https://github.com/devaryakjha/loadwright/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/devaryakjha/loadwright)](go.mod)
[![License](https://img.shields.io/github/license/devaryakjha/loadwright)](LICENSE)

Docker-first, spec-driven JMeter automation.

Loadwright turns readable YAML specs into portable JMeter `.jmx` test plans, runs them through Dockerized JMeter, and emits JSON, Markdown, HTML, and JUnit reports for local development and CI.

It is not a new load-testing engine. It is a small automation layer that keeps JMeter compatibility while making common API load-test workflows easier to review, run, and ship.

## Project Status

Loadwright is at `v0.1.0`. It is usable for HTTP API load-test workflows and CI smoke/performance checks, but the public API and YAML spec may still evolve before `v1.0.0`.

The current development scope is intentionally focused: HTTP requests, Dockerized JMeter execution, OpenAPI/Postman/HAR bootstrapping, CSV data, thresholds, and reports. WebSocket support, plugin management, distributed runners, and AI-assisted workflows are planned later.

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

Current `v0.1.0` scope: HTTP API load tests. See [docs/limitations.md](docs/limitations.md) for known limits.

## Install From Source

Requires Go 1.22+ and Docker.

```bash
go build -o bin/loadwright ./cmd/loadwright
```

## Quickstart

Create a starter spec:

```bash
bin/loadwright init
```

Or use the included example:

```bash
bin/loadwright doctor
bin/loadwright validate examples/api/basic.yaml
bin/loadwright compile examples/api/basic.yaml -o tests/httpbin-basic.jmx
bin/loadwright run examples/api/basic.yaml --ci
```

Check local prerequisites:

```bash
bin/loadwright doctor
bin/loadwright doctor --deep
```

Reports are written to `results/<run-id>/`:

- `results.jtl`
- `summary.json`
- `summary.md`
- `index.html`
- `junit.xml`

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
```

`doctor --deep` runs the configured JMeter Docker image and verifies that JMeter starts.

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
