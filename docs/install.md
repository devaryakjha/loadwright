# Install

Loadwright publishes release binaries, checksums, and a Go module.

The current public release is `v0.2.0`.

## GitHub Releases

Download the archive for your operating system from [GitHub Releases](https://github.com/devaryakjha/loadwright/releases/tag/v0.2.0).

Release artifacts are produced for:

- macOS arm64 and amd64
- Linux arm64 and amd64
- Windows amd64

Each release includes `checksums.txt`.

Example for macOS arm64:

```bash
curl -L -o loadwright.tar.gz https://github.com/devaryakjha/loadwright/releases/download/v0.2.0/loadwright_0.2.0_darwin_arm64.tar.gz
tar -xzf loadwright.tar.gz
./loadwright version
```

## Go Install

```bash
go install github.com/devaryakjha/loadwright/cmd/loadwright@v0.2.0
```

`go install` builds from source, so `loadwright version` may show development metadata. Use the GitHub release archives when you need the exact release version, commit, and build date embedded in the binary.

For the latest commit on the default branch:

```bash
go install github.com/devaryakjha/loadwright/cmd/loadwright@latest
```

## Build From Source

```bash
go build -o bin/loadwright ./cmd/loadwright
bin/loadwright version
```

## Container Image

The release workflow builds `ghcr.io/devaryakjha/loadwright`, but the recommended first-run path is the release binary or `go install`.

Use the container image only after verifying that your environment can pull it:

```bash
docker pull ghcr.io/devaryakjha/loadwright:latest
```

If the pull succeeds, the image is useful for compile/import/report workflows that do not need to start Dockerized JMeter from inside the container. From a source checkout:

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" ghcr.io/devaryakjha/loadwright:latest compile examples/api/basic.yaml
```

Running load tests from inside the container requires a Docker strategy that will be documented separately.

## First Check

After installing, create and check a starter spec:

```bash
loadwright version
loadwright doctor
loadwright init
loadwright validate loadwright.yaml
loadwright compile loadwright.yaml
```

Use `loadwright doctor --deep` before your first `loadwright run` to verify Docker can start JMeter.
