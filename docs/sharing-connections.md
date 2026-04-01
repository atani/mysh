# Sharing Connections

Export and import mysh connection configurations to onboard team members quickly.

## Overview

Engineers configure connections once and share them with the team. Recipients only need to enter their own credentials — all other settings (environment, SSH, masking rules) are inherited.

```
Engineer                          Team member
──────────                        ───────────
mysh export prod > prod.yaml  →  mysh import --from yaml --file prod.yaml
                                  → enter password
                                  → done
```

## Exporting

### Export a specific connection

```bash
mysh export prod > prod.yaml
```

### Export all connections

```bash
mysh export > all-connections.yaml
```

### What's included

| Field | Included | Notes |
|-------|----------|-------|
| Connection name | Yes | |
| Environment | Yes | production, staging, development |
| Database host | Yes | |
| Database port | Yes | |
| Database user | Yes | |
| Database name | Yes | |
| Database password | **No** | Always excluded for security |
| SSH settings | Yes | host, port, user, key path |
| Mask rules | Yes | columns and patterns |
| Driver | Yes | cli or native |
| Redash URL | Yes | For Redash connections |
| Redash API key | **No** | Always excluded for security |
| Redash data source ID | Yes | For Redash connections |

### Example output

```yaml
- name: production
  env: production
  ssh:
    host: bastion.example.com
    user: deploy
    key: ~/.ssh/id_ed25519
  db:
    host: 10.0.0.5
    port: 3306
    user: app
    database: myapp_production
  mask:
    columns:
      - email
      - phone
    patterns:
      - "*address*"
      - "*secret*"
```

## Importing

### From a YAML file

```bash
mysh import --from yaml --file prod.yaml
```

The import flow:
1. Displays the connections found in the file
2. Lets you select which ones to import
3. Prompts for the database password (or Redash API key)
4. Tests the connection
5. Saves the configuration

### Import all without prompts

```bash
mysh import --from yaml --file prod.yaml --all
```

### What recipients need

**For direct DB connections:**
- The YAML file from the engineer
- The database password

**For Redash connections:**
- The YAML file from the engineer
- Their own Redash API key (from Redash profile settings)

## Best Practices

### For engineers sharing connections

1. **Always set environment and mask rules** before exporting — recipients inherit these settings
2. **Use production environment** for connections that access real user data
3. **Include comprehensive mask patterns** to prevent accidental data exposure:
   ```bash
   mysh edit prod
   # Set mask to: email,phone,*password*,*secret*,*token*,*address*,*name*
   ```
4. **Share the YAML file securely** — while it contains no passwords, it does include hostnames and usernames

### For recipients

1. **Set `MYSH_MASTER_PASSWORD`** in your shell profile so AI assistants can use mysh without interactive prompts
2. **Test with `mysh ping`** after import to verify the connection works
3. **Don't modify mask settings** unless instructed by your engineer

## Sharing Redash Connections

Redash connections work the same way:

```bash
# Engineer exports
mysh export analytics > analytics.yaml

# Recipient imports (prompted for their own API key)
mysh import --from yaml --file analytics.yaml
```

Each team member uses their own Redash API key, so access is individually tracked in Redash audit logs.
