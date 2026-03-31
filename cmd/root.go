package cmd

import "fmt"

func Usage() {
	fmt.Print(`mysh - MySQL connection manager with SSH tunnel support

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
  tunnel [name]        Start a background SSH tunnel
  tunnel stop [name]   Stop a background tunnel
  tunnel [list]        List active tunnels
  slice <name> <table> --where "cond"  Extract records as INSERT statements
  queries              List saved SQL queries
  import --from <tool> Import connections from external tools
  remove, rm [name]    Remove a connection
  help                 Show this help

Flags (for add):
  --name <name>        Connection name
  --env <env>          Environment: production, staging, development
  --driver <type>      Connection driver: cli (default), native (MySQL 4.x)
  --mask <cols>        Columns to mask (comma-separated, wildcards OK)
  --db-host <host>     MySQL host
  --db-port <port>     MySQL port
  --db-user <user>     MySQL user
  --db-name <name>     Database name
  --ssh-host <host>    SSH host (enables SSH tunnel)
  --ssh-port <port>    SSH port
  --ssh-user <user>    SSH user
  --ssh-key <path>     SSH key path

Flags (for run/tables):
  --format <type>      Output format: plain (default), markdown, csv, pdf
  -o, --output <file>  Save output to a file
  --mask               Force output masking (run only)
  --raw                Force raw output (run only)

Flags (for import):
  --from <tool>          Source tool: dbeaver, sequel-ace
  --all                  Import all found connections (skip selection)

Flags (for slice):
  --where "cond"       WHERE condition (required)
  -o, --output <file>  Save output to a file
  --raw                Disable masking (requires confirmation)
`)
}
