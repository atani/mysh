package cmd

import (
	"fmt"
	"os"

	"github.com/atani/mysh/internal/config"
	"gopkg.in/yaml.v3"
)

// ExportedConnection is the YAML format for sharing connections.
// Passwords are always omitted so the file can be safely shared.
type ExportedConnection struct {
	Name string             `yaml:"name"`
	Env  string             `yaml:"env,omitempty"`
	SSH  *config.SSHConfig  `yaml:"ssh,omitempty"`
	DB   ExportedDBConfig   `yaml:"db"`
	Mask *config.MaskConfig `yaml:"mask,omitempty"`
}

type ExportedDBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user"`
	Database string `yaml:"database"`
	Driver   string `yaml:"driver,omitempty"`
}

func RunExport(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Connections) == 0 {
		return fmt.Errorf("no connections configured")
	}

	// If a name is given, export only that connection
	var conns []config.Connection
	if len(args) > 0 {
		name := args[0]
		conn := cfg.Find(name)
		if conn == nil {
			return fmt.Errorf("connection %q not found", name)
		}
		conns = []config.Connection{*conn}
	} else {
		conns = cfg.Connections
	}

	exported := make([]ExportedConnection, len(conns))
	for i, c := range conns {
		exported[i] = ExportedConnection{
			Name: c.Name,
			Env:  c.Env,
			SSH:  c.SSH,
			DB: ExportedDBConfig{
				Host:     c.DB.Host,
				Port:     c.DB.Port,
				User:     c.DB.User,
				Database: c.DB.Database,
				Driver:   c.DB.Driver,
			},
			Mask: c.Mask,
		}
	}

	data, err := yaml.Marshal(exported)
	if err != nil {
		return fmt.Errorf("marshaling connections: %w", err)
	}

	fmt.Fprint(os.Stdout, string(data))
	return nil
}
