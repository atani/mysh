package db

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgproto3"
)

// fakePostgresServer accepts one connection on a local port, answers the
// startup message with authRequest, and reports whether the client sent a
// password message. After authentication it answers queries with an empty
// response so a Ping succeeds.
func fakePostgresServer(t *testing.T, authRequest pgproto3.BackendMessage, authType uint32) (port int, sentPassword <-chan bool) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	result := make(chan bool, 1)
	go func() {
		sent := false
		defer func() { result <- sent }()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		be := pgproto3.NewBackend(conn, conn)
		if _, err := be.ReceiveStartupMessage(); err != nil {
			return
		}
		be.Send(authRequest)
		if authType != pgproto3.AuthTypeOk {
			if err := be.Flush(); err != nil {
				return
			}
			if err := be.SetAuthType(authType); err != nil {
				return
			}
			msg, err := be.Receive()
			if err != nil {
				return
			}
			if _, ok := msg.(*pgproto3.PasswordMessage); !ok {
				return
			}
			sent = true
			be.Send(&pgproto3.AuthenticationOk{})
		}
		be.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		if err := be.Flush(); err != nil {
			return
		}
		for {
			msg, err := be.Receive()
			if err != nil {
				return
			}
			switch msg.(type) {
			case *pgproto3.Query:
				be.Send(&pgproto3.EmptyQueryResponse{})
				be.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
				if err := be.Flush(); err != nil {
					return
				}
			case *pgproto3.Terminate:
				return
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, result
}

func pingFake(t *testing.T, port int) error {
	t.Helper()
	conn, err := OpenPostgres(PostgresParams{
		Host: "127.0.0.1", Port: port, User: "app", Password: "top-secret", Database: "d", SSLMode: "disable",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	return conn.Ping()
}

// A server (or an active man-in-the-middle on an unauthenticated TLS or
// plaintext connection) that asks for cleartext password authentication must
// not receive the password.
func TestOpenPostgresRefusesCleartextPassword(t *testing.T) {
	port, sentPassword := fakePostgresServer(t, &pgproto3.AuthenticationCleartextPassword{}, pgproto3.AuthTypeCleartextPassword)
	err := pingFake(t, port)
	if err == nil {
		t.Fatal("expected the connection to be refused")
	}
	if !strings.Contains(err.Error(), "password authentication") {
		t.Errorf("error should name the refused method: %v", err)
	}
	if strings.Contains(err.Error(), "top-secret") {
		t.Errorf("error leaks the password: %v", err)
	}
	if <-sentPassword {
		t.Error("the password was sent to a server that requested cleartext authentication")
	}
}

func TestOpenPostgresAllowsMD5AndTrust(t *testing.T) {
	t.Run("md5", func(t *testing.T) {
		port, sentPassword := fakePostgresServer(t, &pgproto3.AuthenticationMD5Password{Salt: [4]byte{1, 2, 3, 4}}, pgproto3.AuthTypeMD5Password)
		if err := pingFake(t, port); err != nil {
			t.Fatalf("md5 authentication must be allowed: %v", err)
		}
		if !<-sentPassword {
			t.Error("expected an md5 password message")
		}
	})
	t.Run("trust", func(t *testing.T) {
		port, _ := fakePostgresServer(t, &pgproto3.AuthenticationOk{}, pgproto3.AuthTypeOk)
		if err := pingFake(t, port); err != nil {
			t.Fatalf("a server that requires no authentication must be allowed: %v", err)
		}
	})
}
