package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/atani/mysh/internal/config"
)

var envOrder = []string{"production", "staging", "development", ""}

var envLabels = map[string]string{
	"production":  "Production",
	"staging":     "Staging",
	"development": "Development",
	"":            "Other",
}

func RunList(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Connections) == 0 {
		fmt.Fprintln(os.Stderr, "No connections configured. Run `mysh add` to add one.")
		return nil
	}

	grouped := make(map[string][]config.Connection)
	for _, c := range cfg.Connections {
		key := c.Env
		if key != "production" && key != "staging" && key != "development" {
			key = ""
		}
		grouped[key] = append(grouped[key], c)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	first := true
	for _, env := range envOrder {
		conns, ok := grouped[env]
		if !ok {
			continue
		}
		if !first {
			fmt.Fprintln(w)
		}
		first = false
		fmt.Fprintf(w, "[%s]\n", envLabels[env])
		_, _ = fmt.Fprintln(w, "  NAME\tHOST\tDATABASE\tSSH")
		for _, c := range conns {
			ssh := "-"
			if c.SSH != nil {
				ssh = fmt.Sprintf("%s@%s", c.SSH.User, c.SSH.Host)
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				c.Name, c.DB.Host, c.DB.Database, ssh)
		}
	}
	return w.Flush()
}
