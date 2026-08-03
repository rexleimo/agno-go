package pandas

import (
	"context"
	"testing"
)

func TestPandasToolkit_CreateDataFrame(t *testing.T) {
	toolkit := New()

	// Test creating DataFrame
	data := map[string]interface{}{
		"name": []interface{}{"John", "Jane", "Bob"},
		"age":  []interface{}{30, 25, 35},
		"city": []interface{}{"New York", "San Francisco", "Chicago"},
	}

	result, err := toolkit.createDataFrame(context.Background(), map[string]interface{}{
		"data": data,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check shape
	shape, ok := resultMap["shape"].([]int)
	if !ok {
		t.Fatalf("Expected shape array, got: %T", resultMap["shape"])
	}

	if shape[0] != 3 || shape[1] != 3 {
		t.Errorf("Expected shape [3, 3], got %v", shape)
	}

	// Check columns
	columns, ok := resultMap["columns"].([]string)
	if !ok {
		t.Fatalf("Expected columns array, got: %T", resultMap["columns"])
	}

	if len(columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(columns))
	}

	// Check data
	dataRows, ok := resultMap["data"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected data array, got: %T", resultMap["data"])
	}

	if len(dataRows) != 3 {
		t.Errorf("Expected 3 data rows, got %d", len(dataRows))
	}
}

func TestPandasToolkit_DataFrameInfo(t *testing.T) {
	toolkit := New()

	// Create a DataFrame first
	data := map[string]interface{}{
		"name": []interface{}{"John", "Jane", "Bob"},
		"age":  []interface{}{30, 25, 35},
		"city": []interface{}{"New York", "San Francisco", "Chicago"},
	}

	dataframe, err := toolkit.createDataFrame(context.Background(), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		t.Fatalf("Failed to create DataFrame: %v", err)
	}

	// Test getting DataFrame info
	result, err := toolkit.dataframeInfo(context.Background(), map[string]interface{}{
		"dataframe": dataframe,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check shape
	shape, ok := resultMap["shape"].([]int)
	if !ok {
		t.Fatalf("Expected shape array, got: %T", resultMap["shape"])
	}

	if shape[0] != 3 || shape[1] != 3 {
		t.Errorf("Expected shape [3, 3], got %v", shape)
	}

	// Check column info
	columnInfo, ok := resultMap["column_info"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected column_info array, got: %T", resultMap["column_info"])
	}

	if len(columnInfo) != 3 {
		t.Errorf("Expected 3 column info entries, got %d", len(columnInfo))
	}
}

func TestPandasToolkit_DataFrameHead(t *testing.T) {
	toolkit := New()

	// Create a DataFrame first
	data := map[string]interface{}{
		"name": []interface{}{"John", "Jane", "Bob", "Alice", "Charlie", "Diana"},
		"age":  []interface{}{30, 25, 35, 28, 32, 29},
	}

	dataframe, err := toolkit.createDataFrame(context.Background(), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		t.Fatalf("Failed to create DataFrame: %v", err)
	}

	// Test getting head with default n
	result, err := toolkit.dataframeHead(context.Background(), map[string]interface{}{
		"dataframe": dataframe,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check shape (should be 5 rows by default)
	shape, ok := resultMap["shape"].([]int)
	if !ok {
		t.Fatalf("Expected shape array, got: %T", resultMap["shape"])
	}

	if shape[0] != 5 {
		t.Errorf("Expected 5 rows in head, got %d", shape[0])
	}

	// Test getting head with custom n
	result, err = toolkit.dataframeHead(context.Background(), map[string]interface{}{
		"dataframe": dataframe,
		"n":        2,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok = result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	shape, ok = resultMap["shape"].([]int)
	if !ok {
		t.Fatalf("Expected shape array, got: %T", resultMap["shape"])
	}

	if shape[0] != 2 {
		t.Errorf("Expected 2 rows in head, got %d", shape[0])
	}
}

func TestPandasToolkit_DataFrameFilter(t *testing.T) {
	toolkit := New()

	// Create a DataFrame first
	data := map[string]interface{}{
		"name": []interface{}{"John", "Jane", "Bob"},
		"age":  []interface{}{30, 25, 35},
		"city": []interface{}{"New York", "San Francisco", "Chicago"},
	}

	dataframe, err := toolkit.createDataFrame(context.Background(), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		t.Fatalf("Failed to create DataFrame: %v", err)
	}

	// Test filtering DataFrame
	conditions := map[string]interface{}{
		"age_filter": map[string]interface{}{
			"column":   "age",
			"operator": ">",
			"value":    28,
		},
	}

	result, err := toolkit.dataframeFilter(context.Background(), map[string]interface{}{
		"dataframe": dataframe,
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
	originalRows, ok := resultMap["original_rows"].(int)
	if !ok {
		t.Fatalf("Expected original_rows integer, got: %T", resultMap["original_rows"])
	}

	filteredRows, ok := resultMap["filtered_rows"].(int)
	if !ok {
		t.Fatalf("Expected filtered_rows integer, got: %T", resultMap["filtered_rows"])
	}

	if originalRows != 3 {
		t.Errorf("Expected 3 original rows, got %d", originalRows)
	}

	if filteredRows != 2 { // John (30) and Bob (35)
		t.Errorf("Expected 2 filtered rows, got %d", filteredRows)
	}
}

func TestPandasToolkit_DataFrameDescribe(t *testing.T) {
	toolkit := New()

	// Create a DataFrame with numeric data
	data := map[string]interface{}{
		"name": []interface{}{"John", "Jane", "Bob", "Alice"},
		"age":  []interface{}{30, 25, 35, 28},
		"salary": []interface{}{50000, 60000, 70000, 55000},
	}

	dataframe, err := toolkit.createDataFrame(context.Background(), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		t.Fatalf("Failed to create DataFrame: %v", err)
	}

	// Test describing DataFrame
	result, err := toolkit.dataframeDescribe(context.Background(), map[string]interface{}{
		"dataframe": dataframe,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check statistics
	stats, ok := resultMap["statistics"].(map[string]map[string]interface{})
	if !ok {
		t.Fatalf("Expected statistics map, got: %T", resultMap["statistics"])
	}

	// Should have statistics for age and salary columns
	if len(stats) != 2 {
		t.Errorf("Expected statistics for 2 numeric columns, got %d", len(stats))
	}

	// Check that age statistics are present
	ageStats, exists := stats["age"]
	if !exists {
		t.Error("Expected age statistics")
	}

	if ageStats["count"] != 4 {
		t.Errorf("Expected age count 4, got %v", ageStats["count"])
	}

	if ageStats["mean"] != 29.5 {
		t.Errorf("Expected age mean 29.5, got %v", ageStats["mean"])
	}
}

func TestPandasToolkit_DataFrameGroupBy(t *testing.T) {
	toolkit := New()

	// Create a DataFrame for grouping
	data := map[string]interface{}{
		"department": []interface{}{"Engineering", "Engineering", "Sales", "Sales", "Engineering"},
		"name":       []interface{}{"John", "Jane", "Bob", "Alice", "Charlie"},
		"salary":     []interface{}{50000, 60000, 70000, 55000, 65000},
	}

	dataframe, err := toolkit.createDataFrame(context.Background(), map[string]interface{}{
		"data": data,
	})
	if err != nil {
		t.Fatalf("Failed to create DataFrame: %v", err)
	}

	// Test grouping DataFrame
	result, err := toolkit.dataframeGroupBy(context.Background(), map[string]interface{}{
		"dataframe": dataframe,
		"by":       []interface{}{"department"},
		"agg": map[string]interface{}{
			"salary": "mean",
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check groups
	groups, ok := resultMap["groups"].(int)
	if !ok {
		t.Fatalf("Expected groups integer, got: %T", resultMap["groups"])
	}

	if groups != 2 { // Engineering and Sales
		t.Errorf("Expected 2 groups, got %d", groups)
	}

	// Check data
	dataRows, ok := resultMap["data"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected data array, got: %T", resultMap["data"])
	}

	if len(dataRows) != 2 {
		t.Errorf("Expected 2 result rows, got %d", len(dataRows))
	}
}

func TestPandasToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 6 {
		t.Errorf("Expected 6 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"create_dataframe", "dataframe_info", "dataframe_head", "dataframe_filter", "dataframe_describe", "dataframe_groupby"}
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