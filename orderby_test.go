package sqld

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderByBuilder(t *testing.T) {
	t.Run("Basic ordering", func(t *testing.T) {
		builder := NewOrderByBuilder()

		result := builder.Asc("name").Desc("created_at").Build()
		expected := "name ASC, created_at DESC"

		assert.Equal(t, expected, result)
	})

	t.Run("Empty builder", func(t *testing.T) {
		builder := NewOrderByBuilder()

		assert.False(t, builder.HasFields())
		assert.Equal(t, "", builder.Build())
		assert.Equal(t, "", builder.BuildWithPrefix())
	})

	t.Run("With prefix", func(t *testing.T) {
		builder := NewOrderByBuilder()
		builder.Desc("score").Asc("id")

		result := builder.BuildWithPrefix()
		expected := "ORDER BY score DESC, id ASC"

		assert.Equal(t, expected, result)
	})

	t.Run("Clear builder", func(t *testing.T) {
		builder := NewOrderByBuilder()
		builder.Asc("name").Desc("date")

		assert.True(t, builder.HasFields())

		builder.Clear()
		assert.False(t, builder.HasFields())
		assert.Equal(t, "", builder.Build())
	})
}

func TestSortFieldFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected SortField
	}{
		{"name", SortField{"name", SortAsc}},
		{"name:asc", SortField{"name", SortAsc}},
		{"name:desc", SortField{"name", SortDesc}},
		{"-name", SortField{"name", SortDesc}},
		{"+name", SortField{"name", SortAsc}},
		{"email:DESC", SortField{"email", SortDesc}},
		{"created_at:descending", SortField{"created_at", SortDesc}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := SortFieldFromString(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestParseSortFields(t *testing.T) {
	t.Run("Comma separated string", func(t *testing.T) {
		input := "name:desc,email:asc,created_at"
		result := ParseSortFields(input)

		expected := []SortField{
			{"name", SortDesc},
			{"email", SortAsc},
			{"created_at", SortAsc},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("Array of strings", func(t *testing.T) {
		input := []string{"-name", "+email", "created_at:desc"}
		result := ParseSortFields(input)

		expected := []SortField{
			{"name", SortDesc},
			{"email", SortAsc},
			{"created_at", SortDesc},
		}

		assert.Equal(t, expected, result)
	})

	t.Run("Empty string", func(t *testing.T) {
		result := ParseSortFields("")
		assert.Empty(t, result)
	})
}

func TestOrderByConfig(t *testing.T) {
	t.Run("Allowed fields validation", func(t *testing.T) {
		config := &Config{
			AllowedFields: map[string]bool{
				"name":  true,
				"email": true,
			},
			MaxSortFields: 3,
		}

		// Valid fields
		fields := []SortField{
			{"name", SortDesc},
			{"email", SortAsc},
		}

		builder, err := config.ValidateAndBuild(fields)
		assert.NoError(t, err)
		assert.NotNil(t, builder)

		result := builder.Build()
		assert.Equal(t, "name DESC, email ASC", result)
	})

	t.Run("Invalid field rejection", func(t *testing.T) {
		config := &Config{
			AllowedFields: map[string]bool{
				"name": true,
			},
			MaxSortFields: 3,
		}

		fields := []SortField{
			{"name", SortAsc},
			{"forbidden_field", SortDesc},
		}

		_, err := config.ValidateAndBuild(fields)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden_field")
	})

	t.Run("Too many fields", func(t *testing.T) {
		config := &Config{
			AllowedFields: map[string]bool{
				"field1": true,
				"field2": true,
				"field3": true,
			},
			MaxSortFields: 2,
		}

		fields := []SortField{
			{"field1", SortAsc},
			{"field2", SortDesc},
			{"field3", SortAsc},
		}

		_, err := config.ValidateAndBuild(fields)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many sort fields")
	})

	t.Run("Field mapping", func(t *testing.T) {
		config := &Config{
			AllowedFields: map[string]bool{
				"user_name": true,
				"signup":    true,
			},
			FieldMappings: map[string]string{
				"user_name": "name",
				"signup":    "created_at",
			},
			MaxSortFields: 3,
		}

		fields := []SortField{
			{"user_name", SortDesc},
			{"signup", SortAsc},
		}

		builder, err := config.ValidateAndBuild(fields)
		assert.NoError(t, err)

		result := builder.Build()
		assert.Equal(t, "name DESC, created_at ASC", result)
	})

	t.Run("Default sort", func(t *testing.T) {
		config := &Config{
			AllowedFields: map[string]bool{
				"created_at": true,
				"id":         true,
			},
			DefaultSort: []SortField{
				{"created_at", SortDesc},
				{"id", SortAsc},
			},
			MaxSortFields: 3,
		}

		// Empty fields should use default
		builder, err := config.ValidateAndBuild([]SortField{})
		assert.NoError(t, err)

		result := builder.Build()
		assert.Equal(t, "created_at DESC, id ASC", result)
	})
}

func TestParseSortFromValues(t *testing.T) {
	t.Run("Standard sort parameter", func(t *testing.T) {
		values := url.Values{
			"sort": []string{"name:desc,email:asc"},
		}

		config := &Config{
			AllowedFields: map[string]bool{
				"name":  true,
				"email": true,
			},
			MaxSortFields: 5,
		}

		builder, err := ParseSortFromValues(values, config)
		assert.NoError(t, err)

		result := builder.Build()
		assert.Equal(t, "name DESC, email ASC", result)
	})

	t.Run("Individual sort fields", func(t *testing.T) {
		values := url.Values{
			"sort_name":   []string{"desc"},
			"sort_email":  []string{"asc"},
			"sort_status": []string{"desc"},
		}

		config := &Config{
			AllowedFields: map[string]bool{
				"name":   true,
				"email":  true,
				"status": true,
			},
			MaxSortFields: 5,
		}

		builder, err := ParseSortFromValues(values, config)
		assert.NoError(t, err)
		assert.True(t, builder.HasFields())

		// Should contain all three sort fields
		fields := builder.GetFields()
		assert.Len(t, fields, 3)
	})

	t.Run("Multiple sort parameter formats", func(t *testing.T) {
		// Test different parameter names
		paramNames := []string{"sort", "sort_by", "order_by", "orderby", "order"}

		for _, paramName := range paramNames {
			t.Run("param_"+paramName, func(t *testing.T) {
				values := url.Values{
					paramName: []string{"name:desc"},
				}

				config := &Config{
					AllowedFields: map[string]bool{"name": true},
					MaxSortFields: 5,
				}

				builder, err := ParseSortFromValues(values, config)
				assert.NoError(t, err)

				result := builder.Build()
				assert.Equal(t, "name DESC", result)
			})
		}
	})
}

func TestFromRequestWithSort(t *testing.T) {
	t.Run("Combined filters and sorting", func(t *testing.T) {
		// Create a mock request
		req, _ := http.NewRequest("GET", "/users?status=active&age[gte]=18&sort=name:desc,email:asc", nil)

		config := &Config{
			AllowedFields: map[string]bool{
				"status": true,
				"age":    true,
				"name":   true,
				"email":  true,
			},
			DefaultOperator: OpEq,
			MaxFilters:      10,
			MaxSortFields:   5,
		}

		where, orderBy, err := FromRequestWithSort(req, Postgres, config)
		assert.NoError(t, err)
		assert.NotNil(t, where)
		assert.NotNil(t, orderBy)

		// Check filters
		whereSQL, _ := where.Build()
		assert.Contains(t, whereSQL, "status = $")
		assert.Contains(t, whereSQL, "age >= $")

		// Check sorting
		orderSQL := orderBy.Build()
		assert.Equal(t, "name DESC, email ASC", orderSQL)
	})
}
