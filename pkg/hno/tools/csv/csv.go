package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// CSVToolkit provides CSV file manipulation capabilities
// This is a basic implementation that can be extended with more advanced CSV operations

// CSVToolkit provides CSV file manipulation
type CSVToolkit struct {
	*toolkit.BaseToolkit
}

// New creates a new CSV toolkit
func New() *CSVToolkit {
	t := &CSVToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("csv"),
	}

	// Register CSV read function
	t.RegisterFunction(&toolkit.Function{
		Name:        "read_csv",
		Description: "Read a CSV file and return its contents as structured data",
		Parameters: map[string]toolkit.Parameter{
			"file_path": {
				Type:        "string",
				Description: "Path to the CSV file",
				Required:    true,
			},
			"has_header": {
				Type:        "boolean",
				Description: "Whether the CSV file has a header row (default: true)",
				Required:    false,
				Default:     true,
			},
		},
		Handler: t.readCSV,
	})

	// Register CSV write function
	t.RegisterFunction(&toolkit.Function{
		Name:        "write_csv",
		Description: "Write data to a CSV file",
		Parameters: map[string]toolkit.Parameter{
			"file_path": {
				Type:        "string",
				Description: "Path to the CSV file to create",
				Required:    true,
			},
			"data": {
				Type:        "array",
				Description: "Array of rows to write (each row is an array of values)",
				Required:    true,
			},
			"headers": {
				Type:        "array",
				Description: "Optional header row (array of column names)",
				Required:    false,
			},
		},
		Handler: t.writeCSV,
	})

	// Register CSV analyze function
	t.RegisterFunction(&toolkit.Function{
		Name:        "analyze_csv",
		Description: "Analyze a CSV file and return statistics about its structure",
		Parameters: map[string]toolkit.Parameter{
			"file_path": {
				Type:        "string",
				Description: "Path to the CSV file",
				Required:    true,
			},
		},
		Handler: t.analyzeCSV,
	})

	// Register CSV filter function
	t.RegisterFunction(&toolkit.Function{
		Name:        "filter_csv",
		Description: "Filter CSV data based on conditions",
		Parameters: map[string]toolkit.Parameter{
			"file_path": {
				Type:        "string",
				Description: "Path to the CSV file",
				Required:    true,
			},
			"conditions": {
				Type:        "object",
				Description: "Filter conditions (e.g., {\"column\": \"age\", \"operator\": \">\", \"value\": 30})",
				Required:    true,
			},
		},
		Handler: t.filterCSV,
	})

	return t
}

// readCSV reads a CSV file and returns its contents
func (c *CSVToolkit) readCSV(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path must be a string")
	}

	hasHeader := true
	if hasHeaderArg, ok := args["has_header"].(bool); ok {
		hasHeader = hasHeaderArg
	}

	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// Read CSV data
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	if len(allRows) == 0 {
		return map[string]interface{}{
			"file_path": filePath,
			"rows":      []interface{}{},
			"headers":   []string{},
			"count":     0,
		}, nil
	}

	var headers []string
	var dataRows [][]string

	if hasHeader && len(allRows) > 0 {
		headers = allRows[0]
		dataRows = allRows[1:]
	} else {
		// Generate default headers
		headers = make([]string, len(allRows[0]))
		for i := range headers {
			headers[i] = fmt.Sprintf("column_%d", i+1)
		}
		dataRows = allRows
	}

	// Convert data to structured format
	structuredData := make([]map[string]interface{}, len(dataRows))
	for i, row := range dataRows {
		rowData := make(map[string]interface{})
		for j, value := range row {
			if j < len(headers) {
				// Try to convert to number if possible
				if num, err := strconv.ParseFloat(value, 64); err == nil {
					rowData[headers[j]] = num
				} else {
					rowData[headers[j]] = value
				}
			}
		}
		structuredData[i] = rowData
	}

	return map[string]interface{}{
		"file_path": filePath,
		"headers":   headers,
		"rows":      structuredData,
		"count":     len(structuredData),
	}, nil
}

// writeCSV writes data to a CSV file
func (c *CSVToolkit) writeCSV(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path must be a string")
	}

	dataArg, ok := args["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an array")
	}

	var headers []string
	if headersArg, ok := args["headers"].([]interface{}); ok {
		headers = make([]string, len(headersArg))
		for i, h := range headersArg {
			headers[i] = fmt.Sprintf("%v", h)
		}
	}

	// Convert data to string rows
	rows := make([][]string, len(dataArg))
	for i, rowData := range dataArg {
		row, ok := rowData.([]interface{})
		if !ok {
			return nil, fmt.Errorf("data row %d must be an array", i)
		}

		rowStrings := make([]string, len(row))
		for j, value := range row {
			rowStrings[j] = fmt.Sprintf("%v", value)
		}
		rows[i] = rowStrings
	}

	// Create the CSV file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Write headers if provided
	if len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return nil, fmt.Errorf("failed to write headers: %w", err)
		}
	}

	// Write data rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	return map[string]interface{}{
		"file_path": filePath,
		"rows":      len(rows),
		"headers":   headers,
		"columns":   len(headers),
		"message":   "CSV file created successfully",
	}, nil
}

// analyzeCSV analyzes a CSV file and returns statistics
func (c *CSVToolkit) analyzeCSV(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path must be a string")
	}

	// Read the file first
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	if len(allRows) == 0 {
		return map[string]interface{}{
			"file_path": filePath,
			"row_count": 0,
			"column_count": 0,
			"empty": true,
		}, nil
	}

	// Calculate statistics
	rowCount := len(allRows)
	columnCount := len(allRows[0])

	// Check for consistent column count
	consistentColumns := true
	for _, row := range allRows {
		if len(row) != columnCount {
			consistentColumns = false
			if len(row) > columnCount {
				columnCount = len(row)
			}
		}
	}

	// Sample first few rows
	sampleRows := make([]interface{}, 0)
	sampleCount := 3
	if rowCount < sampleCount {
		sampleCount = rowCount
	}
	for i := 0; i < sampleCount; i++ {
		sampleRows = append(sampleRows, allRows[i])
	}

	return map[string]interface{}{
		"file_path":         filePath,
		"row_count":         rowCount,
		"column_count":      columnCount,
		"consistent_columns": consistentColumns,
		"sample_rows":       sampleRows,
		"has_header":        rowCount > 0,
	}, nil
}

// filterCSV filters CSV data based on conditions
func (c *CSVToolkit) filterCSV(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path must be a string")
	}

	conditions, ok := args["conditions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("conditions must be an object")
	}

	// Read the CSV file first
	readResult, err := c.readCSV(ctx, map[string]interface{}{
		"file_path":  filePath,
		"has_header": true,
	})
	if err != nil {
		return nil, err
	}

	readData := readResult.(map[string]interface{})
	rows := readData["rows"].([]map[string]interface{})
	headers := readData["headers"].([]string)

	// Apply filtering
	filteredRows := make([]map[string]interface{}, 0)

	for _, row := range rows {
		if c.matchesConditions(row, conditions) {
			filteredRows = append(filteredRows, row)
		}
	}

	return map[string]interface{}{
		"file_path":       filePath,
		"original_count":  len(rows),
		"filtered_count":  len(filteredRows),
		"filtered_rows":   filteredRows,
		"headers":         headers,
		"conditions":      conditions,
	}, nil
}

// matchesConditions checks if a row matches the given conditions
func (c *CSVToolkit) matchesConditions(row map[string]interface{}, conditions map[string]interface{}) bool {
	for _, condition := range conditions {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		column, ok := conditionMap["column"].(string)
		if !ok {
			continue
		}

		operator, ok := conditionMap["operator"].(string)
		if !ok {
			continue
		}

		value := conditionMap["value"]

		rowValue, exists := row[column]
		if !exists {
			return false
		}

		if !c.compareValues(rowValue, value, operator) {
			return false
		}
	}

	return true
}

// compareValues compares two values based on the operator
func (c *CSVToolkit) compareValues(a, b interface{}, operator string) bool {
	// Try to convert to numbers for numeric comparison
	aNum, aIsNum := c.toNumber(a)
	bNum, bIsNum := c.toNumber(b)

	if aIsNum && bIsNum {
		switch operator {
		case "==":
			return aNum == bNum
		case "!=":
			return aNum != bNum
		case ">":
			return aNum > bNum
		case ">=":
			return aNum >= bNum
		case "<":
			return aNum < bNum
		case "<=":
			return aNum <= bNum
		}
	}

	// Fall back to string comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	switch operator {
	case "==":
		return aStr == bStr
	case "!=":
		return aStr != bStr
	case "contains":
		return strings.Contains(aStr, bStr)
	case "starts_with":
		return strings.HasPrefix(aStr, bStr)
	case "ends_with":
		return strings.HasSuffix(aStr, bStr)
	}

	return false
}

// toNumber attempts to convert a value to a number
func (c *CSVToolkit) toNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}