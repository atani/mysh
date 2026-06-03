# Global launch checklist

This checklist is for maintainers preparing mysh for English-speaking developer communities.

## Repository metadata

Recommended GitHub settings:

- Description: `A safer MySQL CLI for AI coding agents: connection profiles, SSH tunnels, and automatic sensitive-data masking.`
- Homepage: `https://github.com/atani/mysh`
- Enable Discussions
- Topics:
  - `ai-agents`
  - `claude-code`
  - `cursor`
  - `privacy`
  - `pii`
  - `developer-tools`
  - `database-tools`
  - `mysql-cli`
  - `mysql`
  - `ssh-tunnel`
  - `golang`
  - `cli`
  - `data-masking`
  - `security`

## Launch order

Use small, relevant communities first. Improve wording based on feedback before posting to larger communities.

1. AI coding communities: Claude Code / Cursor / agent users
2. `r/golang`
3. `r/commandline`
4. `r/mysql`
5. Lobsters
6. Show HN
7. Product Hunt or similar broader launch surfaces

## Before posting

- README explains the AI-agent safety angle in the first screen
- Install commands are visible and short
- Masking example is easy to understand without reading code
- Comparison table explains why mysh is different from `mysql`, `mycli`, and GUI clients
- Issues, PR template, contributing guide, security policy, and code of conduct exist
- Latest release assets are available for macOS/Linux/Windows
- `go test ./...` and CI are green

## Post timing

- Avoid posting everywhere on the same day
- Post when you can reply for 2-4 hours
- Treat early comments as product research, not just promotion
- Update README wording if the same question appears repeatedly

## Response principles

- Lead with the privacy/AI workflow problem
- Be explicit that mysh reduces accidental CLI-output leakage; do not oversell it as a full compliance solution
- Ask for feedback from people who use AI agents with database-backed apps
- Thank users who report confusing install or masking behavior

## Submission targets to consider

- Hacker News: `Show HN: mysh – a safer MySQL CLI for AI coding agents`
- Reddit: `r/golang`, `r/commandline`, `r/mysql`, AI coding subreddits
- Lobsters: CLI/database/privacy angle
- Dev.to / Hashnode: publish the article draft from `docs/launch-kit.md`
- Awesome lists: Go CLI/database/tooling lists if they accept self-submissions

## Metrics to watch

- GitHub stars and clones after each post
- README bounce questions: what people ask before starring
- Install failures by platform
- Feature requests around masking rules and AI-agent workflows
- Which community produces real users vs. drive-by stars

## Suggested seeded issues

Create these after the launch-readiness PR is merged, so visitors see clear ways to contribute:

- `feat: PostgreSQL support for AI-safe output masking` (`enhancement`, `help wanted`)
- `feat: masking preset library for common PII fields` (`enhancement`, `good first issue`)
- `docs: add Cursor MCP setup screenshots` (`documentation`, `good first issue`)
- `docs: add Claude Desktop MCP setup screenshots` (`documentation`, `good first issue`)
- `feat: configurable masking preview command` (`enhancement`)
- `docs: add real-world SSH tunnel examples` (`documentation`)

## Discussion categories / starter threads

If GitHub Discussions is enabled, seed a few lightweight threads:

- `Show and tell: how are you using mysh with AI coding agents?`
- `Ideas: which masking presets should ship by default?`
- `Q&A: MCP setup with Claude Code, Cursor, and Claude Desktop`
- `Roadmap: PostgreSQL or other database support`

## Maintainer launch checklist

- [ ] Update GitHub description to the recommended short AI-agent-safe positioning
- [ ] Enable Discussions
- [ ] Add or confirm topics include `ai-agents`, `claude-code`, `cursor`, `privacy`, `pii`, `mysql-cli`, and `mcp`
- [ ] Confirm the latest release has macOS/Linux/Windows assets
- [ ] Pin or highlight the demo GIF/video in social posts
- [ ] Create 3-6 seeded issues with `good first issue` / `help wanted`
- [ ] Watch comments for repeated confusion and update README within 24 hours
- [ ] Thank early users and ask for concrete workflow feedback rather than only stars

## Star-conversion reminders

- Put the risk in the first sentence: raw production data can end up in agent contexts.
- Show the masking example before deep feature lists.
- Ask for feedback first; ask for stars only after the value is clear.
- Reuse the same language across README, posts, and release notes so people recognize the project.
