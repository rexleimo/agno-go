package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	toolkit := New()

	if toolkit.Name() != "http" {
		t.Errorf("Name() = %v, want http", toolkit.Name())
	}

	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Functions() count = %v, want 2", len(functions))
	}

	if _, exists := functions["http_get"]; !exists {
		t.Error("http_get function not found")
	}

	if _, exists := functions["http_post"]; !exists {
		t.Error("http_post function not found")
	}
}

func TestHTTPToolkit_Get(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	toolkit := New()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid GET request",
			args: map[string]interface{}{
				"url": server.URL,
			},
			wantErr: false,
		},
		{
			name:    "missing URL",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "invalid URL type",
			args: map[string]interface{}{
				"url": 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolkit.httpGet(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("httpGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Error("httpGet() result is not a map")
					return
				}

				statusCode, ok := resultMap["status_code"].(int)
				if !ok {
					t.Error("status_code not found or not int")
					return
				}

				if statusCode != http.StatusOK {
					t.Errorf("status_code = %v, want %v", statusCode, http.StatusOK)
				}

				body, ok := resultMap["body"].(string)
				if !ok {
					t.Error("body not found or not string")
					return
				}

				if body != `{"message": "success"}` {
					t.Errorf("body = %v, want %v", body, `{"message": "success"}`)
				}
			}
		})
	}
}

func TestHTTPToolkit_Post(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	toolkit := New()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid POST with body",
			args: map[string]interface{}{
				"url":  server.URL,
				"body": `{"name": "test"}`,
			},
			wantErr: false,
		},
		{
			name: "valid POST without body",
			args: map[string]interface{}{
				"url": server.URL,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			args: map[string]interface{}{
				"body": "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolkit.httpPost(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("httpPost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Error("httpPost() result is not a map")
					return
				}

				statusCode, ok := resultMap["status_code"].(int)
				if !ok {
					t.Error("status_code not found or not int")
					return
				}

				if statusCode != http.StatusCreated {
					t.Errorf("status_code = %v, want %v", statusCode, http.StatusCreated)
				}
			}
		})
	}
}

func TestHTTPToolkit_Get_ErrorHandling(t *testing.T) {
	toolkit := New()

	// Test with invalid URL
	_, err := toolkit.httpGet(context.Background(), map[string]interface{}{
		"url": "http://invalid-domain-that-does-not-exist-12345.com",
	})

	if err == nil {
		t.Error("httpGet() should return error for invalid domain")
	}
}

func TestHTTPToolkit_Post_ErrorHandling(t *testing.T) {
	toolkit := New()

	// Test with invalid URL
	_, err := toolkit.httpPost(context.Background(), map[string]interface{}{
		"url": "http://invalid-domain-that-does-not-exist-12345.com",
	})

	if err == nil {
		t.Error("httpPost() should return error for invalid domain")
	}
}

func TestHTTPToolkit_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This will cause timeout if client timeout is very short
		// For this test, we just verify the toolkit has a timeout configured
	}))
	defer server.Close()

	toolkit := New()

	if toolkit.client.Timeout == 0 {
		t.Error("HTTP client should have timeout configured")
	}
}
