package main

import (
	"fmt"
	"os"

	"github.com/atani/mysh/cmd"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		cmd.Usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "add":
		err = cmd.RunAdd(os.Args[2:])
	case "list", "ls":
		err = cmd.RunList(os.Args[2:])
	case "edit":
		err = cmd.RunEdit(os.Args[2:])
	case "connect":
		err = cmd.RunConnect(os.Args[2:])
	case "run":
		err = cmd.RunQuery(os.Args[2:])
	case "slice":
		err = cmd.RunSlice(os.Args[2:])
	case "ping":
		err = cmd.RunPing(os.Args[2:])
	case "tables":
		err = cmd.RunTables(os.Args[2:])
	case "tunnel":
		err = cmd.RunTunnel(os.Args[2:])
	case "queries":
		err = cmd.RunQueries(os.Args[2:])
	case "export":
		err = cmd.RunExport(os.Args[2:])
	case "import":
		err = cmd.RunImport(os.Args[2:])
	case "remove", "rm":
		err = cmd.RunRemove(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("mysh " + version)
	case "help", "-h", "--help":
		cmd.Usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		cmd.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
