# mysh launch kit

Use these snippets when introducing mysh to English-speaking developer communities.

## One-line positioning

> mysh is a safer MySQL CLI for the AI coding era: connection profiles, SSH tunnels, and automatic sensitive-data masking before query output reaches AI agents or scripts.

Alternative shorter hook:

> Stop leaking production data to AI coding agents.

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
- a built-in MCP server so agents like Claude Code and Cursor query through the same masking rules

The main motivation was simple: traditional MySQL clients are designed for humans, but AI-assisted workflows often capture command output and send it back to an agent. mysh tries to make that safer by masking configured columns/patterns before output leaves the CLI. Production `--raw` requires an interactive confirmation, so non-interactive AI/script execution cannot silently bypass it, and the built-in MCP server never exposes raw output at all.

It is written in Go, MIT licensed, and installable via Homebrew, winget/MSI, standalone binaries, or `go install`.

GitHub: https://github.com/atani/mysh

## Reddit: r/golang

Title:

> I built mysh, a Go CLI for MySQL connections, SSH tunnels, and AI-safe output masking

Body:

I built `mysh`, a Go-based MySQL CLI focused on safer database workflows in the AI coding era.

It provides connection profiles, SSH tunnel management, encrypted local password storage, imports from DBeaver/Sequel Ace/MySQL Workbench, output export formats, a built-in MCP server for AI clients, and automatic masking for sensitive query output when used in production or non-TTY contexts.

The masking piece is the part I care about most: when command output is captured by Claude Code, Cursor, scripts, or other AI-assisted tooling, mysh can mask emails/phones/names/custom patterns before the result is returned. The MCP server runs through the same masking layer and never exposes raw output, so an agent cannot pull unmasked production data even when it drives the queries itself.

Repo: https://github.com/atani/mysh

I would love feedback from Go/CLI folks, especially around security, UX, and cross-platform behavior.

## Reddit: AI coding communities

Title:

> I made a MySQL CLI that masks sensitive query output before AI agents see it

Body:

I often want AI coding tools to help inspect database-backed behavior, but raw query output can easily contain emails, phone numbers, names, or other personal data.

So I built `mysh`: a MySQL CLI that manages connection profiles and SSH tunnels, while automatically masking sensitive query output in production/non-TTY contexts before it reaches tools like Claude Code or Cursor. It also ships a built-in MCP server, so an agent can connect directly and still only ever see masked results.

Example use case:

```bash
mysh run production -e "SELECT id, name, email, phone FROM users LIMIT 10" --format markdown
```

Instead of returning raw PII, configured fields are masked. Production `--raw` requires interactive confirmation, so an AI agent cannot silently bypass it, and the MCP server does not expose raw output at all.

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

## Pinned launch comment / first reply

Use this as the first comment when the platform supports it:

> A few useful links:
>
> - GitHub: https://github.com/atani/mysh
> - AI-agent safety guide: https://github.com/atani/mysh/blob/main/docs/ai-agent-safety.md
> - MCP server guide: https://github.com/atani/mysh/blob/main/docs/mcp-guide.md
> - Getting started: https://github.com/atani/mysh/blob/main/docs/getting-started.md
>
> I am especially looking for feedback from people who use Claude Code, Cursor, or other agents against database-backed apps.

## Reddit: r/commandline

Title:

> mysh: a MySQL CLI with SSH tunnels and AI-safe output masking

Body:

I built `mysh` for command-line database workflows that now often involve AI tools or scripts capturing output.

It works as a MySQL connection manager, SSH tunnel helper, and query runner. The differentiating bit is that it can automatically mask configured sensitive fields in production/non-TTY contexts before output reaches an AI agent, shell script, CI log, or copied Markdown snippet.

It also imports profiles from DBeaver, Sequel Ace, and MySQL Workbench, supports Markdown/CSV/JSON/PDF exports, and includes a local MCP server for AI clients.

Repo: https://github.com/atani/mysh

I would love feedback on CLI UX, install friction, and whether the masking defaults match real-world workflows.

## Reddit: r/mysql

Title:

> A safer MySQL CLI for teams using AI coding agents and scripts

Body:

I built `mysh`, a MySQL CLI that combines connection profiles, SSH tunnel management, imports from common GUI clients, and output formats like Markdown/CSV/JSON/PDF.

The part that may be interesting here: mysh can automatically mask sensitive query output when running against production-like connections or in non-interactive contexts. The goal is to reduce accidental leakage when query results get copied into AI tools, scripts, CI logs, or support notes.

Production `--raw` requires an interactive TTY confirmation, and the MCP server does not expose raw output at all.

Repo: https://github.com/atani/mysh

Feedback welcome from MySQL users, especially around old MySQL compatibility, SSH tunnel workflows, and safe defaults.

## LinkedIn post

AI coding agents are becoming part of everyday development, but database CLIs were not designed for agent-captured output.

I built `mysh`, a safer MySQL CLI for teams that need production-like database access from Claude Code, Cursor, shell scripts, or local MCP clients without accidentally exposing raw sensitive data.

It includes:

- MySQL connection profiles
- SSH tunnel management
- encrypted local password storage
- imports from DBeaver, Sequel Ace, and MySQL Workbench
- Markdown/CSV/JSON/PDF exports
- automatic masking for sensitive output in production/non-TTY contexts
- a built-in MCP server with the same masking rules

GitHub: https://github.com/atani/mysh

If your team is trying to make AI-assisted database workflows safer, I would love feedback.

## Product Hunt / directory blurb

mysh is a safer MySQL CLI for the AI coding era. It manages connection profiles, SSH tunnels, encrypted credentials, query exports, and automatic sensitive-data masking before output reaches Claude Code, Cursor, shell scripts, or local MCP clients.

## Article outline

Title:

> Stop leaking production data to AI coding agents: why I built mysh

Outline:

1. AI agents make database debugging faster, but captured CLI output is a new leakage path.
2. Traditional MySQL clients assume a human is reading output directly.
3. mysh adds a safer local workflow layer: connection profiles, SSH tunnels, encrypted credentials, and masking.
4. Show the before/after query output example.
5. Explain production `--raw` confirmation and why MCP never exposes raw output.
6. Invite feedback from Go, MySQL, CLI, and AI-agent users.

## Reply templates

### "Is this a DLP/compliance product?"

No. mysh is a safer local workflow layer that reduces accidental CLI-output leakage. It is not a replacement for database permissions, audit logging, access review, or a full compliance/DLP program.

### "Why not just use prompts?"

Prompts help after data reaches an agent. mysh tries to reduce the chance that raw sensitive values reach the agent in the first place by masking output before it leaves the CLI/MCP server.

### "Does this replace mysql/mycli/DBeaver?"

Not necessarily. mysh is strongest when you need reusable connection profiles, SSH tunnels, exports, imports from existing GUI clients, and AI-safe output handling from the command line. It can complement existing tools.

### "Can I use it with PostgreSQL?"

Not yet. mysh is focused on MySQL today. PostgreSQL support would be a good future direction if enough users want the same AI-safe workflow there.

## Suggested posting cadence

Do not post everywhere at once. Use each wave to improve the README and docs before the next one.

1. Day 1: AI-agent communities and personal X/Bluesky post
2. Day 2-3: r/golang or r/commandline, depending on first feedback
3. Day 4-5: r/mysql and Lobsters if the README has been updated from early questions
4. Day 7+: Show HN once install instructions, releases, and docs feel stable
5. After Show HN: publish the article version and link back to the discussion

## Assets to prepare before major posts

- A 30-60 second screen recording showing raw-looking data becoming masked output
- One screenshot of the MCP setup in Claude Code or Cursor
- One screenshot of imported database profiles
- One short GIF under 5 MB for social previews
- A release note that mentions AI-safe masking and MCP support
