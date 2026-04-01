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

	envKeys := make([]string, len(config.Environments)+1)
	copy(envKeys, config.Environments)
	envKeys[len(config.Environments)] = ""
	grouped := make(map[string][]config.Connection)
	for _, c := range cfg.Connections {
		key := c.Env
		if _, ok := config.EnvironmentLabels[key]; !ok {
			key = ""
		}
		grouped[key] = append(grouped[key], c)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	first := true
	for _, env := range envKeys {
		conns, ok := grouped[env]
		if !ok {
			continue
		}
		if !first {
			fmt.Fprintln(w)
		}
		first = false
		fmt.Fprintf(w, "[%s]\n", config.EnvironmentLabels[env])
		_, _ = fmt.Fprintln(w, "  NAME\tHOST\tPORT\tUSER\tDATABASE\tSSH")
		for _, c := range conns {
			if c.IsRedash() {
				_, _ = fmt.Fprintf(w, "  %s\t%s\t-\t-\t(Redash #%d)\t-\n",
					c.Name, c.Redash.URL, c.Redash.DataSourceID)
				continue
			}
			ssh := "-"
			if c.SSH != nil {
				ssh = fmt.Sprintf("%s@%s", c.SSH.User, c.SSH.Host)
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%d\t%s\t%s\t%s\n",
				c.Name, c.DB.Host, c.DB.Port, c.DB.User, c.DB.Database, ssh)
		}
	}
	return w.Flush()
}
