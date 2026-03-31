package importer

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/atani/mysh/internal/config"
)

func init() {
	Register("workbench", &workbenchProvider{})
}

type workbenchProvider struct{}

func (w *workbenchProvider) Name() string { return "MySQL Workbench" }

func (w *workbenchProvider) Discover() ([]ImportedConnection, error) {
	path := workbenchConnectionsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading MySQL Workbench config: %w", err)
	}
	return parseWorkbenchConnections(data)
}

func workbenchConnectionsPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(home, ".mysql", "workbench", "connections.xml")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "MySQL", "Workbench", "connections.xml")
	default: // darwin
		return filepath.Join(home, "Library", "Application Support", "MySQL", "Workbench", "connections.xml")
	}
}

// MySQL Workbench uses GRT (Generic RunTime) XML format.
// Connections are stored as a list of db.mgmt.Connection objects.
type wbData struct {
	XMLName xml.Name `xml:"data"`
	Values  []wbValue `xml:"value"`
}

type wbValue struct {
	Type          string    `xml:"type,attr"`
	StructType    string    `xml:"struct-name,attr"`
	Key           string    `xml:"key,attr"`
	ContentType   string    `xml:"content-type,attr"`
	ContentStruct string    `xml:"content-struct-name,attr"`
	Content       string    `xml:",chardata"`
	Values        []wbValue `xml:"value"`
}

func parseWorkbenchConnections(data []byte) ([]ImportedConnection, error) {
	var doc wbData
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing MySQL Workbench connections.xml: %w", err)
	}

	// Find the top-level list of connections
	var connValues []wbValue
	for _, v := range doc.Values {
		if v.ContentStruct == "db.mgmt.Connection" {
			connValues = v.Values
			break
		}
	}

	var conns []ImportedConnection
	for _, cv := range connValues {
		if cv.StructType != "db.mgmt.Connection" {
			continue
		}

		params := wbExtractMap(cv, "parameterValues")
		name := wbExtractString(cv, "name")
		connMethod := wbExtractString(cv, "driver")

		if name == "" {
			name = params["hostName"]
		}

		host := params["hostName"]
		if host == "" {
			host = "127.0.0.1"
		}

		port := wbParsePort(params["port"], 3306)

		ic := ImportedConnection{
			Name: name,
			DB: config.DBConfig{
				Host:     host,
				Port:     port,
				User:     params["userName"],
				Database: params["schema"],
			},
		}

		// SSH tunnel: connection method contains "ssh" or sshHost is set
		sshHost := params["sshHost"]
		if sshHost != "" || containsSSH(connMethod) {
			if sshHost == "" {
				sshHost = host
			}
			sshPort := wbParsePort(params["sshPort"], 22)
			ic.SSH = &config.SSHConfig{
				Host: sshHost,
				Port: sshPort,
				User: params["sshUserName"],
				Key:  params["sshKeyFile"],
			}
		}

		conns = append(conns, ic)
	}

	sort.Slice(conns, func(i, j int) bool {
		return conns[i].Name < conns[j].Name
	})
	return conns, nil
}

// wbExtractString finds a direct child <value key="key"> and returns its text.
func wbExtractString(parent wbValue, key string) string {
	for _, v := range parent.Values {
		if v.Key == key {
			return v.Content
		}
	}
	return ""
}

// wbExtractMap finds a child <value key="key" type="dict"> and returns its entries.
func wbExtractMap(parent wbValue, key string) map[string]string {
	m := make(map[string]string)
	for _, v := range parent.Values {
		if v.Key == key && v.Type == "dict" {
			for _, entry := range v.Values {
				m[entry.Key] = entry.Content
			}
			return m
		}
	}
	return m
}

func wbParsePort(s string, defaultPort int) int {
	if s == "" {
		return defaultPort
	}
	n, err := strconv.Atoi(s)
	if err != nil || n == 0 {
		return defaultPort
	}
	return n
}

func containsSSH(s string) bool {
	return strings.Contains(strings.ToLower(s), "ssh")
}
