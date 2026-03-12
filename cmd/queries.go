package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/atani/mysh/internal/config"
)

func RunQueries(_ []string) error {
	dir := config.QueriesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "No queries directory. Save .sql files in %s\n", dir)
			return nil
		}
		return err
	}

	var sqlFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, e)
		}
	}

	if len(sqlFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No SQL files found. Save .sql files in %s\n", dir)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "FILE\tSIZE\tPATH")
	for _, e := range sqlFiles {
		info, _ := e.Info()
		size := "-"
		if info != nil {
			size = fmt.Sprintf("%d B", info.Size())
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			e.Name(), size, filepath.Join(dir, e.Name()))
	}
	return w.Flush()
}
