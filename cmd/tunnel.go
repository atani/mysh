package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"


	"github.com/atani/mysh/internal/tunnel"
)

func RunTunnel(args []string) error {
	if len(args) == 0 {
		return tunnelList()
	}

	switch args[0] {
	case "stop":
		var name string
		if len(args) >= 2 {
			name = args[1]
		}
		return tunnelStop(name)
	case "list", "ls":
		return tunnelList()
	default:
		return tunnelOpen(args[0])
	}
}

func tunnelOpen(name string) error {
	_, conn, err := findConnection(name)
	if err != nil {
		return err
	}

	if conn.SSH == nil {
		return fmt.Errorf("connection %q has no SSH config", name)
	}

	dbPort := conn.DB.Port
	if dbPort == 0 {
		dbPort = 3306
	}

	info, err := tunnel.OpenBackground(name, conn.SSH, conn.DB.Host, dbPort)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Tunnel %q ready (PID %d, localhost:%d)\n", name, info.PID, info.LocalPort)
	return nil
}

func tunnelStop(name string) error {
	if name == "" {
		// Auto-resolve if only one connection
		_, conn, err := findConnection("")
		if err != nil {
			return err
		}
		name = conn.Name
	}
	if err := tunnel.StopBackground(name); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Tunnel %q stopped.\n", name)
	return nil
}

func tunnelList() error {
	tunnels, err := tunnel.ListRunning()
	if err != nil {
		return err
	}

	if len(tunnels) == 0 {
		fmt.Fprintln(os.Stderr, "No active tunnels.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tPID\tLOCAL PORT\tREMOTE")
	for _, t := range tunnels {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%s:%d\n",
			t.Name, t.PID, t.LocalPort, t.RemoteHost, t.RemotePort)
	}
	return w.Flush()
}
