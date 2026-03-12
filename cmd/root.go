package cmd

import "fmt"

const version = "0.1.0"

func Usage() {
	fmt.Print(`mysh - MySQL connection manager with SSH tunnel support

Usage:
  mysh <command> [arguments]

Commands:
  add                  Add a new connection interactively
  list, ls             List saved connections
  connect <name>       Connect to a database
  run <name> <f>       Execute a SQL file
  run <name> -e "SQL"  Execute inline SQL
  tunnel <name>        Start a background SSH tunnel
  tunnel stop <name>   Stop a background tunnel
  tunnel [list]        List active tunnels
  queries              List saved SQL queries
  remove, rm <name>    Remove a connection
  help                 Show this help
`)
}
