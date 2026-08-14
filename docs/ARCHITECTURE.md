# Architecture

## Data flow

```text
stdin
  |
  +-- magic == NOPII_GIT_V1 --> Git v1 parser --> structured field handling
  |
  +-- otherwise -------------> generic recognizers
                                      |
                                      v
                           HMAC pseudonym generator
                                      |
                                      v
                                    stdout
```

## Trust boundaries

`nopii` is intentionally local and does not need network access for built-in
processing. Secret acquisition is delegated to the environment around it:
SOPS, KMS, Vault, CI, Docker/Kubernetes secrets or OS credential managers.

Configuration describes behavior and secret *location*, never a literal secret
value.

## Git v1 wire format

`nopii init git` installs `pretty.nopii-v1` using ASCII Unit Separator (`0x1f`) between metadata fields and NUL (`0x00`) between records. A magic value identifies the stream. The final body field may contain field-separator bytes because parsing is limited to ten fields.

Field order:

1. `NOPII_GIT_V1`
2. commit hash
3. parent hashes
4. author name
5. author email
6. committer name
7. committer email
8. author unix timestamp
9. commit unix timestamp
10. raw commit body

The output renderer intentionally returns normal readable text rather than the
wire format. Identity fields are pseudonymized before output. The body is passed
through generic recognizers.

## Future Presidio backend

A future detector interface can support `builtin` and `presidio` without
changing the pseudonymization layer. Presidio should be optional so the default
release remains a single Go binary.
