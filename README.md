# api-key-rotate

One command to rotate leaked API keys everywhere. Find keys in local `.env` files and cloud providers, update them all at once, and roll back if anything fails.

## Install

```bash
go install github.com/sudokatie/api-key-rotate/cmd/api-key-rotate@latest
```

Or build from source:

```bash
git clone https://github.com/sudokatie/api-key-rotate
cd api-key-rotate
make build
```

## Quick Start

Find where a key exists:

```bash
api-key-rotate find OPENAI_API_KEY
```

Rotate it (dry run first):

```bash
api-key-rotate rotate OPENAI_API_KEY
```

Actually rotate it:

```bash
api-key-rotate rotate OPENAI_API_KEY --execute
```

## Commands

### find

Search local files and cloud providers for a key:

```bash
api-key-rotate find STRIPE_SECRET_KEY
api-key-rotate find DATABASE_URL --local-only
api-key-rotate find API_KEY --format json
```

### rotate

Update a key across all locations:

```bash
# Dry run (default)
api-key-rotate rotate MY_KEY

# Execute rotation
api-key-rotate rotate MY_KEY --execute

# Provide new key value
api-key-rotate rotate MY_KEY --execute --new-key="sk_live_xxx"

# Skip confirmation
api-key-rotate rotate MY_KEY --execute --force

# Filter locations
api-key-rotate rotate MY_KEY --execute --skip-cloud
api-key-rotate rotate MY_KEY --execute --skip-local
api-key-rotate rotate MY_KEY --execute --locations="project-a,production"
api-key-rotate rotate MY_KEY --execute --exclude="staging"
```

### history

View rotation audit log:

```bash
api-key-rotate history
api-key-rotate history --key OPENAI_API_KEY
api-key-rotate history --since 2024-01-01 --status failed
```

### config

Manage configuration:

```bash
api-key-rotate config show
api-key-rotate config init
api-key-rotate config set ui.color false
api-key-rotate config scan-paths add ~/work
```

### providers

Manage cloud providers:

```bash
api-key-rotate providers list
api-key-rotate providers add vercel
api-key-rotate providers test
```

## Configuration

Config file: `~/.config/api-key-rotate/config.yaml`

```yaml
scan_paths:
  - ~/projects
  - ~/work

exclude_patterns:
  - node_modules
  - .git
  - vendor
  - __pycache__

file_patterns:
  - .env
  - .env.*
  - "*.env"

providers:
  vercel:
    enabled: true
  github:
    enabled: true
    orgs:
      - myorg

ui:
  color: true
  verbose: false

audit:
  path: ~/.local/share/api-key-rotate/audit.db
  retention_days: 365
```

## Provider Setup

### Vercel

1. Create a token at https://vercel.com/account/tokens
2. Add it:

```bash
api-key-rotate providers add vercel
```

Or use environment variable:

```bash
export VERCEL_TOKEN=your-token
```

### GitHub

1. Create a PAT with `repo` scope at https://github.com/settings/tokens
2. Add it:

```bash
api-key-rotate providers add github
```

Or use environment variable:

```bash
export GITHUB_TOKEN=your-token
```

### Token Priority

Environment variables take precedence over keychain. This makes it easy to use in CI/CD:

```bash
VERCEL_TOKEN=${{ secrets.VERCEL_TOKEN }} api-key-rotate find MY_KEY
```

## How It Works

1. **Find** - Scans local `.env` files and queries cloud provider APIs
2. **Backup** - Creates backups of all local files before modification
3. **Update** - Atomically updates each location with the new key value
4. **Rollback** - If any update fails, reverts all successful changes
5. **Audit** - Logs the rotation to SQLite for history tracking

## Reliability

- Automatic retry on rate limits (429) with exponential backoff
- Automatic retry on server errors (5xx) up to 3 times
- Atomic file writes prevent corruption
- Full rollback on any failure

## Security

- Provider tokens stored in system keychain (not in config files)
- Environment variables supported for CI/CD (`VERCEL_TOKEN`, `GITHUB_TOKEN`)
- New key values prompted without echo (like passwords)
- Backup files have 0600 permissions
- Audit log never stores full key values (only previews like `sk_l****`)

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Provider error
- `4` - Key not found
- `5` - Rotation failed
- `6` - Rollback failed (critical)

## License

MIT
