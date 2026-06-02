# Contributing to mysh

Thanks for your interest in contributing to mysh! This project aims to make MySQL access safer and easier for both humans and AI-assisted workflows.

## Ways to contribute

- Report bugs or confusing behavior
- Suggest usability improvements for database/SSH workflows
- Improve documentation, examples, and installation instructions
- Add tests for masking, importers, output formats, and tunnel behavior
- Help test mysh on macOS, Linux, and Windows

## Development setup

```bash
git clone https://github.com/atani/mysh.git
cd mysh
go test ./...
go run . --help
```

## Pull request checklist

Before opening a PR, please make sure:

- `go test ./...` passes
- User-facing behavior is documented when it changes
- New functionality includes tests where practical
- Security-sensitive changes explain the privacy impact
- The PR description includes what changed and how you verified it

## Security and privacy expectations

mysh handles database credentials and can process sensitive query results. Please be especially careful with changes related to:

- password storage and encryption
- SSH tunnel setup
- masking rules and TTY/non-TTY detection
- production `--raw` behavior
- config import/export

Never include real credentials, production hostnames, private keys, or personal data in issues, tests, fixtures, screenshots, or logs.

## Reporting vulnerabilities

Please do not report security vulnerabilities in public issues. See [SECURITY.md](SECURITY.md) for the supported reporting process.
