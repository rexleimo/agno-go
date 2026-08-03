package airflow

import (
	"context"
	"testing"
)

func TestAirflowToolkit_ListDAGs(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.listDAGs(context.Background(), map[string]interface{}{
		"base_url": "http://localhost:8080",
		"username": "admin",
		"password": "admin",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that DAGs are returned
	dags, ok := resultMap["dags"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected dags array, got: %T", resultMap["dags"])
	}

	if len(dags) == 0 {
		t.Error("Expected at least one DAG")
	}

    // Check total_entries field (Airflow API schema)
    total, ok := resultMap["total_entries"].(int)
    if !ok {
        t.Fatalf("Expected total_entries integer, got: %T", resultMap["total_entries"])
    }

    if total != len(dags) {
        t.Errorf("total_entries mismatch: expected %d, got %d", len(dags), total)
    }
}

func TestAirflowToolkit_TriggerDAGRun(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.triggerDAGRun(context.Background(), map[string]interface{}{
		"base_url": "http://localhost:8080",
		"username": "admin",
		"password": "admin",
		"dag_id":  "example_dag",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check DAG ID
	dagID, ok := resultMap["dag_id"].(string)
	if !ok {
		t.Fatalf("Expected dag_id string, got: %T", resultMap["dag_id"])
	}

	if dagID != "example_dag" {
		t.Errorf("Expected dag_id 'example_dag', got '%s'", dagID)
	}

    // Check dag_run_id
    dagRunID, ok := resultMap["dag_run_id"].(string)
    if !ok {
        t.Fatalf("Expected dag_run_id string, got: %T", resultMap["dag_run_id"])
    }

    if dagRunID == "" {
        t.Error("Expected non-empty dag_run_id")
    }

	// Check state
	state, ok := resultMap["state"].(string)
	if !ok {
		t.Fatalf("Expected state string, got: %T", resultMap["state"])
	}

	if state != "queued" {
		t.Errorf("Expected state 'queued', got '%s'", state)
	}
}

func TestAirflowToolkit_TriggerDAGRun_WithConf(t *testing.T) {
	toolkit := New()

	// Test with configuration parameters
	conf := map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	}

	result, err := toolkit.triggerDAGRun(context.Background(), map[string]interface{}{
		"base_url": "http://localhost:8080",
		"username": "admin",
		"password": "admin",
		"dag_id":  "example_dag",
		"conf":    conf,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check configuration
	resultConf, ok := resultMap["conf"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected conf map, got: %T", resultMap["conf"])
	}

	if len(resultConf) != 2 {
		t.Errorf("Expected 2 conf parameters, got %d", len(resultConf))
	}
}

func TestAirflowToolkit_GetDAGRunStatus(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
    result, err := toolkit.getDAGRunStatus(context.Background(), map[string]interface{}{
        "base_url": "http://localhost:8080",
        "username": "admin",
        "password": "admin",
        "dag_id":  "example_dag",
        "dag_run_id":  "manual__2024-01-15T10:00:00",
    })

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that DAG run status is returned
	dagRun, ok := resultMap["dag_run"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected dag_run map, got: %T", resultMap["dag_run"])
	}

	// Check DAG ID
	dagID, ok := dagRun["dag_id"].(string)
	if !ok {
		t.Fatalf("Expected dag_id string, got: %T", dagRun["dag_id"])
	}

	if dagID != "example_dag" {
		t.Errorf("Expected dag_id 'example_dag', got '%s'", dagID)
	}

	// Check run ID
	runID, ok := dagRun["dag_run_id"].(string)
	if !ok {
		t.Fatalf("Expected dag_run_id string, got: %T", dagRun["dag_run_id"])
	}

	if runID != "manual__2024-01-15T10:00:00" {
		t.Errorf("Expected dag_run_id 'manual__2024-01-15T10:00:00', got '%s'", runID)
	}

	// Check state
	state, ok := dagRun["state"].(string)
	if !ok {
		t.Fatalf("Expected state string, got: %T", dagRun["state"])
	}

	if state != "success" {
		t.Errorf("Expected state 'success', got '%s'", state)
	}
}

func TestAirflowToolkit_ListDAGs_MissingParameters(t *testing.T) {
	toolkit := New()

	// Test missing base_url
	_, err := toolkit.listDAGs(context.Background(), map[string]interface{}{
		"username": "admin",
		"password": "admin",
	})

	if err == nil {
		t.Error("Expected error for missing base_url")
	}

	// Test missing username
	_, err = toolkit.listDAGs(context.Background(), map[string]interface{}{
		"base_url": "http://localhost:8080",
		"password": "admin",
	})

	if err == nil {
		t.Error("Expected error for missing username")
	}

	// Test missing password
	_, err = toolkit.listDAGs(context.Background(), map[string]interface{}{
		"base_url": "http://localhost:8080",
		"username": "admin",
	})

	if err == nil {
		t.Error("Expected error for missing password")
	}
}

func TestAirflowToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"list_dags", "trigger_dag_run", "get_dag_run_status"}
	for _, expected := range expectedFunctions {
		found := false
		for _, function := range functions {
			if function.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function '%s' not found", expected)
		}
	}
}
