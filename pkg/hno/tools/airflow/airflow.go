package airflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// AirflowToolkit provides Apache Airflow REST API integration.
type AirflowToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new Airflow toolkit. Credentials and the instance URL are
// supplied per call so the same toolkit can address multiple deployments.
func New() *AirflowToolkit {
	t := &AirflowToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("airflow"),
		client:      &http.Client{Timeout: 30 * time.Second},
	}

	t.RegisterFunction(&toolkit.Function{
		Name:        "list_dags",
		Description: "List Airflow DAGs using the Airflow REST API",
		Parameters: map[string]toolkit.Parameter{
			"base_url": {Type: "string", Description: "Airflow base URL, optionally including /api/v1 or /api/v2", Required: true},
			"username": {Type: "string", Description: "Airflow username", Required: true},
			"password": {Type: "string", Description: "Airflow password", Required: true},
		},
		Handler: t.listDAGs,
	})
	t.RegisterFunction(&toolkit.Function{
		Name:        "trigger_dag_run",
		Description: "Trigger a DAG run in Airflow using the Airflow REST API",
		Parameters: map[string]toolkit.Parameter{
			"base_url": {Type: "string", Description: "Airflow base URL, optionally including /api/v1 or /api/v2", Required: true},
			"username": {Type: "string", Description: "Airflow username", Required: true},
			"password": {Type: "string", Description: "Airflow password", Required: true},
			"dag_id":   {Type: "string", Description: "DAG ID to trigger", Required: true},
			"conf":     {Type: "object", Description: "Configuration parameters for the DAG run", Required: false},
		},
		Handler: t.triggerDAGRun,
	})
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_dag_run_status",
		Description: "Get a DAG run from the Airflow REST API",
		Parameters: map[string]toolkit.Parameter{
			"base_url":   {Type: "string", Description: "Airflow base URL, optionally including /api/v1 or /api/v2", Required: true},
			"username":   {Type: "string", Description: "Airflow username", Required: true},
			"password":   {Type: "string", Description: "Airflow password", Required: true},
			"dag_id":     {Type: "string", Description: "DAG ID", Required: true},
			"dag_run_id": {Type: "string", Description: "DAG run ID", Required: true},
		},
		Handler: t.getDAGRunStatus,
	})
	return t
}

func airflowString(args map[string]interface{}, name string) (string, error) {
	value, ok := args[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return strings.TrimSpace(value), nil
}

func airflowAPIBase(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid Airflow base_url: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("base_url must be an absolute http or https URL")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(path, "/api/v1") && !strings.HasSuffix(path, "/api/v2") {
		path += "/api/v1"
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func airflowEndpoint(baseURL, path string) (string, error) {
	base, err := airflowAPIBase(baseURL)
	if err != nil {
		return "", err
	}
	return base + "/" + strings.TrimLeft(path, "/"), nil
}

func (a *AirflowToolkit) listDAGs(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := airflowString(args, "base_url")
	if err != nil {
		return nil, err
	}
	username, err := airflowString(args, "username")
	if err != nil {
		return nil, err
	}
	password, err := airflowString(args, "password")
	if err != nil {
		return nil, err
	}
	endpoint, err := airflowEndpoint(baseURL, "/dags")
	if err != nil {
		return nil, err
	}
	resp, err := a.makeAirflowRequest(ctx, http.MethodGet, endpoint, username, password, nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		DAGs         []map[string]interface{} `json:"dags"`
		TotalEntries int                      `json:"total_entries"`
	}
	if err := a.parseAirflowResponse(resp, &payload); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"dags":          payload.DAGs,
		"total_entries": payload.TotalEntries,
	}, nil
}

func (a *AirflowToolkit) triggerDAGRun(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := airflowString(args, "base_url")
	if err != nil {
		return nil, err
	}
	username, err := airflowString(args, "username")
	if err != nil {
		return nil, err
	}
	password, err := airflowString(args, "password")
	if err != nil {
		return nil, err
	}
	dagID, err := airflowString(args, "dag_id")
	if err != nil {
		return nil, err
	}
	conf := map[string]interface{}{}
	if value, ok := args["conf"]; ok {
		var valid bool
		conf, valid = value.(map[string]interface{})
		if !valid {
			return nil, fmt.Errorf("conf must be an object")
		}
	}
	body, err := json.Marshal(map[string]interface{}{"conf": conf})
	if err != nil {
		return nil, fmt.Errorf("failed to encode Airflow DAG run request: %w", err)
	}
	endpoint, err := airflowEndpoint(baseURL, "/dags/"+url.PathEscape(dagID)+"/dagRuns")
	if err != nil {
		return nil, err
	}
	resp, err := a.makeAirflowRequest(ctx, http.MethodPost, endpoint, username, password, body)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	if err := a.parseAirflowResponse(resp, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *AirflowToolkit) getDAGRunStatus(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	baseURL, err := airflowString(args, "base_url")
	if err != nil {
		return nil, err
	}
	username, err := airflowString(args, "username")
	if err != nil {
		return nil, err
	}
	password, err := airflowString(args, "password")
	if err != nil {
		return nil, err
	}
	dagID, err := airflowString(args, "dag_id")
	if err != nil {
		return nil, err
	}
	runID, err := airflowString(args, "dag_run_id")
	if err != nil {
		return nil, err
	}
	endpoint, err := airflowEndpoint(baseURL, "/dags/"+url.PathEscape(dagID)+"/dagRuns/"+url.PathEscape(runID))
	if err != nil {
		return nil, err
	}
	resp, err := a.makeAirflowRequest(ctx, http.MethodGet, endpoint, username, password, nil)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	if err := a.parseAirflowResponse(resp, &payload); err != nil {
		return nil, err
	}
	return map[string]interface{}{"dag_run": payload}, nil
}

func (a *AirflowToolkit) makeAirflowRequest(ctx context.Context, method, endpoint, username, password string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create Airflow request: %w", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Airflow request failed: %w", err)
	}
	return resp, nil
}

func (a *AirflowToolkit) parseAirflowResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		message := strings.TrimSpace(string(body))
		if message == "" {
			return fmt.Errorf("Airflow API request failed with status: %d", resp.StatusCode)
		}
		return fmt.Errorf("Airflow API request failed with status: %d: %s", resp.StatusCode, message)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Airflow API response: %w", err)
	}
	return nil
}
