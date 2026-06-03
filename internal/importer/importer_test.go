package importer

import (
	"runtime"
	"strings"
	"testing"
)

func TestRegistryGet(t *testing.T) {
	for _, key := range []string{"dbeaver", "sequel-ace", "workbench", "yaml"} {
		p, ok := Get(key)
		if !ok {
			t.Errorf("Get(%q) not registered", key)
			continue
		}
		if p.Name() == "" {
			t.Errorf("Get(%q).Name() is empty", key)
		}
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	if _, ok := Get("does-not-exist"); ok {
		t.Error("Get for unknown key should return false")
	}
}

func TestAvailableSorted(t *testing.T) {
	keys := Available()
	if len(keys) == 0 {
		t.Fatal("Available() returned no providers")
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			t.Errorf("Available() not sorted: %v", keys)
			break
		}
	}
	// All registered keys must resolve via Get.
	for _, k := range keys {
		if _, ok := Get(k); !ok {
			t.Errorf("Available key %q not resolvable via Get", k)
		}
	}
}

func TestProviderNames(t *testing.T) {
	cases := map[string]string{
		"dbeaver":    "DBeaver",
		"sequel-ace": "Sequel Ace",
		"workbench":  "MySQL Workbench",
		"yaml":       "YAML file",
	}
	for key, want := range cases {
		p, ok := Get(key)
		if !ok {
			t.Errorf("provider %q missing", key)
			continue
		}
		if p.Name() != want {
			t.Errorf("%q Name() = %q, want %q", key, p.Name(), want)
		}
	}
}

// TestDiscoverNoConfigFile verifies each provider returns (nil, nil) when its
// source file is absent. We force a temp home with no tool config present.
func TestDiscoverNoConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	for _, key := range []string{"dbeaver", "sequel-ace", "workbench"} {
		p, ok := Get(key)
		if !ok {
			t.Fatalf("provider %q missing", key)
		}
		conns, err := p.Discover()
		if err != nil {
			t.Errorf("%q Discover() with no file: unexpected error %v", key, err)
		}
		if conns != nil {
			t.Errorf("%q Discover() with no file: expected nil, got %v", key, conns)
		}
	}
}

func TestSequelAceFavoritesPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	path := sequelAceFavoritesPath()
	if !strings.Contains(path, "com.sequel-ace.sequel-ace") {
		t.Errorf("path missing container id: %s", path)
	}
	if !strings.HasSuffix(path, "Favorites.plist") {
		t.Errorf("path should end with Favorites.plist: %s", path)
	}
}

func TestDBeaverDataSourcesPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	path := dbeaverDataSourcesPath()
	if !strings.HasSuffix(path, "data-sources.json") {
		t.Errorf("path should end with data-sources.json: %s", path)
	}
}

func TestWorkbenchConnectionsPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	path := workbenchConnectionsPath()
	if !strings.HasSuffix(path, "connections.xml") {
		t.Errorf("path should end with connections.xml: %s", path)
	}
}

func TestPlistToJSONMissingFile(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("plutil only available on macOS")
	}
	_, err := plistToJSON("/nonexistent/path/Favorites.plist")
	if err == nil {
		t.Error("plistToJSON with missing file should return an error")
	}
}
