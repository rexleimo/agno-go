package pandas

import (
	"context"
	"fmt"
	"sort"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// PandasToolkit provides data manipulation capabilities similar to Python pandas
// This is a simplified implementation that provides basic DataFrame-like operations

// DataFrame represents a simplified DataFrame structure
type DataFrame struct {
	Columns []string
	Data    []map[string]interface{}
}

// PandasToolkit provides pandas-like data manipulation
type PandasToolkit struct {
	*toolkit.BaseToolkit
}

// New creates a new Pandas toolkit
func New() *PandasToolkit {
	t := &PandasToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("pandas"),
	}

	// Register DataFrame creation function
	t.RegisterFunction(&toolkit.Function{
		Name:        "create_dataframe",
		Description: "Create a DataFrame from structured data",
		Parameters: map[string]toolkit.Parameter{
			"data": {
				Type:        "object",
				Description: "Data in format {\"column1\": [values], \"column2\": [values]}",
				Required:    true,
			},
		},
		Handler: t.createDataFrame,
	})

	// Register DataFrame info function
	t.RegisterFunction(&toolkit.Function{
		Name:        "dataframe_info",
		Description: "Get information about a DataFrame",
		Parameters: map[string]toolkit.Parameter{
			"dataframe": {
				Type:        "object",
				Description: "DataFrame object",
				Required:    true,
			},
		},
		Handler: t.dataframeInfo,
	})

	// Register DataFrame head function
	t.RegisterFunction(&toolkit.Function{
		Name:        "dataframe_head",
		Description: "Get the first n rows of a DataFrame",
		Parameters: map[string]toolkit.Parameter{
			"dataframe": {
				Type:        "object",
				Description: "DataFrame object",
				Required:    true,
			},
			"n": {
				Type:        "integer",
				Description: "Number of rows to return (default: 5)",
				Required:    false,
				Default:     5,
			},
		},
		Handler: t.dataframeHead,
	})

	// Register DataFrame filter function
	t.RegisterFunction(&toolkit.Function{
		Name:        "dataframe_filter",
		Description: "Filter DataFrame rows based on conditions",
		Parameters: map[string]toolkit.Parameter{
			"dataframe": {
				Type:        "object",
				Description: "DataFrame object",
				Required:    true,
			},
			"conditions": {
				Type:        "object",
				Description: "Filter conditions (e.g., {\"column\": \"age\", \"operator\": \">\", \"value\": 30})",
				Required:    true,
			},
		},
		Handler: t.dataframeFilter,
	})

	// Register DataFrame describe function
	t.RegisterFunction(&toolkit.Function{
		Name:        "dataframe_describe",
		Description: "Generate descriptive statistics for numerical columns",
		Parameters: map[string]toolkit.Parameter{
			"dataframe": {
				Type:        "object",
				Description: "DataFrame object",
				Required:    true,
			},
		},
		Handler: t.dataframeDescribe,
	})

	// Register DataFrame groupby function
	t.RegisterFunction(&toolkit.Function{
		Name:        "dataframe_groupby",
		Description: "Group DataFrame by specified columns and apply aggregation",
		Parameters: map[string]toolkit.Parameter{
			"dataframe": {
				Type:        "object",
				Description: "DataFrame object",
				Required:    true,
			},
			"by": {
				Type:        "array",
				Description: "Columns to group by",
				Required:    true,
			},
			"agg": {
				Type:        "object",
				Description: "Aggregation functions (e.g., {\"age\": \"mean\", \"salary\": \"sum\"})",
				Required:    true,
			},
		},
		Handler: t.dataframeGroupBy,
	})

	return t
}

// createDataFrame creates a DataFrame from structured data
func (p *PandasToolkit) createDataFrame(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dataArg, ok := args["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data must be an object")
	}

	// Extract columns and data
	columns := make([]string, 0, len(dataArg))
	for column := range dataArg {
		columns = append(columns, column)
	}

	// Determine the number of rows
	var rowCount int
	for _, values := range dataArg {
		valueList, ok := values.([]interface{})
		if ok && len(valueList) > rowCount {
			rowCount = len(valueList)
		}
	}

	// Create DataFrame data
	data := make([]map[string]interface{}, rowCount)
	for i := 0; i < rowCount; i++ {
		row := make(map[string]interface{})
		for _, column := range columns {
			values, ok := dataArg[column].([]interface{})
			if ok && i < len(values) {
				row[column] = values[i]
			} else {
				row[column] = nil
			}
		}
		data[i] = row
	}

	return map[string]interface{}{
		"columns": columns,
		"data":    data,
		"shape":   []int{rowCount, len(columns)},
	}, nil
}

// dataframeInfo provides information about a DataFrame
func (p *PandasToolkit) dataframeInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dataframe, ok := args["dataframe"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dataframe must be an object")
	}

	columns, ok := dataframe["columns"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	data, ok := dataframe["data"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	// Calculate column statistics
	columnInfo := make([]map[string]interface{}, len(columns))
	for i, column := range columns {
		info := map[string]interface{}{
			"column": column,
			"non_null_count": 0,
			"null_count":     0,
			"dtype":          "object", // Default to object type
		}

		// Check if column contains numeric data
		isNumeric := true
		for _, row := range data {
			value := row[column]
			if value != nil {
				info["non_null_count"] = info["non_null_count"].(int) + 1
				// Check if value is numeric
				switch value.(type) {
				case int, int32, int64, float32, float64:
					// Numeric value
				default:
					isNumeric = false
				}
			} else {
				info["null_count"] = info["null_count"].(int) + 1
			}
		}

		if isNumeric && info["non_null_count"].(int) > 0 {
			info["dtype"] = "numeric"
		}

		columnInfo[i] = info
	}

	return map[string]interface{}{
		"shape":        []int{len(data), len(columns)},
		"columns":      len(columns),
		"rows":         len(data),
		"memory_usage": "N/A", // Simplified implementation
		"column_info":  columnInfo,
	}, nil
}

// dataframeHead returns the first n rows of a DataFrame
func (p *PandasToolkit) dataframeHead(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dataframe, ok := args["dataframe"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dataframe must be an object")
	}

	n := 5
	if nArg, ok := args["n"].(float64); ok {
		n = int(nArg)
	} else if nArg, ok := args["n"].(int); ok {
		n = nArg
	}

	data, ok := dataframe["data"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	// Get first n rows
	if n > len(data) {
		n = len(data)
	}

	headData := make([]map[string]interface{}, n)
	for i := 0; i < n; i++ {
		headData[i] = data[i]
	}

	return map[string]interface{}{
		"columns": dataframe["columns"],
		"data":    headData,
		"shape":   []int{n, len(dataframe["columns"].([]string))},
	}, nil
}

// dataframeFilter filters DataFrame rows based on conditions
func (p *PandasToolkit) dataframeFilter(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dataframe, ok := args["dataframe"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dataframe must be an object")
	}

	conditions, ok := args["conditions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("conditions must be an object")
	}

	data, ok := dataframe["data"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	// Apply filtering
	filteredData := make([]map[string]interface{}, 0)

	for _, row := range data {
		if p.matchesConditions(row, conditions) {
			filteredData = append(filteredData, row)
		}
	}

	return map[string]interface{}{
		"columns":        dataframe["columns"],
		"data":          filteredData,
		"shape":         []int{len(filteredData), len(dataframe["columns"].([]string))},
		"original_rows": len(data),
		"filtered_rows": len(filteredData),
	}, nil
}

// dataframeDescribe generates descriptive statistics for numerical columns
func (p *PandasToolkit) dataframeDescribe(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dataframe, ok := args["dataframe"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dataframe must be an object")
	}

	data, ok := dataframe["data"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	columns, ok := dataframe["columns"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	// Find numeric columns and calculate statistics
	stats := make(map[string]map[string]interface{})

	for _, column := range columns {
		var numericValues []float64

		// Collect numeric values for this column
		for _, row := range data {
			value := row[column]
			if num, ok := p.toFloat64(value); ok {
				numericValues = append(numericValues, num)
			}
		}

		if len(numericValues) > 0 {
			columnStats := make(map[string]interface{})
			columnStats["count"] = len(numericValues)
			columnStats["mean"] = p.mean(numericValues)
			columnStats["std"] = p.std(numericValues)
			columnStats["min"] = p.min(numericValues)
			columnStats["max"] = p.max(numericValues)
			columnStats["25%"] = p.percentile(numericValues, 25)
			columnStats["50%"] = p.percentile(numericValues, 50)
			columnStats["75%"] = p.percentile(numericValues, 75)

			stats[column] = columnStats
		}
	}

	return map[string]interface{}{
		"statistics": stats,
		"numeric_columns": len(stats),
	}, nil
}

// dataframeGroupBy groups DataFrame by specified columns and applies aggregation
func (p *PandasToolkit) dataframeGroupBy(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dataframe, ok := args["dataframe"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dataframe must be an object")
	}

	byArg, ok := args["by"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("by must be an array")
	}

	aggArg, ok := args["agg"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("agg must be an object")
	}

	data, ok := dataframe["data"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid dataframe structure")
	}

	// Convert by to string slice
	by := make([]string, len(byArg))
	for i, b := range byArg {
		by[i] = fmt.Sprintf("%v", b)
	}

	// Group data
	groups := make(map[string][]map[string]interface{})

	for _, row := range data {
		groupKey := ""
		for _, column := range by {
			if groupKey != "" {
				groupKey += "_"
			}
			groupKey += fmt.Sprintf("%v", row[column])
		}

		groups[groupKey] = append(groups[groupKey], row)
	}

	// Apply aggregations
	result := make([]map[string]interface{}, 0)

	for _, groupRows := range groups {
		resultRow := make(map[string]interface{})

		// Add group by columns
		for _, column := range by {
			if len(groupRows) > 0 {
				resultRow[column] = groupRows[0][column]
			}
		}

		// Apply aggregations
		for aggColumn, aggFunc := range aggArg {
			aggFuncStr := fmt.Sprintf("%v", aggFunc)
			var values []float64

			// Collect numeric values for aggregation
			for _, row := range groupRows {
				if value, ok := p.toFloat64(row[aggColumn]); ok {
					values = append(values, value)
				}
			}

			if len(values) > 0 {
				switch aggFuncStr {
				case "sum":
					resultRow[aggColumn+"_sum"] = p.sum(values)
				case "mean":
					resultRow[aggColumn+"_mean"] = p.mean(values)
				case "count":
					resultRow[aggColumn+"_count"] = len(values)
				case "min":
					resultRow[aggColumn+"_min"] = p.min(values)
				case "max":
					resultRow[aggColumn+"_max"] = p.max(values)
				}
			}
		}

		result = append(result, resultRow)
	}

	// Build result columns
	resultColumns := make([]string, 0)
	resultColumns = append(resultColumns, by...)
	for aggColumn := range aggArg {
		resultColumns = append(resultColumns, aggColumn+"_sum", aggColumn+"_mean", aggColumn+"_count", aggColumn+"_min", aggColumn+"_max")
	}

	return map[string]interface{}{
		"columns": resultColumns,
		"data":    result,
		"shape":   []int{len(result), len(resultColumns)},
		"groups":  len(groups),
	}, nil
}

// matchesConditions checks if a row matches the given conditions
func (p *PandasToolkit) matchesConditions(row map[string]interface{}, conditions map[string]interface{}) bool {
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

		if !p.compareValues(rowValue, value, operator) {
			return false
		}
	}

	return true
}

// compareValues compares two values based on the operator
func (p *PandasToolkit) compareValues(a, b interface{}, operator string) bool {
	// Try to convert to numbers for numeric comparison
	aNum, aIsNum := p.toFloat64(a)
	bNum, bIsNum := p.toFloat64(b)

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
		return contains(aStr, bStr)
	case "starts_with":
		return startsWith(aStr, bStr)
	case "ends_with":
		return endsWith(aStr, bStr)
	}

	return false
}

// Helper functions for statistics
func (p *PandasToolkit) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case float32:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

func (p *PandasToolkit) sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func (p *PandasToolkit) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return p.sum(values) / float64(len(values))
}

func (p *PandasToolkit) std(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := p.mean(values)
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return variance
}

func (p *PandasToolkit) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (p *PandasToolkit) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (p *PandasToolkit) percentile(values []float64, pct float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	index := (pct / 100) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// String helper functions
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}