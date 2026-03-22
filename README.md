# mysh

[![CI](https://github.com/atani/mysh/actions/workflows/ci.yml/badge.svg)](https://github.com/atani/mysh/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/atani/mysh)](https://goreportcard.com/report/github.com/atani/mysh)
[![Go Reference](https://pkg.go.dev/badge/github.com/atani/mysh.svg)](https://pkg.go.dev/github.com/atani/mysh)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

MySQL connection manager with SSH tunnel support.

![demo](demo.gif)

## Features

- Interactive connection setup with encrypted password storage (AES-256-GCM + Argon2id)
- SSH tunnel management (ad-hoc and background persistent tunnels)
- Automatic output masking for AI/non-TTY execution (protects personal data in production)
- Multiple connections across different terminals without conflicts
- mycli preferred, falls back to standard mysql client
- Native Go driver with MySQL 4.x old_password authentication support
- Output format conversion (plain, markdown, CSV, JSON, PDF) with file export
- MySQL 4.x+ compatible (native driver) / MySQL 5.1+ compatible (CLI driver)

## Install

```bash
brew tap atani/tap
brew install mysh
```

Or with Go:

```bash
go install github.com/atani/mysh@latest
```

## Quick Start

```bash
# Add a connection interactively
mysh add

# Test connection (name optional if only one connection)
mysh ping

# Connect
mysh connect
```

## Usage

```
mysh <command> [arguments]
```

### Connection Management

```bash
# Add a new connection (interactive)
mysh add

# List all connections
mysh list

# Edit an existing connection
mysh edit production

# Remove a connection
mysh remove production
```

Connection name can be omitted when only one connection exists.

### Connecting & Querying

```bash
# Test connection
mysh ping production

# Open an interactive MySQL session
mysh connect production

# Execute a SQL file
mysh run production query.sql

# Execute inline SQL
mysh run production -e "SELECT COUNT(*) FROM users"

# Show tables
mysh tables production
```

### SSH Tunnels

By default, `connect` and `run` open an ad-hoc SSH tunnel that closes when the command finishes.

For repeated access, start a persistent background tunnel:

```bash
# Start a background tunnel
mysh tunnel production

# List active tunnels
mysh tunnel

# connect/run will automatically reuse the background tunnel
mysh run production -e "SHOW PROCESSLIST"

# Stop a tunnel
mysh tunnel stop production
```

Multiple tunnels can run simultaneously for different connections.

### Output Masking

mysh can automatically mask sensitive columns (email, phone, etc.) in query output. This is designed to prevent personal data from leaking into AI tool contexts.

#### How it works

Masking is controlled by two factors:

1. **Connection environment** (`env` in config)
2. **TTY detection** (is output going to a terminal or being captured?)

| env | Terminal (human) | Piped/captured (AI) |
|-----|-----------------|---------------------|
| production | **Auto-masked** | **Auto-masked** |
| staging | Raw | **Auto-masked** |
| development | Raw | Raw |

#### Configuration

```yaml
connections:
  - name: production
    env: production
    mask:
      columns: ["email", "phone", "password_hash"]
      patterns: ["*address*", "*secret*"]
    ssh: ...
    db: ...
```

#### Manual override

```bash
# Force masking (even in terminal)
mysh run production --mask -e "SELECT * FROM users LIMIT 5"

# Force raw output (requires interactive confirmation for production)
mysh run production --raw -e "SELECT * FROM users LIMIT 5"
```

For production connections, `--raw` requires interactive confirmation at the terminal. Non-TTY processes (AI tools, scripts) cannot bypass masking.

#### Masking examples

| Type | Original | Masked |
|------|----------|--------|
| Email | alice@example.com | a\*\*\*@example.com |
| Phone | 090-1234-5678 | 0\*\*\* |
| Name | Alice | A\*\*\* |
| Short value | ab | \*\*\* |
| NULL | NULL | NULL |

### Record Slicing

Extract specific records from a database as INSERT statements. Useful for creating reproducible test data or migrating individual records.

```bash
# Extract records matching a condition
mysh slice production products --where "category='electronics'"

# Save to file
mysh slice production products --where "id IN (7,8)" -o subset.sql

# Disable masking (requires interactive confirmation)
mysh slice production customers --where "id=3" --raw
```

Output example:

```sql
-- mysh slice: products WHERE category='electronics'
-- Generated at: 2026-03-15T12:00:00+09:00

INSERT INTO `products` (`id`, `name`, `price`) VALUES (7, 'Widget Pro', 2980);
INSERT INTO `products` (`id`, `name`, `price`) VALUES (8, 'Gadget Mini', NULL);
```

Masking rules from the connection config are always applied by default, regardless of environment. Use `--raw` to disable (requires interactive confirmation).

### Output Formats

Export query results as markdown, CSV, JSON, or PDF.

```bash
# Markdown table
mysh run production -e "SELECT * FROM users LIMIT 5" --format markdown

# CSV file
mysh run production -e "SELECT * FROM users" --format csv -o users.csv

# JSON output
mysh run production -e "SELECT * FROM users LIMIT 5" --format json

# JSON file
mysh run production -e "SELECT * FROM users" --format json -o users.json

# PDF report
mysh run production -e "SELECT * FROM users" --format pdf -o report.pdf

# tables command also supports format/output
mysh tables production --format csv -o tables.csv
```

Supported formats: `plain` (default), `markdown` (`md`), `csv`, `json`, `pdf`

### Saved Queries

Save `.sql` files in `~/.config/mysh/queries/` and list them with:

```bash
mysh queries
```

## Configuration

Connections are stored in `~/.config/mysh/connections.yaml`.

```yaml
connections:
  - name: production
    env: production
    ssh:
      host: bastion.example.com
      port: 22
      user: deploy
      key: ~/.ssh/id_ed25519
    mask:
      columns: [email, phone]
      patterns: ["*address*"]
    db:
      host: 127.0.0.1
      port: 3306
      user: app
      database: myapp_production
      password: <encrypted>

  - name: legacy-db
    env: production
    db:
      host: 10.0.0.5
      port: 3306
      user: app
      database: legacy_production
      password: <encrypted>
      driver: native  # MySQL 4.x old_password support

  - name: local
    env: development
    db:
      host: localhost
      port: 3306
      user: root
      database: myapp_dev
```

### Connection Driver

The `driver` field selects how mysh connects to MySQL.

| driver | Description | Supported versions |
|--------|-------------|-------------------|
| `cli` (default) | Delegates to mysql/mycli CLI | MySQL 5.1+ |
| `native` | Direct connection via Go's database/sql | MySQL 4.x+ |

The `native` driver uses `go-sql-driver/mysql` with `allowOldPasswords=true` to support MySQL 4.x old_password (mysql323) authentication. The `connect` command provides a simple REPL instead of mycli/mysql.

## Security

- Database passwords are encrypted with AES-256-GCM
- Key derivation uses Argon2id (memory-hard, resistant to GPU attacks)
- Master password is stored in macOS Keychain (falls back to prompt on other platforms)
- Config files are created with `0600` permissions
- Production query output is always masked when mask rules are configured
- `--raw` on production requires interactive TTY confirmation (AI tools cannot bypass)

## Caveats

- **old_password is cryptographically weak**: MySQL 4.x old_password (mysql323 hash) is a 16-byte XOR-based hash that does not meet modern security standards. Use the native driver only for legacy system connectivity.
- **Native driver `connect` limitations**: Unlike mycli/mysql CLI, the built-in REPL has no tab completion, syntax highlighting, or pager. For complex interactive work, prefer `run -e`.
- **go-sql-driver/mysql `allowOldPasswords`**: This depends on the driver's support, which may be removed in future driver updates.

## Dependencies

- `golang.org/x/crypto` - Argon2id key derivation
- `golang.org/x/term` - Secure password input and TTY detection
- `gopkg.in/yaml.v3` - Configuration file parsing
- `github.com/go-sql-driver/mysql` - Native MySQL driver (old_password support)
- `github.com/go-pdf/fpdf` - PDF output
