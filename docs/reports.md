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

## JUnit

`junit.xml` contains one testcase for sample failures and one testcase per threshold. This makes Loadwright reports consumable by CI systems that understand JUnit test reports.
