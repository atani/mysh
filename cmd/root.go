package cmd

import "fmt"

func Usage() {
	fmt.Print(`mysh - MySQL connection manager with SSH tunnel support

Usage:
  mysh <command> [arguments]

Commands:
  add                  Add a new connection interactively
  list, ls             List saved connections
  connect <name>       Connect to a database
  ping <name>          Test connection
  run <name> <f>       Execute a SQL file
  run <name> -e "SQL"  Execute inline SQL
  tables <name>        Show tables in the database
  tunnel <name>        Start a background SSH tunnel
  tunnel stop <name>   Stop a background tunnel
  tunnel [list]        List active tunnels
  queries              List saved SQL queries
  remove, rm <name>    Remove a connection
  help                 Show this help

Flags (for run/tables):
  --format <type>      Output format: plain (default), markdown, csv, pdf
  -o, --output <file>  Save output to a file
  --mask               Force output masking (run only)
  --raw                Force raw output (run only)
`)
}
