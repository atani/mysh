# Importing Connections

Import MySQL connection settings from DBeaver, Sequel Ace, or MySQL Workbench into mysh.

## Supported Tools

| Tool | Config file location (macOS) |
|------|---------------------|
| DBeaver | `~/Library/DBeaverData/workspace6/General/.dbeaver/data-sources.json` |
| Sequel Ace | `~/Library/Containers/com.sequel-ace.sequel-ace/Data/Library/Application Support/Sequel Ace/Data/Favorites.plist` |
| MySQL Workbench | `~/Library/Application Support/MySQL/Workbench/connections.xml` |

DBeaver and MySQL Workbench also support Linux paths:

| Tool | Config file location (Linux) |
|------|---------------------|
| DBeaver | `~/.local/share/DBeaverData/workspace6/General/.dbeaver/data-sources.json` |
| MySQL Workbench | `~/.mysql/workbench/connections.xml` |

## Basic Usage

```bash
# Import from DBeaver
mysh import --from dbeaver

# Import from Sequel Ace
mysh import --from sequel-ace

# Import from MySQL Workbench
mysh import --from workbench
```

This displays a list of discovered connections:

```
DBeaver: 5 connection(s) found

  #  NAME              HOST              PORT  USER   DATABASE  SSH
  1  my-production     10.51.80.122      3306  admin  myapp     bastion.example.com
  2  server-db         db-replica.lan    3306  admin  hosting   bastion.example.com
  3  local-db          127.0.0.1         33306 root   devdb     -

Select connections (comma-separated numbers, or 'all') [all]:
```

Enter comma-separated numbers to pick specific connections, or `all` to import everything.

## Import Flow

For each connection, you'll be prompted for only the essentials:

1. **Connection name** — only if it conflicts with an existing name
2. **SSH user** — only if missing from the source config
3. **Password** — must be re-entered for security (press Enter to skip)

After import, you'll be asked whether to apply default output masking settings.

## Passwords

Both DBeaver and Sequel Ace protect passwords with encryption or macOS Keychain, so they cannot be imported automatically. Enter the password during import, or skip and set it later with `mysh edit`.

## Post-Import Setup

After import, mysh asks whether to apply default output masking:

```
Default mask columns: email,phone,*password*,*secret*,*token*,*address*
Apply output masking to protect sensitive data? [Y/n]:
```

- **Yes** — applies default mask rules and sets environment to `production` for all imported connections
- **No** — connections are imported as `development` with no masking; configure later with `mysh edit`

You can fine-tune environment, mask columns, and driver per connection:

```bash
mysh edit <connection-name>
```

| Setting | Description |
|---------|-------------|
| Environment | production / staging / development |
| Mask columns | email, phone, etc. (wildcards supported) |
| Driver | cli (default) / native (MySQL 4.x support) |

Masking behavior by environment:
- **production** — always masks sensitive columns
- **staging** — masks when output is piped (e.g., to AI tools)
- **development** — no masking

## Import All

Use `--all` to skip the selection prompt and import all discovered connections:

```bash
mysh import --from dbeaver --all
```

## What Gets Imported

| Field | DBeaver | Sequel Ace | MySQL Workbench |
|-------|---------|------------|-----------------|
| Host | ✅ | ✅ | ✅ |
| Port | ✅ | ✅ | ✅ |
| User | ✅ | ✅ | ✅ |
| Database | ✅ | ✅ | ✅ |
| Password | ❌ (re-enter) | ❌ (re-enter) | ❌ (re-enter) |
| SSH host | ✅ | ✅ | ✅ |
| SSH port | ✅ | ✅ | ✅ |
| SSH user | △ (may be missing) | ✅ | ✅ |
| SSH key path | ✅ | ✅ | ✅ |

## Migrating from DBeaver

1. DBeaver can be running — mysh only reads the config file
2. Run `mysh import --from dbeaver`
3. Select connections to import
4. Enter passwords (or press Enter to skip)
5. Choose whether to apply default masking
6. Verify with `mysh list` and `mysh ping <name>`

## Migrating from Sequel Ace

1. Ensure connections are saved in Sequel Ace Favorites
2. Run `mysh import --from sequel-ace`
3. Select connections to import
4. Enter passwords (or press Enter to skip)
5. Choose whether to apply default masking
6. Verify with `mysh list` and `mysh ping <name>`

> **Note**: Sequel Ace import uses macOS `plutil` command (included with macOS).

## Migrating from MySQL Workbench

1. Run `mysh import --from workbench`
2. Select connections to import
3. Enter passwords (or press Enter to skip)
4. Choose whether to apply default masking
5. Verify with `mysh list` and `mysh ping <name>`

## Troubleshooting

### "No MySQL connections found"

- **DBeaver**: Check that the DBeaver data directory exists. The path may differ across DBeaver versions.
- **Sequel Ace**: Check that `Favorites.plist` exists at the expected location.
- **MySQL Workbench**: Check that `connections.xml` exists in the Workbench data directory.

### Prompted for SSH user

DBeaver sometimes omits the SSH user from `data-sources.json` (it uses the OS username by default). mysh requires an explicit SSH user, so you'll be prompted during import.

### Setting password after import

```bash
mysh edit <connection-name>
```

All settings including password can be updated after import.
