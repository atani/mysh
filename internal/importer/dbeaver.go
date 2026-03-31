package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atani/mysh/internal/config"
)

func init() {
	Register("dbeaver", &dbeaverProvider{})
}

type dbeaverProvider struct{}

func (d *dbeaverProvider) Name() string { return "DBeaver" }

func (d *dbeaverProvider) Discover() ([]ImportedConnection, error) {
	path := dbeaverDataSourcesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading DBeaver config: %w", err)
	}
	return parseDBeaverDataSources(data)
}

func dbeaverDataSourcesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "DBeaverData", "workspace6", "General", ".dbeaver", "data-sources.json")
}

type dbeaverFile struct {
	Connections map[string]dbeaverConn `json:"connections"`
}

type dbeaverConn struct {
	Provider string       `json:"provider"`
	Driver   string       `json:"driver"`
	Name     string       `json:"name"`
	Folder   string       `json:"folder"`
	Config   dbeaverCfg   `json:"configuration"`
}

type dbeaverCfg struct {
	Host     string                       `json:"host"`
	Port     json.RawMessage              `json:"port"`
	Database string                       `json:"database"`
	Handlers map[string]dbeaverHandler    `json:"handlers"`
	AuthProps map[string]string           `json:"auth-properties"`
}

type dbeaverHandler struct {
	Type       string                 `json:"type"`
	Enabled    bool                   `json:"enabled"`
	Properties map[string]interface{} `json:"properties"`
}

func parseDBeaverDataSources(data []byte) ([]ImportedConnection, error) {
	var file dbeaverFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing DBeaver data-sources.json: %w", err)
	}

	var conns []ImportedConnection
	for _, dc := range file.Connections {
		if dc.Provider != "mysql" && !strings.HasPrefix(dc.Driver, "mysql") {
			continue
		}

		port := parsePort(dc.Config.Port, 3306)
		user := dc.Config.AuthProps["user"]

		ic := ImportedConnection{
			Name:   dc.Name,
			Folder: dc.Folder,
			DB: config.DBConfig{
				Host:     dc.Config.Host,
				Port:     port,
				User:     user,
				Database: dc.Config.Database,
			},
		}

		if ssh, ok := dc.Config.Handlers["ssh_tunnel"]; ok && ssh.Enabled {
			ic.SSH = parseDBeaverSSH(ssh.Properties)
		}

		conns = append(conns, ic)
	}
	sort.Slice(conns, func(i, j int) bool {
		return conns[i].Name < conns[j].Name
	})
	return conns, nil
}

func parseDBeaverSSH(props map[string]interface{}) *config.SSHConfig {
	if props == nil {
		return nil
	}
	sc := &config.SSHConfig{
		Host: stringProp(props, "host"),
	}
	if sc.Host == "" {
		return nil
	}
	sc.Port = intProp(props, "port")
	if sc.Port == 0 {
		sc.Port = 22
	}
	sc.User = stringProp(props, "user")
	sc.Key = stringProp(props, "keyPath")
	return sc
}

func parsePort(raw json.RawMessage, defaultPort int) int {
	if raw == nil {
		return defaultPort
	}
	// Try as number first
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	// Try as string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return defaultPort
}

func stringProp(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func intProp(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}
