package redash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestQueryViaJobPolling covers the asynchronous path where the initial
// query_results call returns a job, which mysh then polls until success and
// fetches the final result. The job reports "started" once, then "success",
// exercising both the polling-continuation and the fetch branches.
func TestQueryViaJobPolling(t *testing.T) {
	jobPolls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/query_results" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(queryResultResponse{
				Job: &job{ID: "job-1", Status: jobStatusPending},
			})
		case strings.HasPrefix(r.URL.Path, "/api/jobs/"):
			jobPolls++
			status := jobStatusSuccess
			if jobPolls == 1 {
				status = jobStatusStarted // force one extra poll iteration
			}
			_ = json.NewEncoder(w).Encode(jobResponse{
				Job: &job{ID: "job-1", Status: status, QueryResultID: 42},
			})
		case r.URL.Path == "/api/query_results/42":
			_ = json.NewEncoder(w).Encode(queryResultResponse{
				QueryResult: &QueryResult{
					Data: &queryData{
						Columns: []column{{Name: "n", Type: "integer"}},
						Rows:    []json.RawMessage{json.RawMessage(`{"n": 7}`)},
					},
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	result, err := client.Query("SELECT 7", 1)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "7" {
		t.Errorf("unexpected result: %+v", result)
	}
	if jobPolls < 2 {
		t.Errorf("expected at least 2 job polls, got %d", jobPolls)
	}
}

func TestQueryJobFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/query_results" {
			_ = json.NewEncoder(w).Encode(queryResultResponse{Job: &job{ID: "j", Status: jobStatusPending}})
			return
		}
		_ = json.NewEncoder(w).Encode(jobResponse{
			Job: &job{ID: "j", Status: jobStatusFailure, Error: "boom"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected job failure error containing 'boom', got %v", err)
	}
}

func TestQueryJobUnknownStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/query_results" {
			_ = json.NewEncoder(w).Encode(queryResultResponse{Job: &job{ID: "j", Status: jobStatusPending}})
			return
		}
		_ = json.NewEncoder(w).Encode(jobResponse{Job: &job{ID: "j", Status: 99}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "unknown job status") {
		t.Errorf("expected unknown status error, got %v", err)
	}
}

func TestQueryJobMissingJobInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/query_results" {
			_ = json.NewEncoder(w).Encode(queryResultResponse{Job: &job{ID: "j", Status: jobStatusPending}})
			return
		}
		// jobs endpoint returns an empty body -> jr.Job == nil
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "unexpected job response") {
		t.Errorf("expected 'unexpected job response' error, got %v", err)
	}
}

func TestQueryJobInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/query_results" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(queryResultResponse{Job: &job{ID: "j", Status: jobStatusPending}})
			return
		}
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "parsing job response") {
		t.Errorf("expected job parse error, got %v", err)
	}
}

func TestFetchQueryResultMissingResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/query_results" {
			_ = json.NewEncoder(w).Encode(queryResultResponse{Job: &job{ID: "j", Status: jobStatusPending}})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/jobs/") {
			_ = json.NewEncoder(w).Encode(jobResponse{Job: &job{ID: "j", Status: jobStatusSuccess, QueryResultID: 1}})
			return
		}
		// query_results/1 returns neither QueryResult nor Job
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "no query result") {
		t.Errorf("expected 'no query result' error, got %v", err)
	}
}

func TestQueryRequestError(t *testing.T) {
	// Closed server -> connection refused -> do() returns an error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := NewClient(url, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil {
		t.Error("expected transport error for closed server")
	}
}

func TestQueryInvalidResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "parsing response") {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestQueryNoResultNoJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.Query("SELECT 1", 1)
	if err == nil || !strings.Contains(err.Error(), "no query_result or job") {
		t.Errorf("expected 'no query_result or job' error, got %v", err)
	}
}

func TestPingRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := NewClient(url, "key")
	if err := client.Ping(); err == nil {
		t.Error("expected transport error for closed server")
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://redash.example.com/", "k")
	if c.BaseURL != "https://redash.example.com" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", c.BaseURL)
	}
	if c.HTTPClient == nil || c.HTTPClient.Timeout != 5*time.Minute {
		t.Error("expected HTTP client with 5m timeout")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short = %q, want unchanged", got)
	}
	got := truncate("0123456789abc", 5)
	if got != "01234..." {
		t.Errorf("truncate = %q, want 01234...", got)
	}
	// Boundary: len == maxLen returns unchanged.
	if got := truncate("12345", 5); got != "12345" {
		t.Errorf("truncate at boundary = %q, want unchanged", got)
	}
}

func TestToResultSkipsUnparsableRow(t *testing.T) {
	qr := &QueryResult{
		Data: &queryData{
			Columns: []column{{Name: "x"}},
			Rows: []json.RawMessage{
				json.RawMessage(`{"x": 1}`),
				json.RawMessage(`not-an-object`), // skipped
			},
		},
	}
	r := toResult(qr)
	if len(r.Rows) != 1 {
		t.Fatalf("expected 1 parsable row, got %d", len(r.Rows))
	}
	if r.Rows[0][0] != "1" {
		t.Errorf("row[0][0] = %q, want 1", r.Rows[0][0])
	}
}
