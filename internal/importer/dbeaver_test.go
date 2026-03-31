package importer

import (
	"testing"
)

func TestParseDBeaverDataSources(t *testing.T) {
	data := []byte(`{
		"connections": {
			"mysql8-001": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "my-db",
				"folder": "Work",
				"configuration": {
					"host": "10.0.0.1",
					"port": "3306",
					"database": "appdb",
					"auth-model": "native",
					"auth-properties": {
						"user": "admin"
					},
					"handlers": {
						"ssh_tunnel": {
							"type": "TUNNEL",
							"enabled": true,
							"properties": {
								"host": "bastion.example.com",
								"port": 22,
								"authType": "PUBLIC_KEY",
								"keyPath": "/home/user/.ssh/id_rsa",
								"user": "deploy"
							}
						}
					}
				}
			},
			"mysql8-002": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "local-db",
				"configuration": {
					"host": "127.0.0.1",
					"port": "33306",
					"database": "devdb",
					"auth-model": "native",
					"auth-properties": {
						"user": "root"
					}
				}
			},
			"postgres-001": {
				"provider": "postgresql",
				"driver": "postgres",
				"name": "pg-db",
				"configuration": {
					"host": "pg.example.com",
					"port": "5432",
					"database": "analytics"
				}
			}
		}
	}`)

	conns, err := parseDBeaverDataSources(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(conns))
	}

	// Find connections by name
	var withSSH, noSSH ImportedConnection
	for _, c := range conns {
		switch c.Name {
		case "my-db":
			withSSH = c
		case "local-db":
			noSSH = c
		}
	}

	// Connection with SSH
	if withSSH.Name != "my-db" {
		t.Errorf("expected name 'my-db', got %q", withSSH.Name)
	}
	if withSSH.Folder != "Work" {
		t.Errorf("expected folder 'Work', got %q", withSSH.Folder)
	}
	if withSSH.DB.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", withSSH.DB.Host)
	}
	if withSSH.DB.Port != 3306 {
		t.Errorf("expected port 3306, got %d", withSSH.DB.Port)
	}
	if withSSH.DB.User != "admin" {
		t.Errorf("expected user 'admin', got %q", withSSH.DB.User)
	}
	if withSSH.DB.Database != "appdb" {
		t.Errorf("expected database 'appdb', got %q", withSSH.DB.Database)
	}
	if withSSH.SSH == nil {
		t.Fatal("expected SSH config, got nil")
	}
	if withSSH.SSH.Host != "bastion.example.com" {
		t.Errorf("expected SSH host 'bastion.example.com', got %q", withSSH.SSH.Host)
	}
	if withSSH.SSH.Port != 22 {
		t.Errorf("expected SSH port 22, got %d", withSSH.SSH.Port)
	}
	if withSSH.SSH.User != "deploy" {
		t.Errorf("expected SSH user 'deploy', got %q", withSSH.SSH.User)
	}
	if withSSH.SSH.Key != "/home/user/.ssh/id_rsa" {
		t.Errorf("expected SSH key path, got %q", withSSH.SSH.Key)
	}

	// Connection without SSH
	if noSSH.Name != "local-db" {
		t.Errorf("expected name 'local-db', got %q", noSSH.Name)
	}
	if noSSH.DB.Port != 33306 {
		t.Errorf("expected port 33306, got %d", noSSH.DB.Port)
	}
	if noSSH.SSH != nil {
		t.Errorf("expected no SSH config, got %+v", noSSH.SSH)
	}
}

func TestParseDBeaverDataSources_PortAsNumber(t *testing.T) {
	data := []byte(`{
		"connections": {
			"mysql-001": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "numeric-port",
				"configuration": {
					"host": "localhost",
					"port": 3307,
					"database": "test"
				}
			}
		}
	}`)

	conns, err := parseDBeaverDataSources(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].DB.Port != 3307 {
		t.Errorf("expected port 3307, got %d", conns[0].DB.Port)
	}
}

func TestParseDBeaverDataSources_DisabledSSH(t *testing.T) {
	data := []byte(`{
		"connections": {
			"mysql-001": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "disabled-ssh",
				"configuration": {
					"host": "localhost",
					"port": "3306",
					"database": "test",
					"handlers": {
						"ssh_tunnel": {
							"type": "TUNNEL",
							"enabled": false,
							"properties": {
								"host": "bastion.example.com"
							}
						}
					}
				}
			}
		}
	}`)

	conns, err := parseDBeaverDataSources(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].SSH != nil {
		t.Errorf("expected no SSH (disabled), got %+v", conns[0].SSH)
	}
}

func TestParseDBeaverDataSources_InvalidJSON(t *testing.T) {
	_, err := parseDBeaverDataSources([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseDBeaverDataSources_Empty(t *testing.T) {
	data := []byte(`{"connections": {}}`)
	conns, err := parseDBeaverDataSources(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestParseDBeaverDataSources_MissingPort(t *testing.T) {
	data := []byte(`{
		"connections": {
			"mysql-001": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "no-port",
				"configuration": {
					"host": "localhost",
					"database": "test"
				}
			}
		}
	}`)

	conns, err := parseDBeaverDataSources(data)
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

func TestParseDBeaverDataSources_SSHPortDefaultsTo22(t *testing.T) {
	data := []byte(`{
		"connections": {
			"mysql-001": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "ssh-no-port",
				"configuration": {
					"host": "10.0.0.1",
					"port": "3306",
					"database": "test",
					"handlers": {
						"ssh_tunnel": {
							"type": "TUNNEL",
							"enabled": true,
							"properties": {
								"host": "bastion.example.com"
							}
						}
					}
				}
			}
		}
	}`)

	conns, err := parseDBeaverDataSources(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conns[0].SSH == nil {
		t.Fatal("expected SSH config, got nil")
	}
	if conns[0].SSH.Port != 22 {
		t.Errorf("expected SSH port 22, got %d", conns[0].SSH.Port)
	}
}

func TestParseDBeaverDataSources_SSHEmptyProperties(t *testing.T) {
	data := []byte(`{
		"connections": {
			"mysql-001": {
				"provider": "mysql",
				"driver": "mysql8",
				"name": "empty-ssh-props",
				"configuration": {
					"host": "localhost",
					"port": "3306",
					"database": "test",
					"handlers": {
						"ssh_tunnel": {
							"type": "TUNNEL",
							"enabled": true,
							"properties": {}
						}
					}
				}
			}
		}
	}`)

	conns, err := parseDBeaverDataSources(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].SSH != nil {
		t.Errorf("expected no SSH (empty host), got %+v", conns[0].SSH)
	}
}

func TestParsePort_InvalidString(t *testing.T) {
	raw := []byte(`"not-a-number"`)
	port := parsePort(raw, 3306)
	if port != 3306 {
		t.Errorf("expected default 3306, got %d", port)
	}
}
