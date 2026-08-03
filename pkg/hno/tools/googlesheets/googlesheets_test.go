package googlesheets

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_CredentialsJSON 测试通过 JSON 字符串创建工具包
// TestConfig_CredentialsJSON tests creating toolkit with JSON string credentials
func TestConfig_CredentialsJSON(t *testing.T) {
	// 跳过集成测试，因为需要真实的 Google 凭证
	// Skip integration test as it requires real Google credentials
	t.Skip("Requires real Google credentials")

	credJSON := `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "key123",
		"private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----\n",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs"
	}`

	config := Config{
		CredentialsJSON: credJSON,
	}

	toolkit, err := New(config)
	assert.NoError(t, err)
	assert.NotNil(t, toolkit)
	assert.Equal(t, "google_sheets", toolkit.Name())
}

// TestConfig_CredentialsFile 测试通过文件路径创建工具包
// TestConfig_CredentialsFile tests creating toolkit with credentials file path
func TestConfig_CredentialsFile(t *testing.T) {
	// 创建临时凭证文件
	// Create temporary credentials file
	tmpFile, err := os.CreateTemp("", "credentials-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	credJSON := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  "test-project",
		"private_key_id":              "key123",
		"private_key":                 "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----\n",
		"client_email":                "test@test-project.iam.gserviceaccount.com",
		"client_id":                   "123456789",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
	}

	data, err := json.Marshal(credJSON)
	require.NoError(t, err)

	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	// 跳过集成测试
	// Skip integration test
	t.Skip("Requires real Google credentials")

	config := Config{
		CredentialsJSON: tmpFile.Name(),
	}

	toolkit, err := New(config)
	assert.NoError(t, err)
	assert.NotNil(t, toolkit)
}

// TestConfig_NoCredentials 测试未提供凭证时的错误
// TestConfig_NoCredentials tests error when no credentials provided
func TestConfig_NoCredentials(t *testing.T) {
	config := Config{}

	toolkit, err := New(config)
	assert.Error(t, err)
	assert.Nil(t, toolkit)
	assert.Contains(t, err.Error(), "credentials not provided")
}

// TestConfig_InvalidJSON 测试无效的 JSON 凭证
// TestConfig_InvalidJSON tests invalid JSON credentials
func TestConfig_InvalidJSON(t *testing.T) {
	config := Config{
		CredentialsJSON: "{invalid json}",
	}

	toolkit, err := New(config)
	assert.Error(t, err)
	assert.Nil(t, toolkit)
	assert.Contains(t, err.Error(), "invalid credentials JSON")
}

// TestConfig_BackwardCompatibility 测试向后兼容性（CredentialsFile）
// TestConfig_BackwardCompatibility tests backward compatibility with CredentialsFile
func TestConfig_BackwardCompatibility(t *testing.T) {
	// 创建临时凭证文件
	// Create temporary credentials file
	tmpFile, err := os.CreateTemp("", "credentials-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	credJSON := map[string]interface{}{
		"type":       "service_account",
		"project_id": "test-project",
	}

	data, err := json.Marshal(credJSON)
	require.NoError(t, err)

	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	// 跳过集成测试
	// Skip integration test
	t.Skip("Requires real Google credentials")

	config := Config{
		CredentialsFile: tmpFile.Name(),
	}

	toolkit, err := New(config)
	assert.NoError(t, err)
	assert.NotNil(t, toolkit)
}

// TestFunctionRegistration 测试函数注册
// TestFunctionRegistration tests function registration
func TestFunctionRegistration(t *testing.T) {
	// 跳过集成测试
	// Skip integration test
	t.Skip("Requires real Google credentials")

	credJSON := `{"type": "service_account", "project_id": "test"}`

	config := Config{
		CredentialsJSON: credJSON,
	}

	toolkit, err := New(config)
	require.NoError(t, err)

	// 验证函数已注册
	// Verify functions are registered
	functions := toolkit.Functions()
	assert.Len(t, functions, 3)

	functionNames := make(map[string]bool)
	for _, fn := range functions {
		functionNames[fn.Name] = true
	}

	assert.True(t, functionNames["read_range"])
	assert.True(t, functionNames["write_range"])
	assert.True(t, functionNames["append_rows"])
}

// TestReadRange_MissingParams 测试 readRange 缺少参数
// TestReadRange_MissingParams tests readRange with missing parameters
func TestReadRange_MissingParams(t *testing.T) {
	toolkit := &GoogleSheetsToolkit{
		ctx: context.Background(),
	}

	ctx := context.Background()

	// 缺少 spreadsheet_id
	// Missing spreadsheet_id
	result, err := toolkit.readRange(ctx, map[string]interface{}{
		"range": "Sheet1!A1:B2",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "spreadsheet_id is required")

	// 缺少 range
	// Missing range
	result, err = toolkit.readRange(ctx, map[string]interface{}{
		"spreadsheet_id": "test-id",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "range is required")
}

// TestWriteRange_MissingParams 测试 writeRange 缺少参数
// TestWriteRange_MissingParams tests writeRange with missing parameters
func TestWriteRange_MissingParams(t *testing.T) {
	toolkit := &GoogleSheetsToolkit{
		ctx: context.Background(),
	}

	ctx := context.Background()

	// 缺少 spreadsheet_id
	// Missing spreadsheet_id
	result, err := toolkit.writeRange(ctx, map[string]interface{}{
		"range":  "Sheet1!A1",
		"values": [][]interface{}{{"test"}},
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "spreadsheet_id is required")

	// 缺少 range
	// Missing range
	result, err = toolkit.writeRange(ctx, map[string]interface{}{
		"spreadsheet_id": "test-id",
		"values":         [][]interface{}{{"test"}},
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "range is required")

	// 缺少 values
	// Missing values
	result, err = toolkit.writeRange(ctx, map[string]interface{}{
		"spreadsheet_id": "test-id",
		"range":          "Sheet1!A1",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "values is required")
}

// TestAppendRows_MissingParams 测试 appendRows 缺少参数
// TestAppendRows_MissingParams tests appendRows with missing parameters
func TestAppendRows_MissingParams(t *testing.T) {
	toolkit := &GoogleSheetsToolkit{
		ctx: context.Background(),
	}

	ctx := context.Background()

	// 缺少 spreadsheet_id
	// Missing spreadsheet_id
	result, err := toolkit.appendRows(ctx, map[string]interface{}{
		"range":  "Sheet1!A:B",
		"values": [][]interface{}{{"test"}},
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "spreadsheet_id is required")

	// 缺少 range
	// Missing range
	result, err = toolkit.appendRows(ctx, map[string]interface{}{
		"spreadsheet_id": "test-id",
		"values":         [][]interface{}{{"test"}},
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "range is required")

	// 缺少 values
	// Missing values
	result, err = toolkit.appendRows(ctx, map[string]interface{}{
		"spreadsheet_id": "test-id",
		"range":          "Sheet1!A:B",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "values is required")
}

// TestConvertToValueRange 测试数据格式转换
// TestConvertToValueRange tests value format conversion
func TestConvertToValueRange(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected [][]interface{}
		wantErr  bool
	}{
		{
			name: "直接二维数组",
			input: [][]interface{}{
				{"A1", "B1"},
				{"A2", "B2"},
			},
			expected: [][]interface{}{
				{"A1", "B1"},
				{"A2", "B2"},
			},
			wantErr: false,
		},
		{
			name: "JSON 数组",
			input: []interface{}{
				[]interface{}{"A1", "B1"},
				[]interface{}{"A2", "B2"},
			},
			expected: [][]interface{}{
				{"A1", "B1"},
				{"A2", "B2"},
			},
			wantErr: false,
		},
		{
			name:     "无效输入",
			input:    "invalid",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToValueRange(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestGoogleSheetsToolkit_Name 测试工具包名称
// TestGoogleSheetsToolkit_Name tests toolkit name
func TestGoogleSheetsToolkit_Name(t *testing.T) {
	// 跳过集成测试
	// Skip integration test
	t.Skip("Requires real Google credentials")

	credJSON := `{"type": "service_account", "project_id": "test"}`

	config := Config{
		CredentialsJSON: credJSON,
	}

	toolkit, err := New(config)
	require.NoError(t, err)

	assert.Equal(t, "google_sheets", toolkit.Name())
}
