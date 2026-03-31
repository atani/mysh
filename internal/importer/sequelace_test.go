package importer

import (
	"testing"
)

func TestParseSequelAceFavorites(t *testing.T) {
	data := []byte(`{
		"Favorites": [
			{
				"name": "production-db",
				"host": "db.example.com",
				"port": 3306,
				"user": "app_user",
				"database": "myapp",
				"type": 2,
				"sshHost": "bastion.example.com",
				"sshPort": 22,
				"sshUser": "deploy",
				"sshKeyLocation": "/Users/test/.ssh/id_rsa",
				"sshKeyLocationEnabled": "1"
			},
			{
				"name": "local-db",
				"host": "127.0.0.1",
				"port": 3306,
				"user": "root",
				"database": "devdb",
				"type": 0
			}
		]
	}`)

	conns, err := parseSequelAceFavorites(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(conns))
	}

	// SSH connection
	ssh := conns[0]
	if ssh.Name != "production-db" {
		t.Errorf("expected name 'production-db', got %q", ssh.Name)
	}
	if ssh.DB.Host != "db.example.com" {
		t.Errorf("expected host 'db.example.com', got %q", ssh.DB.Host)
	}
	if ssh.DB.User != "app_user" {
		t.Errorf("expected user 'app_user', got %q", ssh.DB.User)
	}
	if ssh.SSH == nil {
		t.Fatal("expected SSH config, got nil")
	}
	if ssh.SSH.Host != "bastion.example.com" {
		t.Errorf("expected SSH host 'bastion.example.com', got %q", ssh.SSH.Host)
	}
	if ssh.SSH.User != "deploy" {
		t.Errorf("expected SSH user 'deploy', got %q", ssh.SSH.User)
	}

	// TCP/IP connection
	tcp := conns[1]
	if tcp.Name != "local-db" {
		t.Errorf("expected name 'local-db', got %q", tcp.Name)
	}
	if tcp.SSH != nil {
		t.Errorf("expected no SSH, got %+v", tcp.SSH)
	}
}

func TestParseSequelAceFavorites_Empty(t *testing.T) {
	data := []byte(`{"Favorites": []}`)
	conns, err := parseSequelAceFavorites(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestParseSequelAceFavorites_DefaultPort(t *testing.T) {
	data := []byte(`{
		"Favorites": [
			{
				"name": "no-port",
				"host": "localhost",
				"port": 0,
				"user": "root",
				"database": "test",
				"type": 0
			}
		]
	}`)

	conns, err := parseSequelAceFavorites(data)
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
