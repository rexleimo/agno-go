package media

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil payload",
			input:   nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "slice of attachments",
			input: []Attachment{
				{Type: "image", URL: "https://example.com/image.png"},
			},
			wantLen: 1,
			wantErr: false,
		},
	{
		name: "map slice",
		input: []map[string]interface{}{
			{"type": "file", "path": "/tmp/report.pdf"},
		},
		wantLen: 1,
		wantErr: false,
	},
	{
		name: "interface slice",
		input: []interface{}{
			map[string]interface{}{"type": "image", "url": "https://example.com"},
		},
		wantLen: 1,
		wantErr: false,
	},
		{
			name:    "single map",
			input:   map[string]interface{}{"type": "image", "url": "https://example.com"},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "invalid entry",
			input: []map[string]interface{}{
				{"type": "unknown"},
			},
			wantErr: true,
		},
		{
			name: "missing identifier",
			input: map[string]interface{}{
				"type": "file",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Normalize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if len(result) != tt.wantLen {
				t.Fatalf("expected length %d, got %d", tt.wantLen, len(result))
			}
			if len(result) > 0 {
				if result[0].Metadata == nil {
					t.Fatalf("expected metadata map initialised")
				}
			}
		})
	}
}

func TestNormalizeMaintainsMetadata(t *testing.T) {
	input := map[string]interface{}{
		"type": "file",
		"path": "/tmp/demo.txt",
		"metadata": map[string]interface{}{
			"source": "upload",
		},
	}
	result, err := Normalize(input)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one attachment, got %d", len(result))
	}
	if !reflect.DeepEqual(result[0].Metadata, map[string]interface{}{"source": "upload"}) {
		t.Fatalf("metadata mismatch: %+v", result[0].Metadata)
	}
}
