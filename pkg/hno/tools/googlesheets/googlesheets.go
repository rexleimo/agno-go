package googlesheets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GoogleSheetsToolkit 提供 Google Sheets 操作工具
// GoogleSheetsToolkit provides Google Sheets operation tools
type GoogleSheetsToolkit struct {
	*toolkit.BaseToolkit
	service *sheets.Service
	ctx     context.Context
}

// Config Google Sheets 工具配置
// Config is the Google Sheets tool configuration
type Config struct {
	// CredentialsJSON 服务账号 JSON 凭证（可以是文件路径或 JSON 字符串）
	// CredentialsJSON is the service account JSON credentials (can be file path or JSON string)
	CredentialsJSON string

	// CredentialsFile 服务账号 JSON 文件路径（已弃用，使用 CredentialsJSON）
	// CredentialsFile is the service account JSON file path (deprecated, use CredentialsJSON)
	CredentialsFile string
}

// New 创建一个新的 Google Sheets 工具包
// New creates a new Google Sheets toolkit
func New(config Config) (*GoogleSheetsToolkit, error) {
	ctx := context.Background()

	// 获取凭证
	// Get credentials
	var credsJSON []byte
	var err error

	if config.CredentialsJSON != "" {
		// 尝试作为文件路径读取
		// Try to read as file path
		if _, statErr := os.Stat(config.CredentialsJSON); statErr == nil {
			credsJSON, err = os.ReadFile(config.CredentialsJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to read credentials file: %w", err)
			}
		} else {
			// 作为 JSON 字符串使用
			// Use as JSON string
			credsJSON = []byte(config.CredentialsJSON)
		}
	} else if config.CredentialsFile != "" {
		// 向后兼容
		// Backward compatibility
		credsJSON, err = os.ReadFile(config.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read credentials file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("credentials not provided")
	}

	// 验证 JSON 格式
	// Validate JSON format
	var jsonTest map[string]interface{}
	if err := json.Unmarshal(credsJSON, &jsonTest); err != nil {
		return nil, fmt.Errorf("invalid credentials JSON: %w", err)
	}

	// 创建 Google Sheets 客户端
	// Create Google Sheets client
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	service, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	toolkit := &GoogleSheetsToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("google_sheets"),
		service:     service,
		ctx:         ctx,
	}

	// 注册函数
	// Register functions
	toolkit.registerFunctions()

	return toolkit, nil
}

// registerFunctions 注册所有工具函数
// registerFunctions registers all tool functions
func (t *GoogleSheetsToolkit) registerFunctions() {
	// read_range - 读取指定范围的数据
	// read_range - Read data from a specified range
	t.RegisterFunction(&toolkit.Function{
		Name:        "read_range",
		Description: "从 Google Sheets 中读取指定范围的数据",
		Parameters: map[string]toolkit.Parameter{
			"spreadsheet_id": {
				Type:        "string",
				Description: "电子表格 ID（可从 URL 中获取）",
				Required:    true,
			},
			"range": {
				Type:        "string",
				Description: "要读取的范围，例如 'Sheet1!A1:D10'",
				Required:    true,
			},
		},
		Handler: t.readRange,
	})

	// write_range - 写入数据到指定范围
	// write_range - Write data to a specified range
	t.RegisterFunction(&toolkit.Function{
		Name:        "write_range",
		Description: "将数据写入 Google Sheets 的指定范围",
		Parameters: map[string]toolkit.Parameter{
			"spreadsheet_id": {
				Type:        "string",
				Description: "电子表格 ID",
				Required:    true,
			},
			"range": {
				Type:        "string",
				Description: "要写入的范围，例如 'Sheet1!A1'",
				Required:    true,
			},
			"values": {
				Type:        "array",
				Description: "要写入的数据（二维数组）",
				Required:    true,
			},
		},
		Handler: t.writeRange,
	})

	// append_rows - 追加行到表格
	// append_rows - Append rows to the sheet
	t.RegisterFunction(&toolkit.Function{
		Name:        "append_rows",
		Description: "向 Google Sheets 追加新行",
		Parameters: map[string]toolkit.Parameter{
			"spreadsheet_id": {
				Type:        "string",
				Description: "电子表格 ID",
				Required:    true,
			},
			"range": {
				Type:        "string",
				Description: "追加范围，例如 'Sheet1!A:D'",
				Required:    true,
			},
			"values": {
				Type:        "array",
				Description: "要追加的数据（二维数组）",
				Required:    true,
			},
		},
		Handler: t.appendRows,
	})
}

// readRange 读取指定范围的数据
// readRange reads data from a specified range
func (t *GoogleSheetsToolkit) readRange(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	spreadsheetID, ok := args["spreadsheet_id"].(string)
	if !ok || spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet_id is required")
	}

	rangeStr, ok := args["range"].(string)
	if !ok || rangeStr == "" {
		return nil, fmt.Errorf("range is required")
	}

	// 调用 Google Sheets API
	// Call Google Sheets API
	resp, err := t.service.Spreadsheets.Values.Get(spreadsheetID, rangeStr).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read range: %w", err)
	}

	return map[string]interface{}{
		"range":  resp.Range,
		"values": resp.Values,
		"rows":   len(resp.Values),
	}, nil
}

// writeRange 写入数据到指定范围
// writeRange writes data to a specified range
func (t *GoogleSheetsToolkit) writeRange(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	spreadsheetID, ok := args["spreadsheet_id"].(string)
	if !ok || spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet_id is required")
	}

	rangeStr, ok := args["range"].(string)
	if !ok || rangeStr == "" {
		return nil, fmt.Errorf("range is required")
	}

	valuesArg, ok := args["values"]
	if !ok {
		return nil, fmt.Errorf("values is required")
	}

	// 转换 values 为正确的格式
	// Convert values to correct format
	values, err := convertToValueRange(valuesArg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert values: %w", err)
	}

	// 创建 ValueRange
	// Create ValueRange
	vr := &sheets.ValueRange{
		Values: values,
	}

	// 调用 Google Sheets API
	// Call Google Sheets API
	resp, err := t.service.Spreadsheets.Values.Update(spreadsheetID, rangeStr, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to write range: %w", err)
	}

	return map[string]interface{}{
		"updated_range":   resp.UpdatedRange,
		"updated_rows":    resp.UpdatedRows,
		"updated_cells":   resp.UpdatedCells,
		"updated_columns": resp.UpdatedColumns,
	}, nil
}

// appendRows 追加行到表格
// appendRows appends rows to the sheet
func (t *GoogleSheetsToolkit) appendRows(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	spreadsheetID, ok := args["spreadsheet_id"].(string)
	if !ok || spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet_id is required")
	}

	rangeStr, ok := args["range"].(string)
	if !ok || rangeStr == "" {
		return nil, fmt.Errorf("range is required")
	}

	valuesArg, ok := args["values"]
	if !ok {
		return nil, fmt.Errorf("values is required")
	}

	// 转换 values 为正确的格式
	// Convert values to correct format
	values, err := convertToValueRange(valuesArg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert values: %w", err)
	}

	// 创建 ValueRange
	// Create ValueRange
	vr := &sheets.ValueRange{
		Values: values,
	}

	// 调用 Google Sheets API
	// Call Google Sheets API
	resp, err := t.service.Spreadsheets.Values.Append(spreadsheetID, rangeStr, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to append rows: %w", err)
	}

	return map[string]interface{}{
		"updated_range":   resp.Updates.UpdatedRange,
		"updated_rows":    resp.Updates.UpdatedRows,
		"updated_cells":   resp.Updates.UpdatedCells,
		"updated_columns": resp.Updates.UpdatedColumns,
	}, nil
}

// convertToValueRange 转换输入数据为 Google Sheets 值格式
// convertToValueRange converts input data to Google Sheets value format
func convertToValueRange(input interface{}) ([][]interface{}, error) {
	// 尝试直接转换
	// Try direct conversion
	if values, ok := input.([][]interface{}); ok {
		return values, nil
	}

	// 尝试从 JSON 数组转换
	// Try converting from JSON array
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var values [][]interface{}
	if err := json.Unmarshal(inputBytes, &values); err != nil {
		return nil, err
	}

	return values, nil
}
