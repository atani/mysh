package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/atani/mysh/internal/config"
)

func RunList(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Connections) == 0 {
		fmt.Fprintln(os.Stderr, "No connections configured. Run `mysh add` to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tHOST\tPORT\tUSER\tDATABASE\tSSH")
	for _, c := range cfg.Connections {
		ssh := "-"
		if c.SSH != nil {
			ssh = fmt.Sprintf("%s@%s", c.SSH.User, c.SSH.Host)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			c.Name, c.DB.Host, c.DB.Port, c.DB.User, c.DB.Database, ssh)
	}
	return w.Flush()
}
