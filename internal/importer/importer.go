package importer

import (
	"sort"

	"github.com/atani/mysh/internal/config"
)

// ImportedConnection holds connection data extracted from an external tool.
// Password is always empty; the user must re-enter it during import.
type ImportedConnection struct {
	Name   string
	Folder string
	DB     config.DBConfig
	SSH    *config.SSHConfig
}

// Provider reads connections from an external database tool.
type Provider interface {
	Name() string
	Discover() ([]ImportedConnection, error)
}

var registry = map[string]Provider{}

func Register(key string, p Provider) {
	registry[key] = p
}

func Get(key string) (Provider, bool) {
	p, ok := registry[key]
	return p, ok
}

func Available() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
