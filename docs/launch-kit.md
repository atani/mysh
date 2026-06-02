# mysh launch kit

Use these snippets when introducing mysh to English-speaking developer communities.

## One-line positioning

> mysh is a safer MySQL CLI for the AI coding era: connection profiles, SSH tunnels, and automatic sensitive-data masking before query output reaches AI agents or scripts.

## Short description

mysh manages MySQL connections and SSH tunnels from the command line, imports profiles from tools like DBeaver, Sequel Ace, and MySQL Workbench, and automatically masks sensitive query output in production/non-TTY contexts. It is designed for teams using Claude Code, Cursor, scripts, and other AI-assisted workflows around production-like databases.

## Show HN title ideas

- Show HN: mysh – a safer MySQL CLI for AI coding agents
- Show HN: I built a MySQL CLI that masks sensitive data before AI tools see it
- Show HN: mysh – MySQL connections, SSH tunnels, and AI-safe output masking

## Show HN post body

I built mysh, a MySQL CLI for teams that use AI coding agents and shell scripts around production-like databases.

It handles:

- MySQL connection profiles
- ad-hoc and persistent SSH tunnels
- encrypted local password storage
- imports from DBeaver, Sequel Ace, and MySQL Workbench
- Markdown/CSV/JSON/PDF exports
- automatic masking of sensitive query output in production or non-TTY contexts

The main motivation was simple: traditional MySQL clients are designed for humans, but AI-assisted workflows often capture command output and send it back to an agent. mysh tries to make that safer by masking configured columns/patterns before output leaves the CLI. Production `--raw` requires an interactive confirmation, so non-interactive AI/script execution cannot silently bypass it.

It is written in Go, MIT licensed, and installable via Homebrew, winget/MSI, standalone binaries, or `go install`.

GitHub: https://github.com/atani/mysh

## Reddit: r/golang

Title:

> I built mysh, a Go CLI for MySQL connections, SSH tunnels, and AI-safe output masking

Body:

I built `mysh`, a Go-based MySQL CLI focused on safer database workflows in the AI coding era.

It provides connection profiles, SSH tunnel management, encrypted local password storage, imports from DBeaver/Sequel Ace/MySQL Workbench, output export formats, and automatic masking for sensitive query output when used in production or non-TTY contexts.

The masking piece is the part I care about most: when command output is captured by Claude Code, Cursor, scripts, or other AI-assisted tooling, mysh can mask emails/phones/names/custom patterns before the result is returned.

Repo: https://github.com/atani/mysh

I would love feedback from Go/CLI folks, especially around security, UX, and cross-platform behavior.

## Reddit: AI coding communities

Title:

> I made a MySQL CLI that masks sensitive query output before AI agents see it

Body:

I often want AI coding tools to help inspect database-backed behavior, but raw query output can easily contain emails, phone numbers, names, or other personal data.

So I built `mysh`: a MySQL CLI that manages connection profiles and SSH tunnels, while automatically masking sensitive query output in production/non-TTY contexts before it reaches tools like Claude Code or Cursor.

Example use case:

```bash
mysh run production -e "SELECT id, name, email, phone FROM users LIMIT 10" --format markdown
```

Instead of returning raw PII, configured fields are masked. Production `--raw` requires interactive confirmation, so an AI agent cannot silently bypass it.

Repo: https://github.com/atani/mysh

Feedback welcome, especially from people using AI agents with real database-backed applications.

## X / Bluesky post

I built `mysh`: a safer MySQL CLI for the AI coding era.

- connection profiles
- SSH tunnels
- encrypted local passwords
- imports from DBeaver / Sequel Ace / Workbench
- Markdown/CSV/JSON/PDF exports
- auto-masks sensitive query output before AI agents/scripts see it

MIT, written in Go.

https://github.com/atani/mysh

## Communities to try

Start with smaller, relevant communities before posting to Hacker News:

1. r/ClaudeAI or other AI coding communities
2. r/golang
3. r/commandline
4. r/mysql
5. Lobsters
6. Show HN

Avoid cross-posting everywhere at once. Use feedback from the first posts to improve the README before submitting to larger communities.
