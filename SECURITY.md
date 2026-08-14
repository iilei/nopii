# Security policy

`nopii` is a privacy-reduction tool, not an anonymization guarantee.

Please report security issues privately to the project maintainer rather than
opening a public issue containing sensitive samples.

## Secret handling

- Secrets are accepted from environment variables or files.
- Literal secret values are not supported in the TOML schema.
- `nopii` does not provide a `--key <secret>` option.
- Diagnostic output must never print secret values.

## Detector limitations

No recognizer can guarantee detection of all PII in arbitrary text. Treat the
output according to your organization's data classification and compliance
requirements.
