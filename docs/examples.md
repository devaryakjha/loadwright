# Examples

The `examples/` directory contains runnable specs for common scenarios.

## Basic API

```bash
bin/loadwright run examples/api/basic.yaml --ci
```

Demonstrates a GET request, a POST JSON request, status assertions, and passing thresholds.

## Query Params

```bash
bin/loadwright run examples/api/query-params.yaml --ci
```

Demonstrates query-string parameters on a GET request.

## POST JSON

```bash
bin/loadwright run examples/api/post-json.yaml --ci
```

Demonstrates JSON request bodies and custom headers.

## Urlencoded Form

```bash
bin/loadwright run examples/api/form-urlencoded.yaml --ci
```

Demonstrates `application/x-www-form-urlencoded` request bodies with `body_form`.

## Multipart Upload

```bash
bin/loadwright run examples/api/multipart-upload.yaml --ci
```

Demonstrates multipart form-data uploads with `body_multipart`.

## Checkout Flow

```bash
bin/loadwright run examples/api/checkout-flow.yaml --ci
```

Demonstrates a small multi-step user journey with query params, JSON bodies, status assertions, and thresholds.

## Duration Load

```bash
bin/loadwright run examples/api/duration-load.yaml --ci
```

Demonstrates duration-based load instead of a fixed loop count.

## Threshold Failure

```bash
bin/loadwright run examples/api/threshold-fail.yaml --ci
```

Demonstrates CI failure behavior. This example intentionally uses an unrealistic p95 threshold.

## Bearer Auth

```bash
API_TOKEN=demo-token bin/loadwright run examples/api/bearer-auth.yaml --ci
```

Demonstrates global bearer auth.

## Basic Auth

```bash
BASIC_USERNAME=user BASIC_PASSWORD=pass bin/loadwright run examples/api/basic-auth.yaml --ci
```

Demonstrates global basic auth.

## Env File

```bash
bin/loadwright run examples/api/env-file.yaml --env-file examples/api/.env.example --ci
```

Demonstrates `${ENV}` values and `{{variable}}` substitution.

## Timeouts

```bash
bin/loadwright run examples/api/timeouts.yaml --ci
```

Demonstrates default and request-specific timeouts.

## WebSocket Echo

WebSocket specs require a plugin-enabled JMeter image. Build it once from the bundled Dockerfile (extends `justb4/jmeter:latest` with the [WebSocket Samplers](https://github.com/ptrd/jmeter-websocket-samplers) plugin), then reuse it for all WebSocket examples:

```bash
docker build -t loadwright/jmeter-websocket:latest -f docker/jmeter/Dockerfile .
bin/loadwright run examples/api/websocket-echo.yaml --ci --image loadwright/jmeter-websocket:latest
```

Demonstrates a WebSocket request that sends one message and checks the first response.

## WebSocket Multi-Message

```bash
docker build -t loadwright/jmeter-websocket:latest -f docker/jmeter/Dockerfile .  # skip if already built
bin/loadwright run examples/api/websocket-multi.yaml --ci --image loadwright/jmeter-websocket:latest
```

Demonstrates a multi-message WebSocket sequence with delays and per-message assertions.

## WebSocket Subprotocol

```bash
docker build -t loadwright/jmeter-websocket:latest -f docker/jmeter/Dockerfile .  # skip if already built
bin/loadwright run examples/api/websocket-subprotocol.yaml --ci --image loadwright/jmeter-websocket:latest
```

Demonstrates WebSocket subprotocol negotiation and custom handshake headers.

## OpenAPI Import

```bash
bin/loadwright import openapi examples/openapi/petstore-lite.yaml -o /tmp/petstore-loadwright.yaml
bin/loadwright compile /tmp/petstore-loadwright.yaml -o /tmp/petstore.jmx
```

Demonstrates generating a starter Loadwright spec from OpenAPI 3.x.

## Postman Import

```bash
bin/loadwright import postman examples/postman/checkout-api.postman_collection.json -o /tmp/checkout-loadwright.yaml
bin/loadwright compile /tmp/checkout-loadwright.yaml -o /tmp/checkout.jmx
```

Demonstrates generating a starter Loadwright spec from a Postman Collection v2.1 file.

## HAR Import

```bash
bin/loadwright import har examples/har/checkout.har -o /tmp/checkout-har-loadwright.yaml
bin/loadwright compile /tmp/checkout-har-loadwright.yaml -o /tmp/checkout-har.jmx
```

Demonstrates generating a starter Loadwright spec from a HAR 1.2 traffic capture.

## GitHub Actions

```bash
cp examples/github-actions/loadwright-pr.yml .github/workflows/loadwright.yml
```

Demonstrates a downstream CI workflow with fast pull request validation and a threshold-gated smoke run on `main`.

## CSV Users

```bash
bin/loadwright run examples/api/csv-users.yaml --ci
```

Demonstrates JMeter CSV data sources and runtime variables like `${username}`.
