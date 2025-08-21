package sqld

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected Operator
	}{
		{"gt", OpGt},
		{"gte", OpGte},
		{"lt", OpLt},
		{"lte", OpLte},
		{"ne", OpNe},
		{"neq", OpNe},
		{"sw", OpStartsWith},
		{"startswith", OpStartsWith},
		{"ew", OpEndsWith},
		{"endswith", OpEndsWith},
		{"contains", OpContains},
		{"includes", OpContains},
		{"notcontains", OpDoesNotContain},
		{"between", OpBetween},
		{"in", OpIn},
		{"notin", OpNotIn},
		{"isnull", OpIsNull},
		{"isnotnull", OpIsNotNull},
		{"unknown", OpEq}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MapOperator(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFieldOperator(t *testing.T) {
	tests := []struct {
		key           string
		defaultOp     Operator
		expectedField string
		expectedOp    Operator
	}{
		{"name[eq]", OpGt, "name", OpEq},
		{"age[gt]", OpEq, "age", OpGt},
		{"email[contains]", OpEq, "email", OpContains},
		{"name_eq", OpGt, "name", OpEq},
		{"age_gt", OpEq, "age", OpGt},
		{"user_name_contains", OpEq, "user_name", OpContains},
		{"simple", OpILike, "simple", OpILike},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			field, op := parseFieldOperator(tt.key, tt.defaultOp)
			assert.Equal(t, tt.expectedField, field)
			assert.Equal(t, tt.expectedOp, op)
		})
	}
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		operator Operator
		expected interface{}
		hasError bool
	}{
		{
			name:     "simple string",
			value:    "test",
			operator: OpEq,
			expected: "test",
		},
		{
			name:     "between with two values",
			value:    "10,20",
			operator: OpBetween,
			expected: []string{"10", "20"},
		},
		{
			name:     "between with invalid format",
			value:    "10",
			operator: OpBetween,
			hasError: true,
		},
		{
			name:     "in with multiple values",
			value:    "a,b,c",
			operator: OpIn,
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "integer for gt",
			value:    "42",
			operator: OpGt,
			expected: 42,
		},
		{
			name:     "float for gt",
			value:    "3.14",
			operator: OpGt,
			expected: 3.14,
		},
		{
			name:     "null operator",
			value:    "anything",
			operator: OpIsNull,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValue(tt.value, tt.operator, "2006-01-02")

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseQueryString(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		config      *Config
		expected    []Filter
		hasError    bool
	}{
		{
			name:        "simple equality filter",
			queryString: "name=john",
			config:      DefaultConfig(),
			expected: []Filter{
				{Field: "name", Operator: OpEq, Value: "john"},
			},
		},
		{
			name:        "bracket syntax",
			queryString: "age[gt]=18&status[eq]=active",
			config:      DefaultConfig(),
			expected: []Filter{
				{Field: "age", Operator: OpGt, Value: 18},
				{Field: "status", Operator: OpEq, Value: "active"},
			},
		},
		{
			name:        "underscore syntax",
			queryString: "age_gt=18&email_contains=example",
			config:      DefaultConfig(),
			expected: []Filter{
				{Field: "age", Operator: OpGt, Value: 18},
				{Field: "email", Operator: OpContains, Value: "example"},
			},
		},
		{
			name:        "between operator",
			queryString: "created_at[between]=2024-01-01,2024-12-31",
			config:      DefaultConfig(),
			expected: []Filter{
				{Field: "created_at", Operator: OpBetween, Value: []string{"2024-01-01", "2024-12-31"}},
			},
		},
		{
			name:        "in operator",
			queryString: "role[in]=admin,user,manager",
			config:      DefaultConfig(),
			expected: []Filter{
				{Field: "role", Operator: OpIn, Value: []string{"admin", "user", "manager"}},
			},
		},
		{
			name:        "field mapping",
			queryString: "user_name=john",
			config: &Config{
				AllowedFields:   map[string]bool{"name": true},
				FieldMappings:   map[string]string{"user_name": "name"},
				DefaultOperator: OpEq,
				MaxFilters:      10,
			},
			expected: []Filter{
				{Field: "name", Operator: OpEq, Value: "john"},
			},
		},
		{
			name:        "disallowed field filtered out",
			queryString: "name=john&secret=value",
			config: &Config{
				AllowedFields:   map[string]bool{"name": true},
				DefaultOperator: OpEq,
				MaxFilters:      10,
			},
			expected: []Filter{
				{Field: "name", Operator: OpEq, Value: "john"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQueryString(tt.queryString, tt.config)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(result))

				for i, expected := range tt.expected {
					assert.Equal(t, expected.Field, result[i].Field)
					assert.Equal(t, expected.Operator, result[i].Operator)
					assert.Equal(t, expected.Value, result[i].Value)
				}
			}
		})
	}
}

func TestParseRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "/users?name=john&age[gt]=18", nil)
	require.NoError(t, err)

	config := DefaultConfig()
	filters, err := ParseRequest(req, config)

	assert.NoError(t, err)
	assert.Len(t, filters, 2)

	// Check that we have both filters (order may vary due to map iteration)
	nameFound := false
	ageFound := false

	for _, filter := range filters {
		switch filter.Field {
		case "name":
			assert.Equal(t, OpEq, filter.Operator)
			assert.Equal(t, "john", filter.Value)
			nameFound = true
		case "age":
			assert.Equal(t, OpGt, filter.Operator)
			assert.Equal(t, 18, filter.Value)
			ageFound = true
		}
	}

	assert.True(t, nameFound, "name filter should be present")
	assert.True(t, ageFound, "age filter should be present")
}

func TestApplyFiltersToBuilder(t *testing.T) {
	tests := []struct {
		name     string
		filters  []Filter
		expected string
		params   []interface{}
	}{
		{
			name: "equality filter",
			filters: []Filter{
				{Field: "name", Operator: OpEq, Value: "john"},
			},
			expected: "name = $1",
			params:   []interface{}{"john"},
		},
		{
			name: "multiple filters",
			filters: []Filter{
				{Field: "name", Operator: OpEq, Value: "john"},
				{Field: "age", Operator: OpGt, Value: 18},
			},
			expected: "name = $1 AND age > $2",
			params:   []interface{}{"john", 18},
		},
		{
			name: "contains filter",
			filters: []Filter{
				{Field: "email", Operator: OpContains, Value: "example"},
			},
			expected: "email ILIKE $1",
			params:   []interface{}{"%example%"},
		},
		{
			name: "between filter",
			filters: []Filter{
				{Field: "created_at", Operator: OpBetween, Value: []string{"2024-01-01", "2024-12-31"}},
			},
			expected: "created_at BETWEEN $1 AND $2",
			params:   []interface{}{"2024-01-01", "2024-12-31"},
		},
		{
			name: "in filter",
			filters: []Filter{
				{Field: "role", Operator: OpIn, Value: []string{"admin", "user"}},
			},
			expected: "role IN ($1, $2)",
			params:   []interface{}{"admin", "user"},
		},
		{
			name: "null filter",
			filters: []Filter{
				{Field: "deleted_at", Operator: OpIsNull, Value: nil},
			},
			expected: "deleted_at IS NULL",
			params:   []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewWhereBuilder(Postgres)
			err := ApplyFiltersToBuilder(tt.filters, builder)

			assert.NoError(t, err)

			sql, params := builder.Build()
			assert.Equal(t, tt.expected, sql)
			assert.Equal(t, tt.params, params)
		})
	}
}

func TestFromQueryString(t *testing.T) {
	queryString := "name=john&age[gt]=18&status[in]=active,pending"

	builder, err := FromQueryString(queryString, Postgres, DefaultConfig())
	require.NoError(t, err)

	sql, params := builder.Build()

	// Check that all conditions are present (order may vary due to map iteration)
	assert.Contains(t, sql, "name =")
	assert.Contains(t, sql, "age >")
	assert.Contains(t, sql, "status IN")
	assert.Len(t, params, 4)

	// Check that required values are present
	containsJohn := false
	containsAge := false
	containsActive := false
	containsPending := false

	for _, param := range params {
		switch param {
		case "john":
			containsJohn = true
		case 18:
			containsAge = true
		case "active":
			containsActive = true
		case "pending":
			containsPending = true
		}
	}

	assert.True(t, containsJohn, "Should contain 'john' parameter")
	assert.True(t, containsAge, "Should contain age parameter 18")
	assert.True(t, containsActive, "Should contain 'active' parameter")
	assert.True(t, containsPending, "Should contain 'pending' parameter")
}

func TestFromRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "/users?name=john&age[gte]=21&email[contains]=example", nil)
	require.NoError(t, err)

	config := DefaultConfig()
	builder, err := FromRequest(req, Postgres, config)
	require.NoError(t, err)

	sql, params := builder.Build()

	// Check that all conditions are present (order may vary due to map iteration)
	assert.Contains(t, sql, "name =")
	assert.Contains(t, sql, "age >=")
	assert.Contains(t, sql, "email ILIKE")
	assert.Len(t, params, 3)

	// Check that required values are present
	containsJohn := false
	containsAge := false
	containsExample := false

	for _, param := range params {
		switch param {
		case "john":
			containsJohn = true
		case 21:
			containsAge = true
		case "%example%":
			containsExample = true
		}
	}

	assert.True(t, containsJohn, "Should contain 'john' parameter")
	assert.True(t, containsAge, "Should contain age parameter 21")
	assert.True(t, containsExample, "Should contain '%example%' parameter")
}

func TestFilterConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := DefaultConfig()
		assert.Equal(t, OpEq, config.DefaultOperator)
		assert.Equal(t, "2006-01-02", config.DateLayout)
		assert.Equal(t, 50, config.MaxFilters)
	})

	t.Run("max filters exceeded", func(t *testing.T) {
		config := &Config{
			MaxFilters:      2,
			DefaultOperator: OpEq,
		}

		values := url.Values{}
		values.Add("field1", "value1")
		values.Add("field2", "value2")
		values.Add("field3", "value3")

		_, err := ParseURLValues(values, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many filters")
	})

	t.Run("allowed fields restriction", func(t *testing.T) {
		config := &Config{
			AllowedFields:   map[string]bool{"name": true, "email": true},
			DefaultOperator: OpEq,
			MaxFilters:      10,
		}

		values := url.Values{}
		values.Add("name", "john")
		values.Add("email", "john@example.com")
		values.Add("secret", "hidden") // This should be filtered out

		filters, err := ParseURLValues(values, config)
		assert.NoError(t, err)
		assert.Len(t, filters, 2) // Only name and email should be included
	})
}

func TestComplexQueryFiltering(t *testing.T) {
	// Test a complex real-world scenario
	queryString := "name[contains]=john&age[between]=18,65&status[in]=active,pending&created_at[after]=2024-01-01&deleted_at[isnull]=true"

	config := &Config{
		AllowedFields: map[string]bool{
			"name":       true,
			"age":        true,
			"status":     true,
			"created_at": true,
			"deleted_at": true,
		},
		DefaultOperator: OpEq,
		DateLayout:      "2006-01-02",
		MaxFilters:      20,
	}

	builder, err := FromQueryString(queryString, Postgres, config)
	require.NoError(t, err)

	sql, params := builder.Build()

	// Check that all conditions are present
	assert.Contains(t, sql, "name ILIKE")
	assert.Contains(t, sql, "age BETWEEN")
	assert.Contains(t, sql, "status IN")
	assert.Contains(t, sql, "created_at >")
	assert.Contains(t, sql, "deleted_at IS NULL")

	// Check parameter count and types (order may vary due to map iteration)
	assert.Len(t, params, 6) // %john%, 18, 65, active, pending, 2024-01-01

	// Check that required values are present
	containsJohn := false
	containsAge18 := false
	containsAge65 := false
	containsActive := false
	containsPending := false
	containsDate := false

	for _, param := range params {
		switch param {
		case "%john%":
			containsJohn = true
		case "18":
			containsAge18 = true
		case "65":
			containsAge65 = true
		case "active":
			containsActive = true
		case "pending":
			containsPending = true
		case "2024-01-01":
			containsDate = true
		}
	}

	assert.True(t, containsJohn, "Should contain '%john%' parameter")
	assert.True(t, containsAge18, "Should contain '18' parameter")
	assert.True(t, containsAge65, "Should contain '65' parameter")
	assert.True(t, containsActive, "Should contain 'active' parameter")
	assert.True(t, containsPending, "Should contain 'pending' parameter")
	assert.True(t, containsDate, "Should contain '2024-01-01' parameter")
}
