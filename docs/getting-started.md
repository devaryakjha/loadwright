# Getting Started

Loadwright is a Go CLI for running JMeter from readable YAML specs.

## Install

Install `loadwright` from [GitHub Releases](https://github.com/devaryakjha/loadwright/releases/tag/v0.2.0), with `go install github.com/devaryakjha/loadwright/cmd/loadwright@v0.2.0`, or from a source checkout with `go build -o bin/loadwright ./cmd/loadwright`.

The commands below assume `loadwright` is on your `PATH`. If you built from source and did not install the binary, use `bin/loadwright` instead.

## Check Your Machine

```bash
loadwright version
loadwright doctor
```

For a stronger check that starts JMeter through Docker:

```bash
loadwright doctor --deep
```

## Validate And Compile First

Create a starter spec. Then validate and compile it before running. These commands do not start Docker, so they are the fastest way to verify the spec path:

```bash
loadwright init
loadwright validate loadwright.yaml
loadwright compile loadwright.yaml
```

## Run The Starter Spec

```bash
loadwright run loadwright.yaml --ci
```

The run writes artifacts to `results/<run-id>/`:

- `results.jtl`
- `summary.json`
- `summary.md`
- `index.html`
- `junit.xml`
- `run.json`

Default runs also update `results/latest.json`, and create a best-effort `results/latest` symlink on platforms that support it.

## Compile Without Running

```bash
loadwright compile loadwright.yaml -o tests/example-api.jmx
```

The generated `.jmx` file can be opened in the JMeter GUI or run by JMeter directly.

## Validate Without Running

```bash
loadwright validate loadwright.yaml
```

Validation resolves variables and env files, applies defaults, and reports spec errors without starting Docker or JMeter.

## Env Files

Specs can reference environment values with `${NAME}` and variables with `{{name}}`. From a source checkout, try the included env-file example:

```bash
loadwright validate examples/api/env-file.yaml --env-file examples/api/.env.example
loadwright run examples/api/env-file.yaml --env-file examples/api/.env.example --ci
```

## Create Your Own Spec

If you already used `loadwright init`, edit `loadwright.yaml` and rerun:

```bash
loadwright validate loadwright.yaml
loadwright compile loadwright.yaml
loadwright run loadwright.yaml --ci
```

If any first-run command fails, see [Troubleshooting](troubleshooting.md).
