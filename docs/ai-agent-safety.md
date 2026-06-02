# Using mysh safely with AI coding agents

AI coding agents are useful for debugging database-backed applications, but raw query output can easily contain personal data, credentials, or production-only identifiers. mysh is designed to reduce that risk when you query MySQL from tools such as Claude Code, Cursor, shell scripts, and other non-interactive environments.

## The problem

Traditional database clients assume a human is looking at the terminal. In AI-assisted workflows, command output is often captured and sent back to an agent. That makes accidental leakage more likely:

- emails and phone numbers in user tables
- names, addresses, and profile fields
- access tokens, reset tokens, and secret-like columns
- customer or internal business data copied into prompts

## The mysh approach

mysh combines database access with output safeguards:

1. Store connection profiles locally.
2. Open SSH tunnels when needed.
3. Detect production and non-TTY contexts.
4. Mask configured sensitive columns and patterns before output is returned.
5. Require interactive confirmation before raw production output is shown.

## Example

```bash
mysh run production -e "SELECT id, name, email, phone FROM users LIMIT 2" --format markdown
```

Example output in an AI/non-TTY context:

| id | name | email | phone |
|----|------|-------|-------|
| 1 | A*** | a***@example.com | 0*** |
| 2 | B*** | b***@example.com | 0*** |

## Recommended configuration

```yaml
connections:
  - name: production
    env: production
    mask:
      columns:
        - email
        - phone
        - name
        - password_hash
        - reset_token
      patterns:
        - "*address*"
        - "*secret*"
        - "*token*"
    ssh: ...
    db: ...
```

## Suggested workflow with Claude Code or Cursor

1. Configure the connection once with `mysh add` or import from an existing database client.
2. Mark production-like databases with `env: production`.
3. Define masking rules for your schema.
4. Ask the AI tool to use `mysh run ... --format markdown` instead of raw `mysql` commands.
5. Review and refine masking rules as new sensitive columns appear.

## Production raw output

For production connections, `--raw` requires an interactive terminal confirmation. This is intentional: a non-interactive AI agent or script should not be able to silently bypass masking.

## What mysh does not solve

mysh reduces accidental leakage from CLI query output, but it is not a complete data governance system. You should still:

- use least-privilege database accounts
- avoid broad `SELECT *` queries on sensitive tables
- keep production access audited
- avoid pasting raw logs or screenshots containing personal data into AI tools
- review masking configuration for your own schema

## Quick copy for team docs

> Use `mysh` for AI-assisted MySQL queries. It manages SSH tunnels and connection profiles, and masks configured sensitive output in production/non-TTY contexts before the result reaches the AI agent.
