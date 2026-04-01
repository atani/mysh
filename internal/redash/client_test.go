package redash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuerySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query_results" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Key test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		resp := queryResultResponse{
			QueryResult: &QueryResult{
				Data: &queryData{
					Columns: []column{
						{Name: "id", Type: "integer"},
						{Name: "name", Type: "string"},
						{Name: "email", Type: "string"},
					},
					Rows: []json.RawMessage{
						json.RawMessage(`{"id": 1, "name": "Alice", "email": "alice@example.com"}`),
						json.RawMessage(`{"id": 2, "name": "Bob", "email": "bob@example.com"}`),
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	result, err := client.Query("SELECT * FROM users", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(result.Headers) != 3 {
		t.Fatalf("headers = %d, want 3", len(result.Headers))
	}
	if result.Headers[0] != "id" || result.Headers[1] != "name" || result.Headers[2] != "email" {
		t.Errorf("headers = %v", result.Headers)
	}

	if len(result.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(result.Rows))
	}
	if result.Rows[0][1] != "Alice" {
		t.Errorf("row[0][1] = %q, want Alice", result.Rows[0][1])
	}
	if result.Rows[1][2] != "bob@example.com" {
		t.Errorf("row[1][2] = %q, want bob@example.com", result.Rows[1][2])
	}
}

func TestQueryNullValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := queryResultResponse{
			QueryResult: &QueryResult{
				Data: &queryData{
					Columns: []column{{Name: "val", Type: "string"}},
					Rows: []json.RawMessage{
						json.RawMessage(`{"val": null}`),
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	result, err := client.Query("SELECT NULL", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if result.Rows[0][0] != "NULL" {
		t.Errorf("null value = %q, want NULL", result.Rows[0][0])
	}
}

func TestQueryAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Access denied"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil {
		t.Error("expected error for 403 response")
	}
}

func TestPingSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/session" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user": {}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	if err := client.Ping(); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestPingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	if err := client.Ping(); err == nil {
		t.Error("expected error for 401 response")
	}
}

func TestToResultEmpty(t *testing.T) {
	result := toResult(nil)
	if result.Headers != nil {
		t.Error("expected nil headers")
	}
}
