# CI

Loadwright is designed to fail CI when performance thresholds are breached.

```bash
go build -o bin/loadwright ./cmd/loadwright
bin/loadwright validate examples/api/basic.yaml
bin/loadwright run examples/api/basic.yaml --ci
```

For environment-specific tests:

```bash
bin/loadwright run loadwright.yaml --env-file .env.ci --ci
```

Use `validate` in fast pull request jobs when you want spec checks without starting JMeter:

```bash
bin/loadwright validate loadwright.yaml --env-file .env.ci
```

Use `report` when another JMeter job already produced a JTL file:

```bash
bin/loadwright report results.jtl --out-dir results/report --error-rate-lt 1 --p95-ms-lt 3000 --ci
```

Upload `results/**/junit.xml` with your CI platform's JUnit test report integration when available.

The command exits with:

- `0` when JMeter runs successfully and thresholds pass
- `1` when JMeter fails, report parsing fails, or thresholds fail
- `2` for invalid CLI usage

## Example GitHub Actions Step

```yaml
- uses: actions/setup-go@v6
  with:
    go-version: "1.22"
- run: go build -o bin/loadwright ./cmd/loadwright
- run: bin/loadwright run examples/api/basic.yaml --ci
```

For fast pull request checks, compile specs without running load:

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

## Copy-Paste GitHub Actions Workflow

See [examples/github-actions/loadwright-pr.yml](../examples/github-actions/loadwright-pr.yml) for a complete workflow users can copy into their own repository.

The workflow has two jobs:

- `validate`: runs on pull requests and pushes. It validates YAML specs and compiles them to JMX without starting JMeter.
- `smoke`: runs only on pushes to `main`. It runs one threshold-gated smoke test and uploads the generated report artifacts.

This split keeps pull request checks fast while still giving the default branch a real performance gate.

Expected repository layout for the example:

```text
load-tests/
  smoke.yaml
.env.ci
```

If your specs do not need environment values, remove `--env-file .env.ci` from the workflow.

For private APIs, keep secrets in GitHub Actions secrets and write `.env.ci` during the workflow:

```yaml
- name: Write Loadwright env
  run: |
    {
      echo "API_BASE_URL=${API_BASE_URL}"
      echo "API_TOKEN=${API_TOKEN}"
    } > .env.ci
  env:
    API_BASE_URL: ${{ secrets.API_BASE_URL }}
    API_TOKEN: ${{ secrets.API_TOKEN }}
```

Keep pull request runs small. Large load tests should run on `push`, `schedule`, or manual `workflow_dispatch` triggers where they will not slow every review cycle.
