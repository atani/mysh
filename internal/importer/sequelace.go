package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atani/mysh/internal/config"
)

// jsonBool handles plist values that can be bool, int, or string "0"/"1".
type jsonBool bool

func (b *jsonBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	switch s {
	case "true", "1":
		*b = true
	case "false", "0", "":
		*b = false
	default:
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as bool", s)
		}
		*b = jsonBool(n != 0)
	}
	return nil
}

func init() {
	Register("sequel-ace", &sequelAceProvider{})
}

type sequelAceProvider struct{}

func (s *sequelAceProvider) Name() string { return "Sequel Ace" }

func (s *sequelAceProvider) Discover() ([]ImportedConnection, error) {
	path := sequelAceFavoritesPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := plistToJSON(path)
	if err != nil {
		return nil, err
	}
	return parseSequelAceFavorites(data)
}

func sequelAceFavoritesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home,
		"Library", "Containers", "com.sequel-ace.sequel-ace",
		"Data", "Library", "Application Support", "Sequel Ace", "Data", "Favorites.plist")
}

func plistToJSON(path string) ([]byte, error) {
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", path)
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("plutil not found: Sequel Ace import requires macOS")
		}
		return nil, fmt.Errorf("converting plist to JSON (%s): %w", path, err)
	}
	return out, nil
}

// sequelAceFile represents the top-level structure of Favorites.plist.
type sequelAceFile struct {
	Favorites []sequelAceFavorite `json:"Favorites"`
}

type sequelAceFavorite struct {
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Database       string `json:"database"`
	Type           int    `json:"type"` // 0=TCP/IP, 1=Socket, 2=SSH
	SSHHost        string `json:"sshHost"`
	SSHPort        int    `json:"sshPort"`
	SSHUser        string `json:"sshUser"`
	SSHKeyLocationEnabled jsonBool `json:"sshKeyLocationEnabled"`
	SSHKeyPath            string `json:"sshKeyLocation"`
}

func parseSequelAceFavorites(data []byte) ([]ImportedConnection, error) {
	var file sequelAceFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing Sequel Ace favorites: %w", err)
	}

	var conns []ImportedConnection
	for _, fav := range file.Favorites {
		if fav.Name == "" && fav.Host == "" {
			continue
		}

		port := fav.Port
		if port == 0 {
			port = 3306
		}

		name := fav.Name
		if name == "" {
			name = fav.Host
		}

		ic := ImportedConnection{
			Name: name,
			DB: config.DBConfig{
				Host:     fav.Host,
				Port:     port,
				User:     fav.User,
				Database: fav.Database,
			},
		}

		if fav.Type == 2 && fav.SSHHost != "" {
			sshPort := fav.SSHPort
			if sshPort == 0 {
				sshPort = 22
			}
			var sshKey string
			if bool(fav.SSHKeyLocationEnabled) {
				sshKey = fav.SSHKeyPath
			}
			ic.SSH = &config.SSHConfig{
				Host: fav.SSHHost,
				Port: sshPort,
				User: fav.SSHUser,
				Key:  sshKey,
			}
		}

		conns = append(conns, ic)
	}
	return conns, nil
}
