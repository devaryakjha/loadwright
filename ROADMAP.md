# Loadwright Roadmap

loadwright aims to become a true open source, Docker-first, spec-driven JMeter automation tool. The core product should work without AI, without a hosted service, and without requiring users to understand raw JMX XML.

The OSS implementation is Go-first. The original Python prototype was removed before the public release to keep the repository focused; attribution is preserved in the README.

## Product Promise

Make JMeter easy to use from the terminal and CI:

- write a readable YAML test spec
- compile it to a JMeter `.jmx` plan
- run it in Docker
- generate useful reports
- fail CI when performance thresholds are breached

AI should remain optional. It should help users create, explain, and improve specs, but the deterministic workflow must be good enough on its own.

## Principles

- Spec-driven first: YAML specs are the primary user interface.
- Docker-first: users should not need to install Java or JMeter locally.
- CI-friendly: every run should produce clear exit codes and machine-readable output.
- Offline by default: AI and cloud integrations are optional.
- JMeter-compatible: generated `.jmx` files should remain portable.
- Honest positioning: this is not a new load-testing engine; it makes JMeter easier to automate.

## Stack Choice

The preferred implementation stack is Go.

Why Go:

- single static-ish binary distribution
- fast startup for CLI workflows
- strong standard library for XML, CSV, subprocesses, and filesystem work
- straightforward Docker/JMeter process orchestration
- easier cross-platform releases than Python
- simpler contributor onboarding than Rust for this type of tool

Rust remains a good future option for deeply performance-sensitive internals, but the bottleneck here is JMeter execution rather than CLI code. Zig is not the right first choice because the YAML/CLI/reporting ecosystem is less mature for this product.

## Phase 1: OSS Foundation

Goal: make the project installable, testable, and understandable.

- Add Go module structure under `cmd/` and `internal/`.
- Add a `Loadwright` binary entry point.
- Add a deterministic CLI.
- Add a YAML spec model with validation.
- Add an API test JMX generator.
- Add a Docker-backed JMeter runner.
- Add JTL parsing and threshold evaluation.
- Add JSON, Markdown, and HTML reports.
- Add unit tests and golden-file tests.
- Add examples users can run immediately.
- Add README, contributing docs, license, security policy, and CI.

## Phase 2: Useful Daily Workflow

Goal: make the tool valuable for backend developers, QA engineers, and CI pipelines.

- `loadwright doctor` to verify Docker, image access, and writable paths.
- `loadwright doctor --deep` to verify JMeter startup through Docker.
- `loadwright init` to create a starter spec.
- `loadwright compile spec.yaml --out tests/spec.jmx`.
- `loadwright run spec.yaml --ci`.
- Threshold-based pass/fail behavior.
- Stable output directory layout with `latest` pointers or clear run IDs.
- Better error messages for invalid specs and failed JMeter runs.
- GitHub Actions example for performance checks in pull requests.

## Phase 3: Broader Input Support

Goal: let teams reuse assets they already have.

- Import OpenAPI/Swagger specs into Loadwright YAML.
- Expand Postman collection import beyond the initial HTTP starter-spec support.
- Import browser HAR files.
- Run and report on existing `.jmx` files without conversion.
- Add request variables, CSV data sources, auth helpers, and environment files.

## Phase 4: WebSocket And Plugins

Goal: make JMeter plugin usage less painful.

- First-class WebSocket spec support.
- Reliable WebSocket plugin installation and verification.
- Plugin lockfile or manifest for reproducible plugin versions.
- Plugin health checks in `loadwright doctor`.
- Clear plugin docs and troubleshooting output.

## Phase 5: AI-Assisted Workflow

Goal: add AI where it reduces setup time without making runs non-deterministic.

AI should generate or modify specs, not secretly mutate runtime behavior.

Potential commands:

```bash
loadwright ai generate "test login and checkout with 50 users for 5 minutes"
loadwright ai explain results/run-123/summary.json
loadwright ai improve spec.yaml --from results/run-123/summary.json
```

AI features:

- Generate YAML specs from plain English.
- Convert OpenAPI/Postman/HAR assets into draft specs.
- Explain report results in plain language.
- Suggest thresholds from historical runs.
- Suggest test coverage gaps.
- Repair invalid specs with a reviewable diff.

Non-goals for AI:

- AI-only test execution.
- Hidden changes to `.jmx` or YAML files.
- Requiring API keys for normal CLI usage.

## Phase 6: Distribution

Goal: make installation boring.

- Publish to PyPI.
- Publish a Docker image to GitHub Container Registry.
- Add `pipx` install docs.
- Add a Homebrew tap if there is enough demand.
- Add signed releases and changelog automation.

## Phase 7: Community And Governance

Goal: make the project welcoming and maintainable.

- Add issue templates.
- Add PR template.
- Add `CONTRIBUTING.md`.
- Add `CODE_OF_CONDUCT.md`.
- Add `SECURITY.md`.
- Add `CHANGELOG.md`.
- Label good first issues.
- Document compatibility policy for JMeter versions.

## Adoption Milestones

- A new user can run the quickstart in under 5 minutes.
- A CI user can add performance thresholds in under 15 minutes.
- A JMeter user can run an existing `.jmx` in Docker without local setup.
- A QA engineer can create an API test without opening the JMeter GUI.
- WebSocket plugin setup works reliably on a clean machine.

## Near-Term MVP

The first public-quality release should include:

- `loadwright doctor`
- `loadwright init`
- `loadwright compile`
- `loadwright run`
- YAML API test specs
- Dockerized JMeter execution
- JSON, Markdown, and HTML reports
- threshold-based CI exits
- tests and GitHub Actions
- examples and clear documentation
