# Security Policy

mysh is a database access tool, so security and privacy reports are highly appreciated.

## Supported versions

Security fixes are applied to the latest released version and the `main` branch.

## Reporting a vulnerability

Please do **not** open a public GitHub issue for security vulnerabilities.

Instead, report the issue privately using one of these options:

1. GitHub private vulnerability reporting, if enabled for this repository
2. Email the maintainer listed on the GitHub profile

Please include:

- A clear description of the vulnerability
- Steps to reproduce or a minimal proof of concept
- The affected command, platform, and mysh version
- Whether credentials, query results, masking, or tunnel behavior are involved
- Any suggested fix, if you have one

## Scope

Security-sensitive areas include:

- credential encryption and storage
- `MYSH_MASTER_PASSWORD` handling
- OS credential store integration
- SSH tunnel creation and lifecycle
- automatic masking rules
- TTY/non-TTY detection
- production `--raw` safeguards
- config import/export behavior

## Handling sensitive data

Please do not include real database credentials, private keys, production hostnames, customer data, or unmasked personal data in reports. Use synthetic examples whenever possible.
