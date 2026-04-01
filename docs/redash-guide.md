# Redash Integration Guide

Query production databases through Redash with automatic output masking. No database credentials or SSH tunnels required.

## Overview

```
You / Claude Code → mysh → Redash API → Database
                      ↓
                  Masking applied
                      ↓
                  Safe results
```

mysh sends your SQL query to Redash, receives the results, applies masking rules, and returns the sanitized output. Sensitive data never reaches your AI assistant.

## Setup

### 1. Get your Redash API key

1. Log in to your Redash instance
2. Click your profile icon → **Settings**
3. Copy the **API Key**

### 2. Find the data source ID

Ask your engineer, or find it in Redash:

1. Go to **Settings** → **Data Sources**
2. Click the data source you want to query
3. The ID is in the URL: `https://redash.example.com/data_sources/3` → ID is `3`

### 3. Add the connection

```bash
mysh add --name prod \
  --redash-url https://redash.example.com \
  --redash-key YOUR_API_KEY \
  --redash-datasource 3
```

You'll be prompted for:
- **Environment**: Choose `production` to enable automatic masking
- **Mask columns**: Press Enter to accept the defaults (`email,phone,*password*,*secret*,*token*,*address*`)

### 4. Verify

```bash
mysh ping prod
# Connection "prod" (Redash): OK (85ms)
```

## Usage

### Run queries

```bash
# Inline SQL
mysh run prod -e "SELECT * FROM users LIMIT 5"

# From a SQL file
mysh run prod query.sql
```

### Output formats

```bash
# CSV
mysh run prod -e "SELECT * FROM users" --format csv

# JSON
mysh run prod -e "SELECT * FROM users" --format json

# Markdown table
mysh run prod -e "SELECT * FROM users" --format markdown

# Save to file
mysh run prod -e "SELECT * FROM users" --format csv -o users.csv
```

### With Claude Code

Just describe what you need:

> "Show me a breakdown of active subscriptions by plan type"

Claude Code generates the SQL automatically and runs it through mysh.

## Masking

Masking works identically to direct database connections. Columns matching your mask rules are automatically redacted:

| Type | Original | Masked |
|------|----------|--------|
| Email | alice@example.com | a\*\*\*@example.com |
| Phone | 090-1234-5678 | 0\*\*\* |
| Name | Alice | A\*\*\* |

Production connections always mask, regardless of how the query is executed (terminal, script, or AI assistant).

## Non-Interactive Setup

For scripted or automated setup (e.g., provisioning new team members):

```bash
export MYSH_MASTER_PASSWORD="the-master-password"

mysh add --name prod \
  --redash-url https://redash.example.com \
  --redash-key "$REDASH_API_KEY" \
  --redash-datasource 3 \
  --env production \
  --mask "email,phone,*password*,*secret*,*token*,*address*"
```

## Multiple Data Sources

You can add multiple Redash connections for different databases:

```bash
mysh add --name analytics \
  --redash-url https://redash.example.com \
  --redash-key YOUR_KEY \
  --redash-datasource 5 \
  --env production \
  --mask "email,phone"

mysh add --name logs \
  --redash-url https://redash.example.com \
  --redash-key YOUR_KEY \
  --redash-datasource 8 \
  --env staging
```

```bash
mysh run analytics -e "SELECT count(*) FROM events WHERE date = CURDATE()"
mysh run logs -e "SELECT * FROM access_logs LIMIT 10"
```

## Sharing Connections

Engineers can export Redash connections for team distribution:

```bash
# Export (API key is excluded)
mysh export prod > prod-redash.yaml

# Team member imports and enters their own API key
mysh import --from yaml --file prod-redash.yaml
```

## Troubleshooting

### "redash API returned HTTP 403"

- Your API key is invalid, expired, or lacks permission for the requested data source
- Generate a new API key from your Redash profile settings

### "redash API returned HTTP 400"

- The SQL query has a syntax error
- The data source ID is incorrect

### "redash API request failed: connection refused"

- Check the Redash URL is correct
- You may need to be connected to your company's VPN

### "redash query failed: ..."

- Redash executed the query but it failed (e.g., table not found, permission denied)
- Check the error message for details

### Query takes a long time

- Redash has its own query timeout (typically 5 minutes)
- mysh waits for the Redash job to complete, polling every 500ms
- Consider adding `LIMIT` to large queries

### Masked columns not matching

- Check your mask configuration: `mysh edit prod`
- Column names are matched case-insensitively
- Patterns support wildcards: `*address*` matches `home_address`, `email_address`, etc.
