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
