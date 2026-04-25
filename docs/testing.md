# Testing

Loadwright keeps tests close to the behavior users depend on.

## Local Checks

```bash
go test ./...
go vet ./...
go build -o bin/loadwright ./cmd/loadwright
```

Compile every runnable example:

```bash
mkdir -p /tmp/loadwright-examples
find examples -name '*.yaml' -not -path 'examples/openapi/*' -print | sort | while read -r spec; do
  if grep -q '\${' "$spec"; then
    bin/loadwright validate "$spec" --env-file examples/api/.env.example
    bin/loadwright compile "$spec" --env-file examples/api/.env.example -o "/tmp/loadwright-examples/$(basename "$spec" .yaml).jmx"
  else
    bin/loadwright validate "$spec"
    bin/loadwright compile "$spec" -o "/tmp/loadwright-examples/$(basename "$spec" .yaml).jmx"
  fi
done
```

Import and compile OpenAPI examples:

```bash
mkdir -p /tmp/loadwright-openapi
find examples/openapi -name '*.yaml' -print | sort | while read -r spec; do
  name="$(basename "$spec" .yaml)"
  bin/loadwright import openapi "$spec" -o "/tmp/loadwright-openapi/$name.loadwright.yaml"
  bin/loadwright compile "/tmp/loadwright-openapi/$name.loadwright.yaml" -o "/tmp/loadwright-openapi/$name.jmx"
done
```

Run a real JMeter smoke test before cutting a release:

```bash
bin/loadwright run examples/api/query-params.yaml --out-dir results/manual-smoke --ci
bin/loadwright report results/manual-smoke/results.jtl --out-dir /tmp/loadwright-report --error-rate-lt 1 --p95-ms-lt 3000 --ci
```

## Current Coverage Focus

- Spec validation, defaults, variables, env files, auth, and timeout behavior.
- JMX rendering, including headers, query params, JSON bodies, urlencoded form bodies, assertions, duration loads, and timeouts.
- JTL parsing, percentile summaries, threshold pass/fail, and report artifacts.
- JUnit report generation for CI integrations.
- CLI parsing, non-Docker command flows, and end-to-end run/report generation with a Docker shim.
- OpenAPI/Postman/HAR import for YAML, JSON, path/query params, request bodies, and error cases.
- Runtime helper behavior for doctor/version parsing.
