package cmd

import (
	"fmt"
	"os"
	"os/exec"


)

func RunConnect(args []string) error {
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

	client := "mycli"
	if _, err := exec.LookPath("mycli"); err != nil {
		client = "mysql"
		if _, err := exec.LookPath("mysql"); err != nil {
			return fmt.Errorf("neither mycli nor mysql found in PATH")
		}
	}

	c := exec.Command(client, rc.mysqlArgs()...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
