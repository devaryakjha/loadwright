# HAR Import

Loadwright can generate a starter spec from a HAR 1.2 JSON file exported from a browser or API debugging tool.

```bash
bin/loadwright import har examples/har/checkout.har -o /tmp/checkout-har-loadwright.yaml
bin/loadwright validate /tmp/checkout-har-loadwright.yaml
bin/loadwright compile /tmp/checkout-har-loadwright.yaml -o /tmp/checkout-har.jmx
```

Override the target server:

```bash
bin/loadwright import har capture.har --base-url https://staging.example.com -o loadwright.yaml
```

## What Gets Imported

- The HAR filename becomes the spec name.
- The first request origin becomes `target` unless `--base-url` is provided.
- Supported requests become Loadwright HTTP requests.
- Method, path, query params, headers, and JSON/text request bodies are imported.
- Form params are imported as runnable `body_form` specs.
- Unsupported or lossy capture details are reported as warnings.

## Current Limitations

- HAR 1.2 JSON only.
- Browser replay semantics are not reproduced.
- Cookies are not imported.
- File uploads and encoded/binary request bodies are skipped with warnings.
- Multipart form-data is not rendered as multipart JMeter requests yet.
- Multiple target hosts are imported into a single Loadwright target; review warnings before using the generated spec in CI.
- Imported specs are starter specs and should be reviewed before CI use.
