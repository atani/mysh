# MCP Server Guide

mysh includes a built-in [Model Context Protocol](https://modelcontextprotocol.io) (MCP)
server so AI coding agents — Claude Code, Cursor, and any other MCP client — can query
your databases as a native tool instead of shelling out to the `mysh` CLI.

The server runs locally over stdio. There is no hosting, API key, or external service to
set up: it is the same `mysh` binary you already have, started with `mysh mcp`.

## Why use the MCP server?

- **AI-safe by design.** Query output is masked using the same rules as non-TTY CLI
  execution. Production and staging connections with mask rules are masked, and there is
  **no way to request raw output over MCP** — sensitive values are masked before they ever
  reach the agent.
- **Native discovery.** The agent sees `mysh_query`, `mysh_tables`, etc. as first-class
  tools and can call them directly, rather than guessing at CLI flags.
- **Zero extra infrastructure.** The server is stdio-based and stateless; nothing is
  exposed on a network port.

## Prerequisites: configure connections first

The MCP server does not introduce a new place to define connections — it reads the same
`~/.config/mysh/connections.yaml` the CLI uses. So you configure connections **once with the
CLI**, then the agent uses them through MCP. There are no `add`/`edit`/`remove` tools over
MCP by design (no mutation from the agent).

Set up connections before registering the server, using any of:

```bash
mysh add                                   # interactive: MySQL host/SSH/Redash
mysh import --from dbeaver                  # migrate from DBeaver/Sequel Ace/Workbench
mysh import --from yaml --file team-db.yaml # import a shared, password-free profile
# Redash:
mysh add --name prod --redash-url https://redash.example.com \
  --redash-key YOUR_API_KEY --redash-datasource 1
```

Verify with `mysh list` and `mysh ping`. Anything that works for `mysh run` will work for
`mysh_query` over MCP.

## Setup

### Claude Code

```bash
claude mcp add mysh -- mysh mcp
```

### Cursor / Claude Desktop / generic MCP clients

Add an entry to the client's MCP server configuration (for Claude Desktop this is
`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "mysh": {
      "command": "mysh",
      "args": ["mcp"]
    }
  }
}
```

If `mysh` is not on the client's `PATH`, use an absolute path for `command` (e.g.
`/opt/homebrew/bin/mysh`).

### Master password for unattended decryption

Connections with encrypted passwords (and Redash connections with an encrypted API key)
need the master password to decrypt them. Because the MCP transport is non-interactive,
the server **never prompts for the master password** — if it is unavailable, `mysh_query`
and `mysh_ping` return a clear error telling you to supply it. Provide it one of two ways:

- **OS credential store** (recommended on macOS/Windows): run any `mysh` command
  interactively once. The master password is then saved to the Keychain / Credential
  Manager and reused automatically by the MCP server.
- **Environment variable**: set `MYSH_MASTER_PASSWORD` in the MCP server's environment
  (required on platforms without a supported credential store):

```json
{
  "mcpServers": {
    "mysh": {
      "command": "mysh",
      "args": ["mcp"],
      "env": { "MYSH_MASTER_PASSWORD": "..." }
    }
  }
}
```

## Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `mysh_list_connections` | List configured connections (no secrets). | — |
| `mysh_query` | Run SQL against a connection and return masked results. | `sql` (required), `connection`, `format` |
| `mysh_tables` | List tables in a connection's database. | `connection`, `format` |
| `mysh_ping` | Test connectivity and report success. | `connection` |

Notes:

- `connection` may be omitted when exactly one connection is configured.
- `format` accepts `markdown` (default), `json`, `csv`, or `plain`. PDF is not available
  over MCP.
- `mysh_query` and `mysh_tables` support direct MySQL connections (CLI and native drivers)
  and SSH tunnels. `mysh_query` also supports Redash connections; `mysh_tables` does not.

## Masking behavior over MCP

Masking follows the connection's `env` and mask rules, evaluated as if output were piped
(non-TTY):

| env | Masked over MCP? |
|-----|------------------|
| production | Yes (when mask rules are configured) |
| staging | Yes (when mask rules are configured) |
| development | No |

Because the MCP transport is non-interactive, the production `--raw` confirmation flow is
never reachable — agents cannot bypass masking. If you need an unmasked extract, use the
CLI directly with an interactive terminal.

## Protocol details

- Transport: stdio, newline-delimited JSON-RPC 2.0.
- Implemented methods: `initialize`, `ping`, `tools/list`, `tools/call`.
- The server advertises the `tools` capability and echoes the client's requested
  `protocolVersion` during the handshake.

## Troubleshooting

- **"connection not found"**: run `mysh list` to confirm the connection name, or omit
  `connection` if you only have one.
- **Decryption errors**: ensure the master password is available (OS credential store or
  `MYSH_MASTER_PASSWORD`).
- **No output / empty result**: non-SELECT statements return `Query OK`.
