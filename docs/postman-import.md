# Postman Import

Loadwright can generate a starter spec from a Postman Collection v2.1 JSON file.

```bash
bin/loadwright import postman examples/postman/checkout-api.postman_collection.json -o /tmp/checkout-loadwright.yaml
bin/loadwright validate /tmp/checkout-loadwright.yaml
bin/loadwright compile /tmp/checkout-loadwright.yaml -o /tmp/checkout.jmx
```

Override the target server:

```bash
bin/loadwright import postman collection.json --base-url https://staging.example.com -o loadwright.yaml
```

## What Gets Imported

- Collection name becomes the spec name.
- Collection variables become Loadwright variables.
- Folders are included in request names.
- Supported requests become Loadwright HTTP requests.
- Method, path, query params, headers, raw JSON/text bodies, urlencoded fields, and text form-data fields are imported.
- Collection-level bearer/basic auth becomes global Loadwright auth.
- Request-level bearer/basic auth becomes request auth.
- Postman variables such as `{{base_url}}` remain reviewable in the YAML output.
- Urlencoded fields become runnable `body_form` specs.
- Text form-data fields are imported as flat starter bodies with warnings.

## Current Limitations

- Postman Collection v2.1 JSON only.
- Postman pre-request scripts and tests are not executed or translated.
- File uploads, GraphQL, and advanced auth modes are reported as warnings and skipped where needed.
- Multipart form-data is not rendered as multipart JMeter requests yet, so review generated form-data starter specs before CI use.
- Multiple target hosts are imported into a single Loadwright target; review warnings before using the generated spec in CI.
- Imported specs are starter specs and should be reviewed before CI use.
