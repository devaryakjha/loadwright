# Spec Reference

loadwright specs are YAML files that compile to portable JMeter `.jmx` files.

```yaml
name: httpbin-basic
target: https://httpbin.org
load:
  users: 5
  ramp_up: 10s
  loops: 3
requests:
  - name: get status
    method: GET
    path: /status/200
    expect:
      status: 200
thresholds:
  error_rate_lt: 1
  p95_ms_lt: 3000
```

## Fields

- `name`: required test name.
- `target`: required base `http`, `https`, `ws`, or `wss` URL.
- `variables`: optional map of template variables.
- `defaults.timeout`: optional default request timeout.
- `auth`: optional global auth helper.
- `data`: optional CSV data sources.
- `load.users`: concurrent JMeter threads. Defaults to `1`.
- `load.ramp_up`: ramp-up duration. Supports seconds as an integer or strings like `30s`, `5m`, `1h`.
- `load.loops`: number of loops per user.
- `load.duration`: duration-based run. When set without `loops`, the generated JMX runs loops until the duration expires.
- `requests`: required list of HTTP or WebSocket requests.
- `thresholds`: optional CI pass/fail rules.

## Request Bodies

Use one body field per request.

Legacy `body` remains supported and is rendered as raw body content. Prefer the explicit fields for new specs:

JSON body:

```yaml
requests:
  - method: POST
    path: /users
    headers:
      content-type: application/json
    body_json:
      name: Ada
```

Text body:

```yaml
requests:
  - method: POST
    path: /events
    body_text: plain text payload
```

Urlencoded form body:

```yaml
requests:
  - method: POST
    path: /login
    body_form:
      username: demo
      password: secret
```

`body_form` compiles to JMeter HTTP arguments with `HTTPSampler.postBodyRaw=false` and adds `Content-Type: application/x-www-form-urlencoded` when no content type header is already set.

Multipart form body:

```yaml
requests:
  - method: POST
    path: /upload
    body_multipart:
      - name: title
        value: avatar
      - name: avatar
        file: ./avatar.png
        content_type: image/png
```

`body_multipart` compiles to JMeter multipart form-data arguments and file arguments. Text parts use `value`; file parts use `file` plus optional `content_type`. Loadwright lets JMeter set the multipart `Content-Type` boundary.

## Variables

Variables can be referenced with `{{name}}`.

```yaml
variables:
  token: ${API_TOKEN}
requests:
  - path: /users/{{user_id}}
```

Environment values use `${NAME}` and can come from the process environment or an env file:

```bash
loadwright compile spec.yaml --env-file .env.test
loadwright run spec.yaml --env-file .env.test --ci
```

Env files use simple `KEY=value` lines:

```dotenv
API_TOKEN=secret
API_HOST=api.example.com
```

## Auth

Bearer auth:

```yaml
auth:
  type: bearer
  token: "{{token}}"
```

Basic auth:

```yaml
auth:
  type: basic
  username: "{{username}}"
  password: "{{password}}"
```

Auth can be set globally or per request. If a request already defines an `authorization` header, Loadwright does not overwrite it.

## Timeouts

Set a default timeout for all requests:

```yaml
defaults:
  timeout: 5s
```

Override it per request:

```yaml
requests:
  - path: /slow
    timeout: 2s
```

Loadwright renders the timeout into JMeter connect and response timeout fields in milliseconds.

For WebSocket requests, `timeout` can also be set inside `websocket.timeout`.

## WebSocket Requests

Set `protocol: websocket` to use a WebSocket request in `requests`.

### Basic single-message example

```yaml
target: wss://echo.websocket.events
requests:
  - name: ws ping
    protocol: websocket
    path: /
    websocket:
      message: ping
      expect_contains: ping
      timeout: 5s
```

### Multi-message sequence

Use `websocket.messages[]` for richer multi-message sequences:

```yaml
requests:
  - name: chat flow
    protocol: websocket
    path: /chat
    websocket:
      timeout: 10s
      messages:
        - send: hello
          expect:
            contains: hello
        - send: how are you
          delay: 1s
          expect:
            contains: how are you
            timeout: 5s
```

### WebSocket fields

| Field | Description |
|-------|-------------|
| `websocket.url` | Optional. When omitted, Loadwright builds the URL from `target + path`. |
| `websocket.timeout` | Connection timeout. Inherited by `messages[].expect.timeout` when not set per message. |
| `websocket.subprotocol` | Optional WebSocket subprotocol (e.g. `graphql-ws`). |
| `websocket.headers` | Optional map of custom headers sent during the WebSocket handshake. |
| `websocket.close_timeout` | Optional timeout for the graceful close handshake. Defaults to `timeout`. |
| `websocket.messages[]` | Ordered list of messages to send and optionally assert. |
| `websocket.messages[].send` | Required. The text payload to send. |
| `websocket.messages[].type` | `text` (default) or `binary`. For `binary`, `send` must be valid base64. |
| `websocket.messages[].delay` | Optional delay before sending this message. |
| `websocket.messages[].expect.contains` | Optional substring assertion on the received response. |
| `websocket.messages[].expect.timeout` | Per-message read timeout. Defaults to the connection `timeout`. |

### Legacy fields

The `websocket.message` and `websocket.expect_contains` fields are still supported for backward compatibility. They are equivalent to a single-element `messages[]` list. You cannot mix legacy fields with `messages[]` in the same request.

### Restrictions

WebSocket requests do not support HTTP-only fields (`method`, `headers`, `query`, `body`, `expect.status`, `auth`). Use `websocket.headers` for custom WebSocket handshake headers.

## Data Sources

CSV data sources are declared under `data`.

```yaml
data:
  users:
    file: users.csv
    recycle: true
    stop_thread: false
    sharing: all
```

When `variables` is omitted, Loadwright reads the CSV header row. Use CSV columns in requests with JMeter runtime variables such as `${username}`.

## Thresholds

- `error_rate_lt`: fail if error rate is greater than or equal to this percentage.
- `p95_ms_lt`: fail if p95 latency is greater than or equal to this value.
- `avg_ms_lt`: fail if average latency is greater than or equal to this value.
