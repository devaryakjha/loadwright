# Compatibility

Loadwright is a Go CLI that generates JMeter-compatible `.jmx` files and runs them through Docker.

## Go

Development and CI target Go 1.22 or newer.

## Operating Systems

Release builds target:

- macOS arm64 and amd64
- Linux arm64 and amd64
- Windows amd64

## JMeter

Generated plans are tested against the pinned HTTP runtime image:

```text
justb4/jmeter@sha256:088ac52b759a198a5afa5ae13d0a6306e9f2017d71ad140ff57427f6930406f7
```

Loadwright does not use floating `latest` tags as its support boundary. Patch releases may move to a newer immutable image digest after the CLI, generated JMX, and report flow are tested against that image.

HTTP specs use the default image unless `--image` is provided. WebSocket specs require a plugin-enabled image because the generated JMX uses the WebSocket Samplers plugin. Build the bundled image with:

```sh
docker build -t loadwright/jmeter-websocket:5.5 -f docker/jmeter/Dockerfile .
```

Use `loadwright doctor --deep --image <image:tag>` to verify that the configured image pulls and starts on your machine before running a load test.

## Docker

Docker is required for `loadwright run`, `loadwright doctor`, and `loadwright doctor --deep`. The basic doctor checks verify the Docker CLI, daemon reachability, writable directories, and image availability. The deep check also starts JMeter in the configured image.

Docker is not required for:

- `loadwright init`
- `loadwright validate`
- `loadwright compile`
- `loadwright import openapi`
- `loadwright import postman`
- `loadwright import har`
- `loadwright report`

## Spec Stability

Before `v1.0.0`, the YAML spec can still evolve. Changes should be documented in `CHANGELOG.md`, and breaking changes should include migration notes.

After `v1.0.0`, compatible additions should be preferred over breaking field changes.
