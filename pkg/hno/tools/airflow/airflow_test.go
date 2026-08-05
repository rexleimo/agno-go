package airflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAirflowTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin"))
		if r.Header.Get("Authorization") != expectedAuth {
			http.Error(w, `{"detail":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dags":
			_, _ = w.Write([]byte(`{"dags":[{"dag_id":"real_dag","is_paused":false}],"total_entries":1}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/dags/example_dag/dagRuns":
			var request struct {
				Conf map[string]interface{} `json:"conf"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Conf["param"] != "value" {
				http.Error(w, `{"detail":"invalid conf"}`, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"dag_id":"example_dag","dag_run_id":"manual__real","state":"queued","conf":{"param":"value"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/dags/example_dag/dagRuns/manual__real":
			_, _ = w.Write([]byte(`{"dag_id":"example_dag","dag_run_id":"manual__real","state":"success","conf":{"param":"value"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func airflowArgs(serverURL string) map[string]interface{} {
	return map[string]interface{}{
		"base_url": serverURL,
		"username": "admin",
		"password": "admin",
	}
}

func TestAirflowToolkit_ListDAGs(t *testing.T) {
	server := newAirflowTestServer(t)
	defer server.Close()
	result, err := New().Execute(context.Background(), "list_dags", airflowArgs(server.URL))
	if err != nil {
		t.Fatalf("list_dags failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	dags := resultMap["dags"].([]map[string]interface{})
	if len(dags) != 1 || dags[0]["dag_id"] != "real_dag" || resultMap["total_entries"] != 1 {
		t.Fatalf("unexpected DAG response: %#v", resultMap)
	}
}

func TestAirflowToolkit_TriggerDAGRun(t *testing.T) {
	server := newAirflowTestServer(t)
	defer server.Close()
	args := airflowArgs(server.URL)
	args["dag_id"] = "example_dag"
	args["conf"] = map[string]interface{}{"param": "value"}
	result, err := New().Execute(context.Background(), "trigger_dag_run", args)
	if err != nil {
		t.Fatalf("trigger_dag_run failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	if resultMap["dag_run_id"] != "manual__real" || resultMap["state"] != "queued" {
		t.Fatalf("unexpected trigger response: %#v", resultMap)
	}
}

func TestAirflowToolkit_GetDAGRunStatus(t *testing.T) {
	server := newAirflowTestServer(t)
	defer server.Close()
	args := airflowArgs(server.URL)
	args["dag_id"] = "example_dag"
	args["dag_run_id"] = "manual__real"
	result, err := New().Execute(context.Background(), "get_dag_run_status", args)
	if err != nil {
		t.Fatalf("get_dag_run_status failed: %v", err)
	}
	dagRun := result.(map[string]interface{})["dag_run"].(map[string]interface{})
	if dagRun["dag_id"] != "example_dag" || dagRun["state"] != "success" {
		t.Fatalf("unexpected status response: %#v", dagRun)
	}
}

func TestAirflowToolkit_UsesExplicitAPIVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v2/") {
			t.Errorf("expected API v2 path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dags":[],"total_entries":0}`))
	}))
	defer server.Close()
	args := airflowArgs(server.URL + "/api/v2")
	if _, err := New().Execute(context.Background(), "list_dags", args); err != nil {
		t.Fatalf("list_dags with explicit v2 base failed: %v", err)
	}
}

func TestAirflowToolkit_ValidationAndAPIError(t *testing.T) {
	if _, err := New().Execute(context.Background(), "list_dags", map[string]interface{}{
		"username": "admin", "password": "admin",
	}); err == nil {
		t.Error("expected missing base_url error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"failed"}`, http.StatusInternalServerError)
	}))
	defer server.Close()
	if _, err := New().Execute(context.Background(), "list_dags", airflowArgs(server.URL)); err == nil {
		t.Error("expected Airflow API error")
	}
}

func TestAirflowToolkit_New(t *testing.T) {
	tk := New()
	if tk == nil || len(tk.Functions()) != 3 {
		t.Fatalf("expected three registered functions")
	}
}
