package airflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// AirflowToolkit provides Apache Airflow API integration capabilities
// This is a basic implementation that can be extended with specific Airflow API endpoints

// AirflowToolkit provides Airflow API integration
type AirflowToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new Airflow toolkit
func New() *AirflowToolkit {
	t := &AirflowToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("airflow"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

    // Register Airflow DAG list function
	t.RegisterFunction(&toolkit.Function{
		Name:        "list_dags",
		Description: "List Airflow DAGs (Directed Acyclic Graphs)",
		Parameters: map[string]toolkit.Parameter{
            "base_url": {
                Type:        "string",
                Description: "Airflow base URL (e.g., http://localhost:8080)",
                Required:    true,
            },
            "username": {
                Type:        "string",
                Description: "Airflow username",
                Required:    true,
            },
            "password": {
                Type:        "string",
                Description: "Airflow password",
                Required:    true,
            },
        },
        Handler: t.listDAGs,
    })

	// Register Airflow DAG run trigger function
	t.RegisterFunction(&toolkit.Function{
		Name:        "trigger_dag_run",
		Description: "Trigger a DAG run in Airflow",
		Parameters: map[string]toolkit.Parameter{
            "base_url": {
                Type:        "string",
                Description: "Airflow base URL",
                Required:    true,
            },
            "username": {
                Type:        "string",
                Description: "Airflow username",
                Required:    true,
            },
            "password": {
                Type:        "string",
                Description: "Airflow password",
                Required:    true,
            },
            "dag_id": {
                Type:        "string",
                Description: "DAG ID to trigger",
                Required:    true,
            },
            "conf": {
                Type:        "object",
                Description: "Configuration parameters for the DAG run (optional)",
                Required:    false,
            },
        },
        Handler: t.triggerDAGRun,
    })

	// Register Airflow DAG run status function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_dag_run_status",
		Description: "Get the status of a specific DAG run",
		Parameters: map[string]toolkit.Parameter{
            "base_url": {
                Type:        "string",
                Description: "Airflow base URL",
                Required:    true,
            },
            "username": {
                Type:        "string",
                Description: "Airflow username",
                Required:    true,
            },
            "password": {
                Type:        "string",
                Description: "Airflow password",
                Required:    true,
            },
            "dag_id": {
                Type:        "string",
                Description: "DAG ID",
                Required:    true,
            },
            "dag_run_id": {
                Type:        "string",
                Description: "DAG run ID (a.k.a. run_id)",
                Required:    true,
            },
        },
        Handler: t.getDAGRunStatus,
    })

	return t
}

// listDAGs lists Airflow DAGs
func (a *AirflowToolkit) listDAGs(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url must be a string")
	}

	_, ok = args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password must be a string")
	}

    // For now, return a mock response since we need actual Airflow API integration
    // In a real implementation, this would call the Airflow API
    // Schema aligned to Airflow OpenAPI (Context7 /apache/airflow):
    // GET /api/v2/dags -> { dags: [...], total_entries: N }
    mockDAGs := []map[string]interface{}{
        {
            "dag_id":          "example_dag",
            "dag_display_name": "Example DAG",
            "description":     "Example DAG for demonstration",
            "is_paused":       false,
            "is_active":       true,
            "last_parsed_time": "2024-01-15T10:00:00Z",
            "tags":            []map[string]interface{}{{"name": "demo"}},
        },
        {
            "dag_id":          "data_pipeline",
            "dag_display_name": "Data Pipeline",
            "description":     "Data processing pipeline",
            "is_paused":       false,
            "is_active":       true,
            "last_parsed_time": "2024-01-15T09:30:00Z",
            "tags":            []map[string]interface{}{{"name": "etl"}},
        },
        {
            "dag_id":          "backup_dag",
            "dag_display_name": "Backup DAG",
            "description":     "Database backup DAG",
            "is_paused":       true,
            "is_active":       false,
            "last_parsed_time": "2024-01-14T16:00:00Z",
            "tags":            []map[string]interface{}{{"name": "ops"}},
        },
    }

    return map[string]interface{}{
        "dags":          mockDAGs,
        "total_entries": len(mockDAGs),
        "note":          "Placeholder implementation. Aligns with Airflow API schema; integrate real API calls.",
    }, nil
}

// triggerDAGRun triggers a DAG run in Airflow
func (a *AirflowToolkit) triggerDAGRun(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url must be a string")
	}

	_, ok = args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

	_, ok = args["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password must be a string")
	}

	dagID, ok := args["dag_id"].(string)
	if !ok {
		return nil, fmt.Errorf("dag_id must be a string")
	}

	conf := map[string]interface{}{}
	if confArg, ok := args["conf"].(map[string]interface{}); ok {
		conf = confArg
	}

    // For now, return a mock response aligned to Airflow API v2
    mockDagRunID := fmt.Sprintf("manual__%s", time.Now().Format("2006-01-02T15:04:05"))

    return map[string]interface{}{
        "dag_id":       dagID,
        "dag_run_id":   mockDagRunID,
        "conf":         conf,
        "logical_date": time.Now().Format(time.RFC3339),
        "state":        "queued",
        "note":         "Placeholder implementation. Aligns with Airflow API schema; integrate real API calls.",
    }, nil
}

// getDAGRunStatus gets the status of a specific DAG run
func (a *AirflowToolkit) getDAGRunStatus(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, ok := args["base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("base_url must be a string")
	}

	_, ok = args["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username must be a string")
	}

    _, ok = args["password"].(string)
    if !ok {
        return nil, fmt.Errorf("password must be a string")
    }

    dagID, ok := args["dag_id"].(string)
    if !ok {
        return nil, fmt.Errorf("dag_id must be a string")
    }

    // Prefer dag_run_id, but accept legacy run_id for compatibility
    runID, ok := args["dag_run_id"].(string)
    if !ok {
        // fallback to legacy key
        legacy, lok := args["run_id"].(string)
        if !lok {
            return nil, fmt.Errorf("dag_run_id must be a string")
        }
        runID = legacy
    }

	// For now, return a mock response
    mockStatus := map[string]interface{}{
        "dag_id":          dagID,
        "dag_run_id":      runID,
        "state":           "success",
        "logical_date":    "2024-01-15T10:00:00Z",
        "start_date":      "2024-01-15T10:00:05Z",
        "end_date":        "2024-01-15T10:02:30Z",
        "duration":        145.5,
        "conf":            map[string]interface{}{},
        "external_trigger": true,
        "task_instances": []map[string]interface{}{
            {
                "task_id":        "start_task",
                "state":          "success",
                "start_date":     "2024-01-15T10:00:10Z",
                "end_date":       "2024-01-15T10:00:15Z",
                "duration":       5.0,
                "try_number":     1,
            },
            {
                "task_id":        "process_data",
                "state":          "success",
                "start_date":     "2024-01-15T10:00:20Z",
                "end_date":       "2024-01-15T10:01:45Z",
                "duration":       85.0,
                "try_number":     1,
            },
            {
                "task_id":        "end_task",
                "state":          "success",
                "start_date":     "2024-01-15T10:01:50Z",
                "end_date":       "2024-01-15T10:02:30Z",
                "duration":       40.0,
                "try_number":     1,
            },
        },
    }

    return map[string]interface{}{
        "dag_run": mockStatus,
        "note":    "Placeholder implementation. Aligns with Airflow API schema; integrate real API calls.",
    }, nil
}

// Helper function to make authenticated Airflow API requests
func (a *AirflowToolkit) makeAirflowRequest(ctx context.Context, url, username, password string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth for Airflow API
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Helper function to parse Airflow API response
func (a *AirflowToolkit) parseAirflowResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Airflow API request failed with status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode Airflow API response: %w", err)
	}

	return nil
}
