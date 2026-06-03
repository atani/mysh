package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atani/mysh/internal/config"
)

func redashConn(name, url string) config.Connection {
	return config.Connection{
		Name:   name,
		Env:    "development",
		Redash: &config.RedashConfig{URL: url, APIKey: "plain-key", DataSourceID: 1},
	}
}

func TestRunQueryRedashSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Immediate query_result (no job polling).
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query_result":{"data":{"columns":[{"name":"id","type":"integer"},{"name":"email","type":"string"}],"rows":[{"id":1,"email":"a@b.com"}]}}}`))
	}))
	defer srv.Close()

	setupConfig(t, redashConn("r", srv.URL))
	if err := RunQuery([]string{"r", "-e", "SELECT id, email FROM users"}); err != nil {
		t.Fatalf("RunQuery redash: %v", err)
	}
}

func TestRunQueryRedashWithMasking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"query_result":{"data":{"columns":[{"name":"email","type":"string"}],"rows":[{"email":"alice@example.com"}]}}}`))
	}))
	defer srv.Close()

	c := redashConn("r", srv.URL)
	c.Env = "production"
	c.Mask = &config.MaskConfig{Columns: []string{"email"}}
	setupConfig(t, c)

	if err := RunQuery([]string{"r", "-e", "SELECT email FROM users", "--mask"}); err != nil {
		t.Fatalf("RunQuery redash masking: %v", err)
	}
}

func TestRunQueryRedashError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()

	setupConfig(t, redashConn("r", srv.URL))
	if err := RunQuery([]string{"r", "-e", "SELECT 1"}); err == nil {
		t.Error("expected redash API error")
	}
}

func TestRunPingRedashSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	setupConfig(t, redashConn("r", srv.URL))
	if err := RunPing([]string{"r"}); err != nil {
		t.Fatalf("RunPing redash: %v", err)
	}
}

func TestRunPingRedashFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	setupConfig(t, redashConn("r", srv.URL))
	if err := RunPing([]string{"r"}); err == nil {
		t.Error("expected redash ping failure")
	}
}
