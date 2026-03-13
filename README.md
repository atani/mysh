# mysh

MySQL connection manager with SSH tunnel support.

![demo](demo.gif)

## Features

- Interactive connection setup with encrypted password storage (AES-256-GCM + Argon2id)
- SSH tunnel management (ad-hoc and background persistent tunnels)
- Automatic output masking for AI/non-TTY execution (protects personal data in production)
- Multiple connections across different terminals without conflicts
- mycli preferred, falls back to standard mysql client
- Output format conversion (plain, markdown, CSV, PDF) with file export
- MySQL 5.1+ compatible

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

### Output Formats

Export query results as markdown, CSV, or PDF.

```bash
# Markdown table
mysh run production -e "SELECT * FROM users LIMIT 5" --format markdown

# CSV file
mysh run production -e "SELECT * FROM users" --format csv -o users.csv

# PDF report
mysh run production -e "SELECT * FROM users" --format pdf -o report.pdf

# tables command also supports format/output
mysh tables production --format csv -o tables.csv
```

Supported formats: `plain` (default), `markdown` (`md`), `csv`, `pdf`

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

  - name: local
    env: development
    db:
      host: localhost
      port: 3306
      user: root
      database: myapp_dev
```

## Security

- Database passwords are encrypted with AES-256-GCM
- Key derivation uses Argon2id (memory-hard, resistant to GPU attacks)
- Master password is stored in macOS Keychain (falls back to prompt on other platforms)
- Config files are created with `0600` permissions
- Production query output is always masked when mask rules are configured
- `--raw` on production requires interactive TTY confirmation (AI tools cannot bypass)

## Dependencies

- `golang.org/x/crypto` - Argon2id key derivation
- `golang.org/x/term` - Secure password input and TTY detection
- `gopkg.in/yaml.v3` - Configuration file parsing
