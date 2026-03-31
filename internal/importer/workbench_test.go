package importer

import (
	"strings"
	"testing"
)

// Sample MySQL Workbench GRT XML for testing.
// This mimics the actual connections.xml structure.
const workbenchXML = `<?xml version="1.0" encoding="UTF-8"?>
<data>
  <value type="list" key="connections" content-type="object" content-struct-name="db.mgmt.Connection">
    <value type="object" struct-name="db.mgmt.Connection">
      <value type="string" key="name">production-db</value>
      <value type="string" key="driver">com.mysql.cj.jdbc.Driver</value>
      <value type="dict" key="parameterValues">
        <value type="string" key="hostName">10.0.0.1</value>
        <value type="string" key="port">3306</value>
        <value type="string" key="userName">admin</value>
        <value type="string" key="schema">myapp</value>
        <value type="string" key="sshHost">bastion.example.com</value>
        <value type="string" key="sshPort">22</value>
        <value type="string" key="sshUserName">deploy</value>
        <value type="string" key="sshKeyFile">/home/user/.ssh/id_rsa</value>
      </value>
    </value>
    <value type="object" struct-name="db.mgmt.Connection">
      <value type="string" key="name">local-dev</value>
      <value type="string" key="driver">com.mysql.cj.jdbc.Driver</value>
      <value type="dict" key="parameterValues">
        <value type="string" key="hostName">127.0.0.1</value>
        <value type="string" key="port">3306</value>
        <value type="string" key="userName">root</value>
        <value type="string" key="schema">devdb</value>
      </value>
    </value>
  </value>
</data>`

func TestParseWorkbenchConnections(t *testing.T) {
	conns, err := parseWorkbenchConnections([]byte(workbenchXML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(conns))
	}

	// Sorted by name: local-dev, production-db
	local := conns[0]
	prod := conns[1]

	if local.Name != "local-dev" {
		t.Errorf("expected name 'local-dev', got %q", local.Name)
	}
	if local.DB.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got %q", local.DB.Host)
	}
	if local.DB.Port != 3306 {
		t.Errorf("expected port 3306, got %d", local.DB.Port)
	}
	if local.DB.User != "root" {
		t.Errorf("expected user 'root', got %q", local.DB.User)
	}
	if local.DB.Database != "devdb" {
		t.Errorf("expected database 'devdb', got %q", local.DB.Database)
	}
	if local.SSH != nil {
		t.Errorf("expected no SSH, got %+v", local.SSH)
	}

	if prod.Name != "production-db" {
		t.Errorf("expected name 'production-db', got %q", prod.Name)
	}
	if prod.DB.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", prod.DB.Host)
	}
	if prod.DB.User != "admin" {
		t.Errorf("expected user 'admin', got %q", prod.DB.User)
	}
	if prod.DB.Database != "myapp" {
		t.Errorf("expected database 'myapp', got %q", prod.DB.Database)
	}
	if prod.SSH == nil {
		t.Fatal("expected SSH config, got nil")
	}
	if prod.SSH.Host != "bastion.example.com" {
		t.Errorf("expected SSH host 'bastion.example.com', got %q", prod.SSH.Host)
	}
	if prod.SSH.Port != 22 {
		t.Errorf("expected SSH port 22, got %d", prod.SSH.Port)
	}
	if prod.SSH.User != "deploy" {
		t.Errorf("expected SSH user 'deploy', got %q", prod.SSH.User)
	}
	if prod.SSH.Key != "/home/user/.ssh/id_rsa" {
		t.Errorf("expected SSH key path, got %q", prod.SSH.Key)
	}
}

func TestParseWorkbenchConnections_InvalidXML(t *testing.T) {
	_, err := parseWorkbenchConnections([]byte(`<invalid`))
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}

func TestParseWorkbenchConnections_Empty(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<data>
  <value type="list" key="connections" content-type="object" content-struct-name="db.mgmt.Connection">
  </value>
</data>`)
	conns, err := parseWorkbenchConnections(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestParseWorkbenchConnections_DefaultPort(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<data>
  <value type="list" key="connections" content-type="object" content-struct-name="db.mgmt.Connection">
    <value type="object" struct-name="db.mgmt.Connection">
      <value type="string" key="name">no-port</value>
      <value type="string" key="driver">com.mysql.cj.jdbc.Driver</value>
      <value type="dict" key="parameterValues">
        <value type="string" key="hostName">localhost</value>
        <value type="string" key="userName">root</value>
        <value type="string" key="schema">test</value>
      </value>
    </value>
  </value>
</data>`)

	conns, err := parseWorkbenchConnections(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].DB.Port != 3306 {
		t.Errorf("expected default port 3306, got %d", conns[0].DB.Port)
	}
}

func TestParseWorkbenchConnections_SSHDefaultPort(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<data>
  <value type="list" key="connections" content-type="object" content-struct-name="db.mgmt.Connection">
    <value type="object" struct-name="db.mgmt.Connection">
      <value type="string" key="name">ssh-no-port</value>
      <value type="string" key="driver">com.mysql.cj.jdbc.Driver</value>
      <value type="dict" key="parameterValues">
        <value type="string" key="hostName">10.0.0.1</value>
        <value type="string" key="port">3306</value>
        <value type="string" key="userName">app</value>
        <value type="string" key="schema">mydb</value>
        <value type="string" key="sshHost">bastion.example.com</value>
        <value type="string" key="sshUserName">deploy</value>
      </value>
    </value>
  </value>
</data>`)

	conns, err := parseWorkbenchConnections(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conns[0].SSH == nil {
		t.Fatal("expected SSH config")
	}
	if conns[0].SSH.Port != 22 {
		t.Errorf("expected SSH port 22, got %d", conns[0].SSH.Port)
	}
}

func TestParseWorkbenchConnections_NameFallback(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<data>
  <value type="list" key="connections" content-type="object" content-struct-name="db.mgmt.Connection">
    <value type="object" struct-name="db.mgmt.Connection">
      <value type="string" key="name"></value>
      <value type="string" key="driver">com.mysql.cj.jdbc.Driver</value>
      <value type="dict" key="parameterValues">
        <value type="string" key="hostName">myhost.example.com</value>
        <value type="string" key="port">3306</value>
        <value type="string" key="userName">root</value>
        <value type="string" key="schema">test</value>
      </value>
    </value>
  </value>
</data>`)

	conns, err := parseWorkbenchConnections(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conns[0].Name != "myhost.example.com" {
		t.Errorf("expected name 'myhost.example.com', got %q", conns[0].Name)
	}
}

func TestContainsSSH(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Standard TCP/IP over SSH", true},
		{"ssh", true},
		{"SSH", true},
		{"com.mysql.cj.jdbc.Driver", false},
		{"", false},
	}
	for _, tt := range tests {
		got := containsSSH(tt.input)
		if got != tt.want {
			t.Errorf("containsSSH(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestWbParsePort(t *testing.T) {
	tests := []struct {
		input      string
		defaultVal int
		want       int
	}{
		{"3306", 3306, 3306},
		{"33060", 3306, 33060},
		{"", 3306, 3306},
		{"0", 3306, 3306},
		{"abc", 3306, 3306},
	}
	for _, tt := range tests {
		got := wbParsePort(tt.input, tt.defaultVal)
		if got != tt.want {
			t.Errorf("wbParsePort(%q, %d) = %d, want %d", tt.input, tt.defaultVal, got, tt.want)
		}
	}
}

func TestWorkbenchConnectionsPathFor(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"darwin", "Library/Application Support/MySQL/Workbench"},
		{"linux", ".mysql/workbench"},
		{"windows", "AppData/Roaming/MySQL/Workbench"},
	}
	for _, tt := range tests {
		path := workbenchConnectionsPathFor(tt.goos, "/home/test")
		if !strings.Contains(path, tt.want) {
			t.Errorf("workbenchConnectionsPathFor(%q): got %q, want path containing %q", tt.goos, path, tt.want)
		}
		if !strings.HasSuffix(path, "connections.xml") {
			t.Errorf("workbenchConnectionsPathFor(%q): got %q, want path ending with connections.xml", tt.goos, path)
		}
	}
}
