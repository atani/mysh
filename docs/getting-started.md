# Getting Started (for Non-Engineers)

A step-by-step guide to set up mysh and start querying production databases safely with AI assistants like Claude Code.

## Prerequisites

- A terminal application (Terminal on macOS, PowerShell on Windows)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or another AI coding assistant

## Install mysh

### macOS / Linux

```bash
brew tap atani/tap
brew install mysh
```

### Windows

1. Download `mysh-windows-amd64.exe` from the [latest release](https://github.com/atani/mysh/releases/latest)
2. Rename it to `mysh.exe`
3. Place it in a directory on your PATH (ask your engineer if unsure)

### Verify installation

```bash
mysh help
```

## Choose Your Setup Method

There are two ways to connect to databases with mysh:

| Method | Best for | What you need |
|--------|----------|---------------|
| [Redash](#option-a-redash-recommended) | Teams with Redash | Redash API key |
| [Direct DB](#option-b-direct-database-connection) | Teams without Redash | YAML file from your engineer |

## Option A: Redash (Recommended)

If your team uses [Redash](https://redash.io/), this is the easiest setup. You only need a Redash API key — no database credentials, no SSH, no tunnels.

### 1. Get your API key

1. Log in to your team's Redash
2. Click your profile icon (top right) → **Settings**
3. Find **API Key** and copy it

### 2. Add the connection

```bash
mysh add --name prod --redash-url https://redash.yourcompany.com --redash-key YOUR_API_KEY --redash-datasource 1
```

Your engineer will tell you the correct `--redash-datasource` number. If unsure, `1` is usually the main database.

### 3. Set up your master password

mysh will ask you to create a master password. This protects your stored API key. Remember it — you'll need it once per session.

To avoid entering the master password every time (especially when using AI assistants):

**macOS / Linux** — add to your shell profile (`~/.zshrc` or `~/.bashrc`):
```bash
export MYSH_MASTER_PASSWORD="your-master-password"
```

**Windows** — run in PowerShell:
```powershell
[Environment]::SetEnvironmentVariable("MYSH_MASTER_PASSWORD", "your-master-password", "User")
```

### 4. Test the connection

```bash
mysh ping prod
```

You should see:
```
Connection "prod" (Redash): OK (123ms)
```

### 5. Start querying

You can now ask Claude Code to query your database:

> "Show me the number of new user registrations by plan for last month"

Claude Code will use `mysh run prod -e "SELECT ..."` behind the scenes. Sensitive columns (email, phone, etc.) are automatically masked.

## Option B: Direct Database Connection

If your team doesn't use Redash, an engineer can export a connection configuration for you.

### 1. Get the configuration file

Ask your engineer to run:

```bash
mysh export prod > prod.yaml
```

They'll share the `prod.yaml` file with you (via Slack, email, etc.). This file contains connection settings but **no passwords**.

### 2. Import the configuration

```bash
mysh import --from yaml --file prod.yaml
```

You'll be prompted to enter the database password. Ask your engineer for it.

### 3. Set up your master password

Same as the Redash setup — create a master password and optionally set `MYSH_MASTER_PASSWORD` in your shell profile. See [Step 3 above](#3-set-up-your-master-password).

### 4. Test and start querying

```bash
mysh ping prod
```

Then ask Claude Code to query your database naturally.

## Using with Claude Code

Once mysh is set up, you can ask Claude Code questions like:

- "Show me the top 10 customers by order count this month"
- "How many support tickets were created last week?"
- "What's the distribution of users by plan?"

Claude Code will:
1. Write the SQL query for you
2. Run it through mysh (which applies masking automatically)
3. Analyze and present the results

### Safety features

- **Production data masking**: Email addresses, phone numbers, and other sensitive fields are automatically masked (e.g., `alice@example.com` → `a***@example.com`)
- **AI tools cannot bypass masking**: Even if Claude Code tries `--raw`, production masking cannot be disabled without interactive confirmation at the terminal
- **API keys are encrypted**: Your Redash API key is stored with AES-256-GCM encryption

## Troubleshooting

### "mysh: command not found"

mysh is not in your PATH. On macOS/Linux, try restarting your terminal after installing via Homebrew. On Windows, ensure the directory containing `mysh.exe` is in your PATH.

### "wrong master password"

You entered the wrong master password. If you've forgotten it, you'll need to reset by removing `~/.config/mysh/.master_check` and re-adding your connections.

### "redash API returned HTTP 403"

Your Redash API key is invalid or expired. Generate a new one from your Redash profile settings.

### "redash API request failed: connection refused"

Check the Redash URL. You may need to be on your company's VPN to access it.

### Connection test fails

For direct DB connections, ensure:
- You're on the correct network (VPN may be required)
- The password is correct
- SSH tunnel settings are correct (ask your engineer)

## Next Steps

- Run `mysh list` to see your configured connections
- Run `mysh help` for all available commands
- See [Redash Guide](redash-guide.md) for advanced Redash usage
- See [Import Guide](import-guide.md) for importing from DBeaver, Sequel Ace, or MySQL Workbench
