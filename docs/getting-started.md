# Getting Started

Loadwright is a Go CLI for running JMeter from readable YAML specs.

## Build

```bash
go build -o bin/loadwright ./cmd/loadwright
```

## Check Your Machine

```bash
bin/loadwright doctor
```

For a stronger check that starts JMeter through Docker:

```bash
bin/loadwright doctor --deep
```

## Run The Basic Example

```bash
bin/loadwright run examples/api/basic.yaml --ci
```

The run writes artifacts to `results/<run-id>/`:

- `results.jtl`
- `summary.json`
- `summary.md`
- `index.html`
- `junit.xml`

Default runs also update `results/latest.json`, and create a best-effort `results/latest` symlink on platforms that support it.

## Compile Without Running

```bash
bin/loadwright compile examples/api/basic.yaml -o tests/httpbin-basic.jmx
```

The generated `.jmx` file can be opened in the JMeter GUI or run by JMeter directly.

## Validate Without Running

```bash
bin/loadwright validate examples/api/basic.yaml
```

Validation resolves variables and env files, applies defaults, and reports spec errors without starting Docker or JMeter.

## Env Files

Specs can reference environment values with `${NAME}` and variables with `{{name}}`.

```bash
bin/loadwright validate examples/api/env-file.yaml --env-file examples/api/.env.example
bin/loadwright run examples/api/env-file.yaml --env-file examples/api/.env.example --ci
```
