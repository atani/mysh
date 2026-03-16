package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/atani/mysh/internal/db"
)

func RunPing(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	_, conn, err := findConnection(name)
	if err != nil {
		return err
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	start := time.Now()

	if rc.isNative() {
		dbConn, err := rc.openDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Connection %q: FAILED (%v)\n", conn.Name, err)
			return err
		}
		defer func() { _ = dbConn.Close() }()

		if err := db.Ping(dbConn); err != nil {
			fmt.Fprintf(os.Stderr, "Connection %q: FAILED (%v)\n", conn.Name, err)
			return err
		}
	} else {
		mysqlArgs := rc.mysqlArgs()
		mysqlArgs = append(mysqlArgs, "-e", "SELECT 1")

		c := exec.Command("mysql", mysqlArgs...)
		c.Env = rc.mysqlEnv()
		c.Stdout = nil
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Connection %q: FAILED (%v)\n", conn.Name, err)
			return err
		}
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "Connection %q: OK (%s)\n", conn.Name, elapsed.Round(time.Millisecond))
	return nil
}
