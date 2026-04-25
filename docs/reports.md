# Reports

Each `loadwright run` writes a JMeter result file plus Loadwright summaries.

You can also regenerate reports from an existing JMeter JTL file without running JMeter:

```bash
bin/loadwright report results/manual-smoke/results.jtl --out-dir results/manual-smoke
```

## Files

- `results.jtl`: raw JMeter CSV output
- `summary.json`: machine-readable summary
- `summary.md`: concise Markdown summary
- `index.html`: standalone human-readable report
- `junit.xml`: CI-friendly JUnit report for sample and threshold failures
- `run.json`: metadata about the run input, JMeter image, timings, and artifact paths

`run.json` does not record environment variable values or env-file contents.

```json
{
  "run_id": "20260425-120000",
  "input": "loadwright.yaml",
  "input_type": "yaml",
  "jmx": "results/20260425-120000/example-api.jmx",
  "generated_jmx": true,
  "image": "justb4/jmeter:latest",
  "ci": true,
  "started_at": "2026-04-25T12:00:00Z",
  "finished_at": "2026-04-25T12:00:10Z",
  "artifacts": {
    "results_jtl": "results/20260425-120000/results.jtl",
    "summary_json": "results/20260425-120000/summary.json",
    "summary_md": "results/20260425-120000/summary.md",
    "report_html": "results/20260425-120000/index.html",
    "junit_xml": "results/20260425-120000/junit.xml"
  }
}
```

When `loadwright run` uses the default output directory, Loadwright also writes `results/latest.json`:

```json
{
  "run_id": "20260425-120000",
  "run_dir": "results/20260425-120000",
  "report": "results/20260425-120000/index.html",
  "updated_at": "2026-04-25T12:00:00Z"
}
```

On platforms that support symlinks, `results/latest` points at the newest default run directory. Explicit `--out-dir` runs do not update `results/latest.json`.

## Metrics

Loadwright currently reports:

- total samples
- successful and failed samples
- error rate
- min, max, average
- p50, p90, p95, p99
- per-endpoint count, failures, average, and p95
- threshold pass/fail results

The HTML and Markdown reports include endpoint tables sorted for triage: failing endpoints first, then highest p95 latency, then highest average latency.

## Thresholds

Thresholds live in the YAML spec:

```yaml
thresholds:
  error_rate_lt: 1
  p95_ms_lt: 3000
  avg_ms_lt: 1000
```

When `--ci` is set, any failed threshold exits with code `1`.

For existing JTL files, pass thresholds as CLI flags:

```bash
bin/loadwright report results.jtl \
  --out-dir results/report \
  --error-rate-lt 1 \
  --p95-ms-lt 3000 \
  --avg-ms-lt 1000 \
  --ci
```

The report command writes artifacts before checking the `--ci` exit status, so failed threshold runs still leave inspectable summaries behind.

## Compare Runs

Compare two Loadwright `summary.json` files:

```bash
bin/loadwright compare results/baseline/summary.json results/candidate/summary.json
```

Write the comparison to a Markdown file:

```bash
bin/loadwright compare \
  results/baseline/summary.json \
  results/candidate/summary.json \
  -o results/comparison.md
```

The comparison includes top-level metric deltas plus per-endpoint deltas for failure count, error rate, average latency, and p95 latency. Added and removed endpoints are called out in the endpoint table.

## JUnit

`junit.xml` contains one testcase for sample failures and one testcase per threshold. This makes Loadwright reports consumable by CI systems that understand JUnit test reports.
