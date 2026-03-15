package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type SSHConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port,omitempty"`
	User string `yaml:"user"`
	Key  string `yaml:"key,omitempty"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user"`
	Database string `yaml:"database"`
	Password string `yaml:"password"` // encrypted
}

type MaskConfig struct {
	Columns  []string `yaml:"columns,omitempty"`
	Patterns []string `yaml:"patterns,omitempty"`
}

// Environments defines the recognized environment values in display order.
var Environments = []string{"production", "staging", "development"}

// EnvironmentLabels maps environment values to display labels.
var EnvironmentLabels = map[string]string{
	"production":  "Production",
	"staging":     "Staging",
	"development": "Development",
	"":            "Other",
}

type Connection struct {
	Name string      `yaml:"name"`
	Env  string      `yaml:"env,omitempty"` // production, staging, development
	SSH  *SSHConfig  `yaml:"ssh,omitempty"`
	DB   DBConfig    `yaml:"db"`
	Mask *MaskConfig `yaml:"mask,omitempty"`
}

// HasMaskConfig returns true if the connection has any mask rules configured.
func (c *Connection) HasMaskConfig() bool {
	return c.Mask != nil && (len(c.Mask.Columns) > 0 || len(c.Mask.Patterns) > 0)
}

// ShouldMask returns true if masking should be applied given TTY status.
// Production environments always mask by default (use --raw to override).
// Staging environments mask only when output is piped (non-TTY).
// Development environments never mask.
func (c *Connection) ShouldMask(isTTY bool) bool {
	if !c.HasMaskConfig() {
		return false
	}
	if c.Env == "development" {
		return false
	}
	if c.Env == "production" {
		return true
	}
	return !isTTY
}

type Config struct {
	Connections []Connection `yaml:"connections"`
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mysh")
}

func Path() string {
	return filepath.Join(Dir(), "connections.yaml")
}

func QueriesDir() string {
	return filepath.Join(Dir(), "queries")
}

func TunnelsDir() string {
	return filepath.Join(Dir(), "tunnels")
}

func EnsureDir() error {
	if err := os.MkdirAll(Dir(), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.MkdirAll(QueriesDir(), 0700); err != nil {
		return fmt.Errorf("creating queries directory: %w", err)
	}
	if err := os.MkdirAll(TunnelsDir(), 0700); err != nil {
		return fmt.Errorf("creating tunnels directory: %w", err)
	}
	return nil
}

func Load() (*Config, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(Path(), data, 0600)
}

func (c *Config) Find(name string) *Connection {
	for i := range c.Connections {
		if c.Connections[i].Name == name {
			return &c.Connections[i]
		}
	}
	return nil
}

func (c *Config) Add(conn Connection) error {
	if c.Find(conn.Name) != nil {
		return fmt.Errorf("connection %q already exists", conn.Name)
	}
	c.Connections = append(c.Connections, conn)
	return nil
}

func (c *Config) Remove(name string) error {
	for i := range c.Connections {
		if c.Connections[i].Name == name {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("connection %q not found", name)
}
