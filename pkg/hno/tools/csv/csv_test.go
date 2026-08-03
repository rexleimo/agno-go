package csv

import (
	"context"
	"os"
	"testing"
)

func TestCSVToolkit_ReadCSV(t *testing.T) {
	toolkit := New()

	// Create a temporary CSV file for testing
	tempFile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data
	testData := `name,age,city
John,30,New York
Jane,25,San Francisco
Bob,35,Chicago`
	if _, err := tempFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile.Close()

	// Test reading CSV with headers
	result, err := toolkit.readCSV(context.Background(), map[string]interface{}{
		"file_path":  tempFile.Name(),
		"has_header": true,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check headers
	headers, ok := resultMap["headers"].([]string)
	if !ok {
		t.Fatalf("Expected headers array, got: %T", resultMap["headers"])
	}

	expectedHeaders := []string{"name", "age", "city"}
	if len(headers) != len(expectedHeaders) {
		t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(headers))
	}

	// Check rows
	rows, ok := resultMap["rows"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected rows array, got: %T", resultMap["rows"])
	}

	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}

	// Check first row
	firstRow := rows[0]
	if firstRow["name"] != "John" {
		t.Errorf("Expected name 'John', got '%v'", firstRow["name"])
	}
	if firstRow["age"] != 30.0 {
		t.Errorf("Expected age 30, got '%v'", firstRow["age"])
	}
}

func TestCSVToolkit_WriteCSV(t *testing.T) {
	toolkit := New()

	// Create a temporary file path
	tempFile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	// Test writing CSV with headers
	data := []interface{}{
		[]interface{}{"John", 30, "New York"},
		[]interface{}{"Jane", 25, "San Francisco"},
	}

	result, err := toolkit.writeCSV(context.Background(), map[string]interface{}{
		"file_path": tempPath,
		"data":      data,
		"headers":   []interface{}{"name", "age", "city"},
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check result
	rows, ok := resultMap["rows"].(int)
	if !ok {
		t.Fatalf("Expected rows integer, got: %T", resultMap["rows"])
	}

	if rows != 2 {
		t.Errorf("Expected 2 rows written, got %d", rows)
	}

	// Verify the file was created and has content
	fileInfo, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Failed to stat created file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("Created file should not be empty")
	}
}

func TestCSVToolkit_AnalyzeCSV(t *testing.T) {
	toolkit := New()

	// Create a temporary CSV file for testing
	tempFile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data
	testData := `name,age,city
John,30,New York
Jane,25,San Francisco`
	if _, err := tempFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile.Close()

	// Test analyzing CSV
	result, err := toolkit.analyzeCSV(context.Background(), map[string]interface{}{
		"file_path": tempFile.Name(),
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check row count
	rowCount, ok := resultMap["row_count"].(int)
	if !ok {
		t.Fatalf("Expected row_count integer, got: %T", resultMap["row_count"])
	}

	if rowCount != 3 { // Includes header
		t.Errorf("Expected 3 rows (including header), got %d", rowCount)
	}

	// Check column count
	columnCount, ok := resultMap["column_count"].(int)
	if !ok {
		t.Fatalf("Expected column_count integer, got: %T", resultMap["column_count"])
	}

	if columnCount != 3 {
		t.Errorf("Expected 3 columns, got %d", columnCount)
	}

	// Check consistent columns
	consistent, ok := resultMap["consistent_columns"].(bool)
	if !ok {
		t.Fatalf("Expected consistent_columns boolean, got: %T", resultMap["consistent_columns"])
	}

	if !consistent {
		t.Error("Expected consistent columns to be true")
	}
}

func TestCSVToolkit_FilterCSV(t *testing.T) {
	toolkit := New()

	// Create a temporary CSV file for testing
	tempFile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data
	testData := `name,age,city
John,30,New York
Jane,25,San Francisco
Bob,35,Chicago`
	if _, err := tempFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile.Close()

	// Test filtering CSV
	conditions := map[string]interface{}{
		"age_filter": map[string]interface{}{
			"column":   "age",
			"operator": ">",
			"value":    28,
		},
	}

	result, err := toolkit.filterCSV(context.Background(), map[string]interface{}{
		"file_path":  tempFile.Name(),
		"conditions": conditions,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check counts
	originalCount, ok := resultMap["original_count"].(int)
	if !ok {
		t.Fatalf("Expected original_count integer, got: %T", resultMap["original_count"])
	}

	filteredCount, ok := resultMap["filtered_count"].(int)
	if !ok {
		t.Fatalf("Expected filtered_count integer, got: %T", resultMap["filtered_count"])
	}

	if originalCount != 3 {
		t.Errorf("Expected 3 original rows, got %d", originalCount)
	}

	if filteredCount != 2 { // John (30) and Bob (35)
		t.Errorf("Expected 2 filtered rows, got %d", filteredCount)
	}
}

func TestCSVToolkit_ReadCSV_NoHeader(t *testing.T) {
	toolkit := New()

	// Create a temporary CSV file for testing
	tempFile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data without headers
	testData := `John,30,New York
Jane,25,San Francisco`
	if _, err := tempFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tempFile.Close()

	// Test reading CSV without headers
	result, err := toolkit.readCSV(context.Background(), map[string]interface{}{
		"file_path":  tempFile.Name(),
		"has_header": false,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that default headers were generated
	headers, ok := resultMap["headers"].([]string)
	if !ok {
		t.Fatalf("Expected headers array, got: %T", resultMap["headers"])
	}

	if len(headers) != 3 {
		t.Errorf("Expected 3 default headers, got %d", len(headers))
	}

	expectedHeaders := []string{"column_1", "column_2", "column_3"}
	for i, header := range headers {
		if header != expectedHeaders[i] {
			t.Errorf("Expected header '%s', got '%s'", expectedHeaders[i], header)
		}
	}
}

func TestCSVToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 4 {
		t.Errorf("Expected 4 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"read_csv", "write_csv", "analyze_csv", "filter_csv"}
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