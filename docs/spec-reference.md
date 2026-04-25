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
- `target`: required base `http` or `https` URL.
- `variables`: optional map of template variables.
- `defaults.timeout`: optional default request timeout.
- `auth`: optional global auth helper.
- `data`: optional CSV data sources.
- `load.users`: concurrent JMeter threads. Defaults to `1`.
- `load.ramp_up`: ramp-up duration. Supports seconds as an integer or strings like `30s`, `5m`, `1h`.
- `load.loops`: number of loops per user.
- `load.duration`: duration-based run. When set without `loops`, the generated JMX runs loops until the duration expires.
- `requests`: required list of HTTP requests.
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
