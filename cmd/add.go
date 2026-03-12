package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new MySQL connection interactively",
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var (
		name     string
		useSSH   bool
		sshHost  string
		sshPort  string
		sshUser  string
		sshKey   string
		dbHost   string
		dbPort   string
		dbUser   string
		dbPass   string
		dbName   string
	)

	// Connection name
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Connection name").
				Description("A unique name for this connection (e.g., production, staging)").
				Value(&name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					if cfg.Find(s) != nil {
						return fmt.Errorf("connection %q already exists", s)
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return err
	}

	// SSH settings
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use SSH tunnel?").
				Value(&useSSH),
		),
	).Run()
	if err != nil {
		return err
	}

	if useSSH {
		sshPort = "22"
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("SSH host").
					Value(&sshHost).
					Validate(notEmpty("SSH host")),
				huh.NewInput().
					Title("SSH port").
					Value(&sshPort),
				huh.NewInput().
					Title("SSH user").
					Value(&sshUser).
					Validate(notEmpty("SSH user")),
				huh.NewInput().
					Title("SSH key path (leave empty for default)").
					Description("e.g., ~/.ssh/id_ed25519").
					Value(&sshKey),
			),
		).Run()
		if err != nil {
			return err
		}
	}

	// DB settings
	dbHost = "127.0.0.1"
	dbPort = "3306"
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("MySQL host").
				Description("Use 127.0.0.1 if connecting via SSH tunnel").
				Value(&dbHost).
				Validate(notEmpty("MySQL host")),
			huh.NewInput().
				Title("MySQL port").
				Value(&dbPort),
			huh.NewInput().
				Title("MySQL user").
				Value(&dbUser).
				Validate(notEmpty("MySQL user")),
			huh.NewInput().
				Title("MySQL password").
				EchoMode(huh.EchoModePassword).
				Value(&dbPass),
			huh.NewInput().
				Title("Database name").
				Value(&dbName).
				Validate(notEmpty("Database name")),
		),
	).Run()
	if err != nil {
		return err
	}

	// Encrypt password
	var encryptedPassword string
	if dbPass != "" {
		masterPass, err := getMasterPassword()
		if err != nil {
			return err
		}

		enc, err := crypto.Encrypt([]byte(dbPass), masterPass)
		if err != nil {
			return fmt.Errorf("encrypting password: %w", err)
		}

		encryptedPassword, err = crypto.MarshalEncrypted(enc)
		if err != nil {
			return fmt.Errorf("encoding encrypted password: %w", err)
		}
	}

	conn := config.Connection{
		Name: name,
		DB: config.DBConfig{
			Host:     dbHost,
			Port:     mustAtoi(dbPort, 3306),
			User:     dbUser,
			Database: dbName,
			Password: encryptedPassword,
		},
	}

	if useSSH {
		conn.SSH = &config.SSHConfig{
			Host: sshHost,
			Port: mustAtoi(sshPort, 22),
			User: sshUser,
			Key:  sshKey,
		}
	}

	if err := cfg.Add(conn); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Connection %q added.\n", name)
	return nil
}

func getMasterPassword() ([]byte, error) {
	if !crypto.MasterPasswordInitialized() {
		fmt.Fprintln(os.Stderr, "Setting up master password for the first time.")
		fmt.Fprintln(os.Stderr, "This password protects your stored database credentials.")
		pass, err := crypto.ReadMasterPassword()
		if err != nil {
			return nil, err
		}
		fmt.Fprint(os.Stderr, "Confirm master password: ")
		confirm, err := crypto.ReadMasterPassword()
		if err != nil {
			return nil, err
		}
		if string(pass) != string(confirm) {
			return nil, fmt.Errorf("passwords do not match")
		}
		if err := config.EnsureDir(); err != nil {
			return nil, err
		}
		if err := crypto.InitMasterPassword(pass); err != nil {
			return nil, err
		}
		return pass, nil
	}

	pass, err := crypto.ReadMasterPassword()
	if err != nil {
		return nil, err
	}
	if err := crypto.VerifyMasterPassword(pass); err != nil {
		return nil, err
	}
	return pass, nil
}

func notEmpty(field string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

func mustAtoi(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
