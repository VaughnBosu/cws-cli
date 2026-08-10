# cws — Chrome Web Store CLI

A single-binary CLI for managing Chrome Web Store extensions. Upload, publish, rollout, and more — from your terminal, powered by the latest V2 API.

**[Documentation](https://vaughnbosu.github.io/cws-cli/)**

## Install

```bash
brew install --cask vaughnbosu/tap/cws
```

Or via script:

```bash
curl -fsSL https://vaughnbosu.github.io/cws-cli/install.sh | bash
```

Or from source:

```bash
go install github.com/vaughnbosu/cws-cli/cmd/cws@latest
```

## Quick Start

```bash
cws init --global   # credential setup with browser sign-in
cws validate ./dist # pre-flight checks
cws upload ./dist   # validate, zip, and upload
cws publish         # publish to the store
```

## Commands

| Command | Description |
|---------|-------------|
| `cws init [--global]` | Interactive credential setup wizard (browser sign-in) |
| `cws login` | Re-acquire a refresh token via browser sign-in |
| `cws validate [source]` | Pre-flight validation (manifest, version, icons, size, policy flags) |
| `cws pack [source]` | Zip an extension directory without uploading |
| `cws upload [source]` | Validate, zip, and upload a package (directory, .zip, or .crx) |
| `cws publish` | Publish the latest uploaded version |
| `cws status` | Check extension status, policy warnings, and takedowns |
| `cws rollout <percentage>` | Set deploy percentage (10k+ users required) |
| `cws cancel` | Cancel a pending submission |
| `cws version` | Print CLI version |

Store and packaging commands accept `--json` for machine-readable output.
Use `-e/--extension-id` to override the target extension or `--ext <name>` to
select a named profile from `cws.toml`.

### Validate

Run pre-flight checks before uploading:

```bash
cws validate ./dist          # full validation (local + remote)
cws validate ./dist --local  # local checks only (no credentials needed)
```

Checks include: manifest.json validity, required fields, version format, icon
files, package size, version higher than the published *and* any submitted
revision, no pending submission, and policy warnings/takedowns.

Validation runs automatically before every `cws upload`. Use `--skip-validate` to bypass.

### Publish

```bash
cws publish                        # publish after review
cws publish --staged               # submit for review without auto-publishing
cws publish --skip-review          # attempt to skip review (eligible changes only)
cws publish --block-on-warnings    # fail if the store reports warnings
cws publish --deploy-percentage 10 # publish with an initial partial rollout
```

Non-blocking store warnings are printed after every publish.

## Configuration

Config priority: **CLI flags > env vars (`CWS_*`) > local `cws.toml` > global `~/.config/cws/cws.toml`**.

Keep credentials in the global config:

```bash
cws init --global
```

This writes OAuth credentials and the publisher ID to
`~/.config/cws/cws.toml`. A project `cws.toml` can then contain only extension
and packaging settings, so it is safe to commit:

```toml
[extensions.default]
id = "abcdefghijklmnopabcdefghijklmnop"
source = "./dist"

# Select with --ext beta
[extensions.beta]
id = "ponmlkjihgfedcbaponmlkjihgfedcba"
source = "./dist-beta"

# Optional packaging controls
[package]
exclude = ["docs", ".log"]      # extra exclusions
include = ["package.json"]      # keep files the defaults would drop
```

Running `cws init` without `--global` writes credentials to `./cws.toml` and
adds that file to `.gitignore`. Do not commit a local config containing secrets.
Generated `.zip` and `.crx` packages are excluded from directory builds by
default; list one under `package.include` only when it is intentionally part of
the extension.

See [cws.toml.example](cws.toml.example) for the full reference.

## CI/CD

### GitHub Action

```yaml
- name: Upload and publish extension
  uses: vaughnbosu/cws-cli@v1.3.1
  with:
    version: v1.3.1
    args: upload ./dist --publish
    client-id: ${{ secrets.CWS_CLIENT_ID }}
    client-secret: ${{ secrets.CWS_CLIENT_SECRET }}
    refresh-token: ${{ secrets.CWS_REFRESH_TOKEN }}
    publisher-id: ${{ secrets.CWS_PUBLISHER_ID }}
    extension-id: ${{ vars.EXTENSION_ID }}
```

The Action supports Linux and macOS runners on amd64 or arm64. The CLI also
ships standalone Windows binaries.

### Any other CI

Install the binary and drive it with `CWS_*` env vars:

```bash
curl -fsSL https://vaughnbosu.github.io/cws-cli/install.sh | bash
cws upload ./dist --publish --json
```

## Development

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
go build ./...

# Opt-in live checks (hit Google endpoints; safe, read-only)
CWS_LIVE_CONTRACT=1 go test ./tests/ -run TestLiveDiscoveryContract
```

## License

MIT
