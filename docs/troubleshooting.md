# Troubleshooting

Use this page when the quickstart fails before you get a report.

## Docker Is Not Running

`loadwright doctor` checks that the Docker CLI exists. `loadwright doctor --deep` also starts the configured JMeter image.

If `doctor --deep` fails before JMeter starts:

```bash
docker version
docker info
loadwright doctor --deep
```

Start Docker Desktop or your Docker daemon, then rerun `loadwright doctor --deep`.

## Image Pull Or Startup Fails

Loadwright uses `justb4/jmeter:latest` by default for HTTP runs.

```bash
docker pull justb4/jmeter:latest
loadwright doctor --deep --image justb4/jmeter:latest
```

If your environment blocks public image pulls, mirror the image internally and pass it with `--image`:

```bash
loadwright run loadwright.yaml --ci --image registry.example.com/jmeter:5.6
```

## Permission Or Output Errors

`loadwright run` writes reports under `results/` unless `--out-dir` is provided.

```bash
mkdir -p results
loadwright run loadwright.yaml --out-dir results/basic-smoke --ci
```

For containerized compile/import/report commands, first verify that your environment can anonymously pull the Loadwright image. From a source checkout or release archive:

```bash
scripts/verify-ghcr-public-pull.sh ghcr.io/devaryakjha/loadwright:latest
```

If the pull succeeds, mount a writable working directory and run as your local user. From a source checkout:

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" ghcr.io/devaryakjha/loadwright:latest compile examples/api/basic.yaml
```

If Docker reports `unauthorized`, use the release binary or `go install` path instead. If you intentionally need a private GHCR package, authenticate with a token that has `read:packages` before pulling:

```bash
echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin
docker pull ghcr.io/devaryakjha/loadwright:latest
```

For maintainers, an anonymous `unauthorized` response means the container image is not yet publicly pullable. First confirm a release workflow has published after the repository became public. If a fresh public-repo publish still fails, open the `loadwright` package under the GitHub profile's Packages tab and adjust the package visibility/access settings.

## WebSocket Examples

WebSocket specs currently require the bundled plugin-enabled JMeter image.

```bash
docker build -t loadwright/jmeter-websocket:latest -f docker/jmeter/Dockerfile .
loadwright doctor --deep --image loadwright/jmeter-websocket:latest
loadwright run examples/api/websocket-echo.yaml --ci --image loadwright/jmeter-websocket:latest
```

HTTP examples do not require this WebSocket image.

## Validate Before Running

If a spec fails during a run, validate and compile it first. These commands do not start Docker:

```bash
loadwright validate loadwright.yaml
loadwright compile loadwright.yaml
```
