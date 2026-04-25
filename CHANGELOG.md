# Changelog

## Unreleased

- Remove the legacy Python prototype from the public repository.
- Add public project-status badges and initial prototype attribution.
- Add mailmap attribution for Jaykumar Gori's earlier commit identities.
- Add initial Postman Collection v2.1 import support.
- Add initial HAR 1.2 import support.
- Improve Markdown and HTML reports with endpoint triage tables.
- Add a copy-paste GitHub Actions workflow example for downstream CI adoption.
- Add latest-run metadata for default `loadwright run` result directories.
- Add `loadwright compare` for Markdown comparisons between two `summary.json` files.

## 0.1.0 - 2026-04-25

- Start Go-first OSS implementation.
- Add roadmap.
- Add YAML API spec model.
- Add JMX generation.
- Add Dockerized JMeter run command.
- Add JSON, Markdown, and HTML report generation.
- Add initial tests, docs, and CI workflow.
- Rename project to Loadwright.
- Move the original Python prototype under `legacy/python-prototype`.
- Expand `doctor` with directory, image, and optional deep JMeter runtime checks.
- Expand tests across spec validation, JMX rendering, report parsing, and CLI flows.
- Add query-param, POST JSON, duration-load, and threshold-failure examples.
- Add getting started, examples, CI, and reports documentation.
- Add variables, env files, bearer/basic auth helpers, and request timeouts.
- Add initial OpenAPI 3.x import support.
- Expand test coverage across CLI, spec, report, runtime helper, and OpenAPI edge cases.
- Add CSV data source support backed by JMeter `CSVDataSet`.
- Add version metadata, GoReleaser config, release workflows, Dockerfile, and install docs.
- Add release checklist, realistic checkout example, and pre-release hardening tests.
- Add `loadwright validate` for fast no-Docker spec checks.
- Add GitHub issue/PR templates and compatibility policy.
- Add `loadwright report` for regenerating reports from existing JTL files.
- Add CI-friendly `junit.xml` report output.
- Document the `v0.1.0` HTTP API testing scope and known limitations.
