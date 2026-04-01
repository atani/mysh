package importer

import (
	"fmt"
	"os"

	"github.com/atani/mysh/internal/config"
	"gopkg.in/yaml.v3"
)

func init() {
	Register("yaml", &yamlProvider{})
}

type yamlProvider struct{}

func (y *yamlProvider) Name() string { return "YAML file" }

type yamlRedash struct {
	URL          string `yaml:"url"`
	DataSourceID int    `yaml:"data_source_id"`
}

type yamlConnection struct {
	Name   string             `yaml:"name"`
	Env    string             `yaml:"env,omitempty"`
	SSH    *config.SSHConfig  `yaml:"ssh,omitempty"`
	DB     *config.DBConfig   `yaml:"db,omitempty"`
	Redash *yamlRedash        `yaml:"redash,omitempty"`
	Mask   *config.MaskConfig `yaml:"mask,omitempty"`
}

func (y *yamlProvider) Discover() ([]ImportedConnection, error) {
	return nil, fmt.Errorf("yaml provider requires a file path; use --file <path>")
}

// DiscoverFromFile reads connections from a YAML file exported by `mysh export`.
func (y *yamlProvider) DiscoverFromFile(path string) ([]ImportedConnection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var conns []yamlConnection
	if err := yaml.Unmarshal(data, &conns); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if len(conns) == 0 {
		return nil, nil
	}

	result := make([]ImportedConnection, len(conns))
	for i, c := range conns {
		ic := ImportedConnection{
			Name: c.Name,
			SSH:  c.SSH,
			Mask: c.Mask,
		}
		if c.Env != "" {
			ic.Env = c.Env
		}
		if c.Redash != nil && c.Redash.URL != "" {
			ic.Redash = &config.RedashConfig{
				URL:          c.Redash.URL,
				DataSourceID: c.Redash.DataSourceID,
			}
		}
		if c.DB != nil {
			ic.DB = *c.DB
		}
		result[i] = ic
	}

	return result, nil
}
