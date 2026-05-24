# Limitations

Loadwright is intentionally focused on HTTP and WebSocket API load-test workflows.

## In Scope

- YAML specs for HTTP API requests.
- GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS requests.
- Query params, headers, JSON/string bodies, expected HTTP status assertions.
- Urlencoded and multipart form bodies.
- Basic and bearer auth helpers.
- Environment files and simple variable substitution.
- CSV data sources.
- WebSocket testing with multi-message sequences, subprotocols, custom headers, binary frames, and per-message assertions.
- Dockerized JMeter execution.
- Reports from Loadwright runs or existing JMeter JTL files.
- Initial OpenAPI 3.x import for simple HTTP APIs.
- Initial Postman Collection v2.1 import for common HTTP API collections.
- Initial HAR 1.2 import for common HTTP API traffic captures.

## Not Yet In Scope

- Full JMeter GUI feature parity.
- JMeter plugin management.
- Distributed load generation across multiple workers.
- Historical trend storage.
- Browser-level performance testing.
- AI-assisted spec generation or result explanation.

## Compatibility Boundary

Generated JMX files are meant to be portable JMeter plans, but Loadwright only generates the subset it understands today. Hand-editing generated JMX is fine, but changes made directly to JMX are not converted back into YAML.

For advanced JMeter features not represented in the YAML spec yet, keep using JMeter directly and use `loadwright report` to generate Loadwright summaries from the resulting JTL file.
