# Release Checklist

Loadwright releases are created by pushing a semantic version tag. Do not cut a tag until the checklist below is complete.

## Before Tagging

- Confirm `main` is green in GitHub Actions.
- Run the local quality gate:

```bash
go test ./...
go vet ./...
actionlint
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

- Build and smoke-test the CLI:

```bash
go build -o bin/loadwright ./cmd/loadwright
bin/loadwright version
bin/loadwright validate examples/api/basic.yaml
bin/loadwright compile examples/api/basic.yaml -o /tmp/loadwright-basic.jmx
```

- Run one real JMeter smoke test against a known endpoint and inspect `summary.json`:

```bash
bin/loadwright run examples/api/query-params.yaml --out-dir results/release-smoke --ci
bin/loadwright report results/release-smoke/results.jtl --out-dir results/release-smoke --error-rate-lt 1 --p95-ms-lt 3000 --ci
```

- Confirm docs and examples match the released behavior.
- Update `CHANGELOG.md` with the release date and notable changes.

## Tagging

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

Pushing the tag triggers `.github/workflows/release.yml`, which publishes GitHub release artifacts and GHCR images.

## After Tagging

- Confirm the GitHub Release has archives for macOS, Linux, and Windows.
- Confirm `checksums.txt` exists.
- Confirm the container image exists in GHCR and is publicly pullable before documenting it as an install path.
- Run a downloaded binary with `loadwright version`.
- Confirm `summary.json`, `summary.md`, `index.html`, and `junit.xml` are produced by the release smoke test.
- Create a follow-up issue for anything deferred from the release.
