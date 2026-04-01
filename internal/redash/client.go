package redash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with the Redash API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a Redash API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type queryRequest struct {
	Query        string `json:"query"`
	DataSourceID int    `json:"data_source_id"`
}

type queryResultResponse struct {
	QueryResult *QueryResult `json:"query_result"`
	Job         *job         `json:"job"`
}

type job struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

type jobResponse struct {
	Job *job `json:"job"`
}

// QueryResult holds the result of a Redash query.
type QueryResult struct {
	Data *queryData `json:"data"`
}

type queryData struct {
	Columns []column          `json:"columns"`
	Rows    []json.RawMessage `json:"rows"`
}

type column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Result is a simplified query result with headers and rows.
type Result struct {
	Headers []string
	Rows    [][]string
}

// Query executes a SQL query via the Redash API and returns structured results.
func (c *Client) Query(query string, dataSourceID int) (*Result, error) {
	body, err := json.Marshal(queryRequest{
		Query:        query,
		DataSourceID: dataSourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.do("POST", "/api/query_results", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Redash API error (HTTP %d): %s", resp.StatusCode, truncate(string(data), 200))
	}

	var qrr queryResultResponse
	if err := json.Unmarshal(data, &qrr); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// If a job was returned, poll until complete
	if qrr.Job != nil {
		qr, err := c.waitForJob(qrr.Job.ID)
		if err != nil {
			return nil, err
		}
		return toResult(qr), nil
	}

	if qrr.QueryResult == nil {
		return nil, fmt.Errorf("unexpected Redash response: no query_result or job")
	}

	return toResult(qrr.QueryResult), nil
}

// Ping tests connectivity to the Redash API.
func (c *Client) Ping() error {
	resp, err := c.do("GET", "/api/session", nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Redash API returned HTTP %d", resp.StatusCode)
	}
	return nil
}

const (
	jobStatusPending  = 1
	jobStatusStarted  = 2
	jobStatusSuccess  = 3
	jobStatusFailure  = 4
)

func (c *Client) waitForJob(jobID string) (*QueryResult, error) {
	for {
		resp, err := c.do("GET", fmt.Sprintf("/api/jobs/%s", jobID), nil)
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading job response: %w", err)
		}

		var jr jobResponse
		if err := json.Unmarshal(data, &jr); err != nil {
			return nil, fmt.Errorf("parsing job response: %w", err)
		}

		if jr.Job == nil {
			return nil, fmt.Errorf("unexpected job response")
		}

		switch jr.Job.Status {
		case jobStatusSuccess:
			// Fetch the query result
			return c.fetchQueryResult(jr.Job.ID)
		case jobStatusFailure:
			return nil, fmt.Errorf("Redash query failed: %s", jr.Job.Error)
		case jobStatusPending, jobStatusStarted:
			time.Sleep(500 * time.Millisecond)
			continue
		default:
			return nil, fmt.Errorf("unknown job status: %d", jr.Job.Status)
		}
	}
}

func (c *Client) fetchQueryResult(queryResultID string) (*QueryResult, error) {
	resp, err := c.do("GET", fmt.Sprintf("/api/query_results/%s", queryResultID), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading query result: %w", err)
	}

	var qrr queryResultResponse
	if err := json.Unmarshal(data, &qrr); err != nil {
		return nil, fmt.Errorf("parsing query result: %w", err)
	}

	if qrr.QueryResult == nil {
		return nil, fmt.Errorf("no query result in response")
	}

	return qrr.QueryResult, nil
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Key "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Redash API request failed: %w", err)
	}

	return resp, nil
}

func toResult(qr *QueryResult) *Result {
	if qr == nil || qr.Data == nil {
		return &Result{}
	}

	headers := make([]string, len(qr.Data.Columns))
	for i, col := range qr.Data.Columns {
		headers[i] = col.Name
	}

	rows := make([][]string, 0, len(qr.Data.Rows))
	for _, rawRow := range qr.Data.Rows {
		var rowMap map[string]interface{}
		if err := json.Unmarshal(rawRow, &rowMap); err != nil {
			continue
		}

		row := make([]string, len(headers))
		for i, h := range headers {
			val, ok := rowMap[h]
			if !ok || val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		rows = append(rows, row)
	}

	return &Result{
		Headers: headers,
		Rows:    rows,
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
