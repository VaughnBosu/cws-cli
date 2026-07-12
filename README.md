# cws — Chrome Web Store CLI

A single-binary CLI for managing Chrome Web Store extensions. Upload, publish, rollout, and more — from your terminal, powered by the latest V2 API.

**[Documentation](https://vaughnbosu.github.io/cws-cli/)**

## Install

```bash
brew install vaughnbosu/tap/cws
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
cws init            # credential setup with browser sign-in
cws validate ./dist # pre-flight checks
cws upload ./dist   # validate, zip, and upload
cws publish         # publish to the store
```

## Commands

| Command | Description |
|---------|-------------|
| `cws init` | Interactive credential setup wizard (browser sign-in) |
| `cws login` | Re-acquire a refresh token via browser sign-in |
| `cws validate [source]` | Pre-flight validation (manifest, version, icons, size, policy flags) |
| `cws pack [source]` | Zip an extension directory without uploading |
| `cws upload [source]` | Validate, zip, and upload a package (directory, .zip, or .crx) |
| `cws publish` | Publish the latest uploaded version |
| `cws status` | Check extension status, policy warnings, and takedowns |
| `cws rollout <percentage>` | Set deploy percentage (10k+ users required) |
| `cws cancel` | Cancel a pending submission |
| `cws version` | Print CLI version |

Every command accepts `--json` for machine-readable output, `-e/--extension-id`
to override the target extension, and `--ext <name>` to select a named extension
profile from `cws.toml`.

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

```toml
publisher_id = "abc1234567890"

[auth]
client_id = "xxxx.apps.googleusercontent.com"
client_secret = "GOCSPX-xxxx"
refresh_token = "1//xxxx"

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

See [cws.toml.example](cws.toml.example) for the full reference.

## CI/CD

### GitHub Action

```yaml
- name: Upload and publish extension
  uses: vaughnbosu/cws-cli@main
  with:
    args: upload ./dist --publish
    client-id: ${{ secrets.CWS_CLIENT_ID }}
    client-secret: ${{ secrets.CWS_CLIENT_SECRET }}
    refresh-token: ${{ secrets.CWS_REFRESH_TOKEN }}
    publisher-id: ${{ secrets.CWS_PUBLISHER_ID }}
    extension-id: ${{ vars.EXTENSION_ID }}
```

### Any other CI

Install the binary and drive it with `CWS_*` env vars:

```bash
curl -fsSL https://vaughnbosu.github.io/cws-cli/install.sh | bash
cws upload ./dist --publish --json
```

## Why cws?

| | `cws` | `chrome-webstore-upload-cli` |
|---|---|---|
| **Runtime** | Single binary — no dependencies | Requires Node.js + npm |
| **API version** | Chrome Web Store API **V2** | V1 ([migration requested](https://github.com/fregante/chrome-webstore-upload/issues/114)) |
| **Setup** | `cws init` wizard with browser sign-in | Manual env var configuration |
| **Commands** | validate, pack, upload, publish, status, rollout, cancel, login | upload, publish |
| **Pre-upload validation** | Built-in — manifest, version, icons, and size checks before upload | None |
| **Config** | TOML file (multi-extension) + env vars + CLI flags | Env vars only |
| **CI/CD** | GitHub Action or drop-in binary — no `npm install` step | Requires Node.js in your CI image |

## Development

```bash
go build ./...
go test ./...

# Opt-in live checks (hit Google endpoints; safe, read-only)
CWS_LIVE_CONTRACT=1 go test ./tests/ -run TestLiveDiscoveryContract
```

## License

MIT
