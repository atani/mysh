package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mysh",
	Short: "MySQL connection manager with SSH tunnel support",
	Long:  "mysh manages MySQL connections, handles SSH tunnels automatically, and stores saved queries.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
