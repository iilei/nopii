# nopii

`nopii` is a small, pipe-first CLI for deterministic PII pseudonymization.
It is designed for workflows where command output should be sent to an LLM or
another external system without losing relationships between repeated
identities.

```text
git log --pretty=nopii-v1 | nopii | your-ai-cli
kubectl logs deployment/api | nopii | your-ai-cli
cat support-ticket.txt | nopii
```

The same input value produces the same pseudonym for the same key, scope and
entity type. `nopii` never needs network access for its built-in recognizers.

> [!IMPORTANT]
> `nopii` reduces exposure of detected PII. It does not prove that arbitrary
> input is anonymous, and pseudonymized data may still be personal data under
> applicable law or policy.

## Status

This repository is a v0.1 scaffold. The core design is implemented and tested:

- stdin -> stdout filter
- deterministic HMAC-SHA256 pseudonyms
- TOML config discovery from the working directory upward to the user's home
- environment/file based key loading
- built-in recognizers for email, IPv4, UUID and phone-like values
- structured Git log integration via a versioned `pretty.nopii-v1` contract
- `nopii init git`, `nopii doctor`, and `nopii config`
- GoReleaser configuration for Linux/macOS/Windows archives and Homebrew tap
- Chocolatey package template

A Presidio backend is intentionally not a runtime dependency in v0.1. It is a
planned optional deep-detection backend for free text so the base CLI can stay a
single, portable Go binary.

## Install from source

Requires Go 1.25 or newer.

```sh
go install github.com/iilei/nopii/cmd/nopii@latest
```

Or locally:

```sh
go build -o nopii ./cmd/nopii
```

## First run

Generate a strong secret and expose it through your secret manager. Do not put
secret values into `.nopiirc.toml`.

For a local shell test:

```sh
export NOPII_KEY="replace-with-a-long-random-secret"
printf 'alice@example.com called 10.20.30.40\n' | nopii
```

Example output:

```text
EMAIL_2HTPOVZV4CD2 called IP_5TNLBK2ZVSWX
```

Pseudonyms are derived from:

```text
HMAC-SHA256(key, "nopii:v1\\0" + scope + "\\0" + entity-type + "\\0" + normalized-value)
```

The digest is Base32 encoded and truncated to `output.token_length` characters.

## Git integration

`nopii` remains a stream filter. It does not wrap Git and does not silently
change `git log` behavior.

Initialize the explicit, versioned Git pretty format once:

```sh
nopii init git
```

This writes the following global Git config key:

```text
pretty.nopii-v1
```

Then use:

```sh
git log --pretty=nopii-v1 | nopii
```

Example output:

```sh
export NOPII_KEY=foo
git log --pretty=nopii-v1 | nopii
```

```text
commit 94a1f3e00fd9a761936082b1177c3a0668d042f3
parents 94caa08319f8f205765d1619cd66eb0bee57ebfe
Author: PERSON_H2OE6EACY2TJ <EMAIL_XK5RTMXZD65F>
Committer: PERSON_H2OE6EACY2TJ <EMAIL_XK5RTMXZD65F>
AuthorDate: 1786674827
CommitDate: 1786674827

chore: clean-code basic setup

commit 94caa08319f8f205765d1619cd66eb0bee57ebfe
Author: PERSON_H2OE6EACY2TJ <EMAIL_XK5RTMXZD65F>
Committer: PERSON_H2OE6EACY2TJ <EMAIL_XK5RTMXZD65F>
AuthorDate: 1786674194
CommitDate: 1786674194

feat: base project layout
```

The Git format contains a magic marker, so `nopii` automatically recognizes
this stream. Commit hashes, parent hashes and timestamps remain intact; author
and committer identities are deterministically pseudonymized; commit-message
free text is passed through the configured recognizers.

Optionally, author and commit unix timestamps can be floored to a configurable
granularity to reduce timing precision without losing ordering:

```toml
[git.date_clamp]
enabled = true
granularity_seconds = 86400  # daily buckets
```

The same input timestamp always produces the same clamped value for the same
granularity. Enabled via `NOPII_GIT_DATE_CLAMP_ENABLED=true` and
`NOPII_GIT_DATE_CLAMP_GRANULARITY=<seconds>` environment variables.

Repository-local initialization is available as well:

```sh
nopii init git --local
```

Initialization is idempotent. A conflicting existing value is not overwritten
unless requested explicitly:

```sh
nopii init git --force
```

Remove it again with:

```sh
nopii init git --remove
```

## Configuration

Configuration is optional. `nopii` looks for `.nopiirc.toml` starting in the
current working directory and walks parent directories up to and including the
user's home directory. It never intentionally traverses above the user's home
when started below it.

An explicit file wins:

```sh
nopii --config ./path/to/config.toml
```

Example:

```toml
version = 1
scope = "payments"

[key]
env = "NOPII_KEY"
# file = "/run/secrets/nopii-key"

[output]
token_length = 12

[recognizers]
email = true
ipv4 = true
uuid = true
phone = true

[git.date_clamp]
enabled = false
granularity_seconds = 86400  # floor timestamps to this boundary (e.g. 86400 = daily)

[classifiers]
username = "USER"
```

`scope` deliberately remains separate from the secret key. Use the same scope
where referential identity should be preserved across runs, machines or CI.

Custom recognizers can be added without changing the core algorithm. Each
`classifiers.<name>` entry maps a semantic class to the output label that should
appear in the pseudonym, and a matching `NOPII_CUSTOM_PATTERN__<NAME>`
environment variable provides the regular expression to match.

Example:

```sh
export NOPII_CUSTOM_PATTERN__USERNAME='(?m)(?:^|[[:space:]])@([A-Za-z0-9_-]+)'
```

```toml
[classifiers]
username = "USER"
```

This will replace matches like `@alice` with values like `USER_...`, while
keeping the semantic type in the output label.

CLI `--scope` overrides the config value.

### Secret management

The preferred interface is an environment variable or secret file. There is no
`--key value` flag, so secrets do not need to appear in command arguments.

```sh
NOPII_KEY=... command-producing-data | nopii
```

SOPS fits naturally without being a dependency of `nopii`:

```sh
sops exec-env ~/.config/nopii/secrets.sops.yaml \
  'git log --pretty=nopii-v1 | nopii | your-ai-cli'
```

The encrypted file can contain:

```yaml
NOPII_KEY: ENC[...]
```

Other secret sources such as CI secrets, Vault, Kubernetes secrets, Docker
secrets or OS credential managers can inject the same environment variable or
materialize a secret file.

## Commands

```text
nopii                         stdin -> pseudonymized stdout
nopii --scope NAME            override pseudonym scope
nopii --key-env NAME          read secret from a named env variable
nopii --key-file PATH         read secret from a file
nopii init git                install global Git pretty.nopii-v1
nopii init git --local        install in current repository
nopii init git --force        replace a conflicting integration
nopii init git --remove       remove the integration
nopii doctor                  show configuration/integration health
nopii config                  show effective non-secret configuration
nopii version                 print version
```

## Design principles

1. **Pipe first.** `nopii` reads stdin and writes stdout.
2. **No network requirement.** Built-in processing is local.
3. **Structure beats heuristics.** Integrations such as Git expose known fields;
   generic streams use recognizers.
4. **Deterministic, keyed pseudonyms.** Same entity + scope + key => same token.
5. **Secrets are external.** Config can describe where a key comes from, but it
   should not contain the secret itself.
6. **Explicit integrations.** Package installation has no Git side effects;
   `nopii init git` performs the opt-in setup.
7. **Version contracts.** Git uses `nopii-v1`; future incompatible formats can
   coexist instead of silently breaking existing setups.

## Presidio roadmap

The built-in recognizers deliberately cover high-confidence structured PII.
General person/location/organization detection in arbitrary text requires a
NER-capable backend. The planned architecture keeps this optional, for example:

```toml
[detection]
backend = "builtin"

# future option
# backend = "presidio"
# endpoint = "http://127.0.0.1:5001"
```

A [Presidio](https://github.com/data-privacy-stack/presidio/) sidecar can then run locally with network disabled while the Go CLI
remains portable. The security boundary must still assume that detectors can
miss PII.

## Homebrew

The included `.goreleaser.yml` can publish a formula to a separate tap, e.g.
`iilei/homebrew-tap` once that repository exists and a release token is
configured.

Target UX:

```sh
brew install iilei/tap/nopii
```

Moving to Homebrew Core can be considered after the project has sufficient
usage and meets its acceptance requirements.

## Chocolatey

`packaging/chocolatey/` contains a package template. Release automation should
replace checksum/version placeholders with values from the release artifacts
before publishing.

Target UX:

```powershell
choco install nopii
```

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/nopii
```

Try Git integration in an expendable repository or globally:

```sh
go run ./cmd/nopii init git
export NOPII_KEY='development-only-secret'
git log --pretty=nopii-v1 | go run ./cmd/nopii
```

## Security notes

- Use a high-entropy secret, ideally managed by SOPS/KMS/Vault/CI secret storage.
- Do not pass the secret as a CLI argument.
- A pseudonym is not encryption and should not be treated as anonymization.
- Low-entropy identifiers are resistant to simple dictionary attacks only when
  the HMAC key remains secret.
- Rotating the key changes pseudonyms. Changing the scope changes pseudonyms.
- The current built-in recognizers are intentionally conservative and do not
  claim comprehensive PII detection.

## License

MIT
