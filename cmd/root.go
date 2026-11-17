package cmd

import "fmt"

func Usage() {
	fmt.Print(`mysh - MySQL / PostgreSQL connection manager with SSH tunnel support

Usage:
  mysh <command> [arguments]

Commands:
  add [flags]          Add a new connection (interactive or via flags)
  list, ls             List saved connections
  edit [name]          Edit an existing connection
  connect [name]       Connect to a database
  ping [name]          Test connection
  run [name] <f>       Execute a SQL file
  run [name] -e "SQL"  Execute inline SQL
  tables [name]        Show tables in the database
  mcp                  Start an MCP server over stdio (for AI coding agents)
  tunnel [name]        Start a background SSH tunnel
  tunnel stop [name]   Stop a background tunnel
  tunnel [list]        List active tunnels
  slice <name> <table> --where "cond"  Extract records as INSERT statements (MySQL only)
  queries              List saved SQL queries
  export [name]        Export connections as YAML (passwords excluded)
                       --with-queries also bundles saved .sql queries
  import --from <tool> Import connections from external tools
  remove, rm [name]    Remove a connection
  help                 Show this help

Flags (for add):
  --name <name>        Connection name
  --env <env>          Environment: production, staging, development
  --type <type>        Database type: mysql (default), postgres, supabase
                       (supabase = PostgreSQL with sslmode=require)
  --url <url>          PostgreSQL connection string (postgresql://user:pass@host:port/db)
                       A password in --url stays in shell history; omit it (or keep
                       [YOUR-PASSWORD]) to be prompted. Supabase Direct connection is
                       IPv6-only by default; on IPv4 use the Session pooler.
  --sslmode <mode>     PostgreSQL sslmode: disable, allow, prefer (default),
                       require, verify-ca, verify-full
  --driver <type>      MySQL connection driver: cli (default), native (MySQL 4.x)
                       (PostgreSQL always uses the built-in native driver)
  --mask <cols>        Columns to mask (comma-separated, wildcards OK)
  --db-host <host>     Database host (overrides --url)
  --db-port <port>     Database port (default 3306 / 5432; overrides --url)
  --db-user <user>     Database user (overrides --url)
  --db-name <name>     Database name (overrides --url)
  --ssh-host <host>    SSH host (enables SSH tunnel)
  --ssh-port <port>    SSH port
  --ssh-user <user>    SSH user
  --ssh-key <path>     SSH key path
  --redash-url <url>   Redash server URL (enables Redash mode)
  --redash-key <key>   Redash API key
  --redash-datasource <id>  Redash data source ID

Flags (for run/tables):
  --format <type>      Output format: plain (default), markdown, csv, pdf
  -o, --output <file>  Save output to a file
  --mask               Force output masking (run only)
  --raw                Force raw output (run only)

Flags (for import):
  --from <tool>          Source tool: dbeaver, sequel-ace, workbench, yaml
  --file <path>          File path (required for yaml source)
  --all                  Import all found connections (skip selection)
  --ask-user             Prompt to override DB user for each imported DB connection
  --db-user <user>       Use this DB user for all imported DB connections
  --reuse-password       Ask once and reuse the password for all imported DB connections

Flags (for slice):
  --where "cond"       WHERE condition (required)
  -o, --output <file>  Save output to a file
  --raw                Disable masking (requires confirmation)
`)
}
