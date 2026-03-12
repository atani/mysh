# mysh

MySQL connection manager with SSH tunnel support.

## Features

- Interactive connection setup with encrypted password storage (AES-256-GCM + Argon2id)
- SSH tunnel management (ad-hoc and background persistent tunnels)
- Multiple connections across different terminals without conflicts
- mycli preferred, falls back to standard mysql client
- MySQL 5.1+ compatible

## Install

```bash
go install github.com/atani/mysh@latest
```

## Quick Start

```bash
# Add a connection interactively
mysh add

# List connections
mysh list

# Connect
mysh connect production
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

# Remove a connection
mysh remove production
```

### Connecting

```bash
# Open an interactive MySQL session
mysh connect production

# Execute a SQL file
mysh run production query.sql

# Execute inline SQL
mysh run production -e "SELECT COUNT(*) FROM users"
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
    ssh:
      host: bastion.example.com
      port: 22
      user: deploy
      key: ~/.ssh/id_ed25519
    db:
      host: 127.0.0.1
      port: 3306
      user: app
      database: myapp_production
      password: <encrypted>

  - name: local
    db:
      host: localhost
      port: 3306
      user: root
      database: myapp_dev
```

## Security

- Database passwords are encrypted with AES-256-GCM
- Key derivation uses Argon2id (memory-hard, resistant to GPU attacks)
- A master password is required to encrypt/decrypt credentials
- Config files are created with `0600` permissions

## Dependencies

- `golang.org/x/crypto` - Argon2id key derivation
- `golang.org/x/term` - Secure password input
- `gopkg.in/yaml.v3` - Configuration file parsing
