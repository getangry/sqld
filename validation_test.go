package sqld

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		dialect     Dialect
		expectError bool
		errorType   string
	}{
		{
			name:        "empty query",
			query:       "",
			dialect:     Postgres,
			expectError: true,
			errorType:   "validation",
		},
		{
			name:        "simple select",
			query:       "SELECT * FROM users",
			dialect:     Postgres,
			expectError: false,
		},
		{
			name:        "multiple statements",
			query:       "SELECT * FROM users; DROP TABLE users;",
			dialect:     Postgres,
			expectError: true,
			errorType:   "validation",
		},
		{
			name:        "valid subquery",
			query:       "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			dialect:     Postgres,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQuery(tt.query, tt.dialect)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType == "validation" {
					var vErr *ValidationError
					assert.True(t, errors.As(err, &vErr))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name        string
		column      string
		expectError bool
	}{
		{
			name:        "empty column",
			column:      "",
			expectError: true,
		},
		{
			name:        "simple column",
			column:      "name",
			expectError: false,
		},
		{
			name:        "quoted column",
			column:      `"user_name"`,
			expectError: false,
		},
		{
			name:        "table.column",
			column:      "users.name",
			expectError: false,
		},
		{
			name:        "column with special chars",
			column:      "name--",
			expectError: true,
		},
		{
			name:        "SQL injection attempt",
			column:      "name; DROP TABLE users;",
			expectError: true,
		},
		{
			name:        "function expression",
			column:      "UPPER(name)",
			expectError: false, // This should be allowed for complex expressions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColumnName(tt.column)

			if tt.expectError {
				assert.Error(t, err)
				var vErr *ValidationError
				assert.True(t, errors.As(err, &vErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		expectError bool
	}{
		{
			name:        "empty table",
			table:       "",
			expectError: true,
		},
		{
			name:        "simple table",
			table:       "users",
			expectError: false,
		},
		{
			name:        "schema.table",
			table:       "public.users",
			expectError: false,
		},
		{
			name:        "quoted table",
			table:       `"user_table"`,
			expectError: false,
		},
		{
			name:        "invalid characters",
			table:       "users@domain",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTableName(tt.table)

			if tt.expectError {
				assert.Error(t, err)
				var vErr *ValidationError
				assert.True(t, errors.As(err, &vErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateOrderBy(t *testing.T) {
	tests := []struct {
		name        string
		orderBy     string
		expectError bool
	}{
		{
			name:        "empty order by",
			orderBy:     "",
			expectError: true,
		},
		{
			name:        "simple column",
			orderBy:     "name",
			expectError: false,
		},
		{
			name:        "column with ASC",
			orderBy:     "name ASC",
			expectError: false,
		},
		{
			name:        "column with DESC",
			orderBy:     "name DESC",
			expectError: false,
		},
		{
			name:        "multiple columns",
			orderBy:     "name ASC, created_at DESC",
			expectError: false,
		},
		{
			name:        "invalid direction",
			orderBy:     "name INVALID",
			expectError: true,
		},
		{
			name:        "too many tokens",
			orderBy:     "name ASC EXTRA",
			expectError: true,
		},
		{
			name:        "invalid column name",
			orderBy:     "name; DROP TABLE users;",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrderBy(tt.orderBy)

			if tt.expectError {
				assert.Error(t, err)
				var vErr *ValidationError
				assert.True(t, errors.As(err, &vErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "string value",
			value: "test",
		},
		{
			name:  "string with SQL keywords",
			value: "SELECT something",
		},
		{
			name:  "integer value",
			value: 42,
		},
		{
			name:  "nil value",
			value: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValue(tt.value)
			// Currently, ValidateValue doesn't return errors for any values
			// as parameterized queries handle SQL injection protection
			assert.NoError(t, err)
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dialect  Dialect
		expected string
	}{
		{
			name:     "postgres simple",
			input:    "user_name",
			dialect:  Postgres,
			expected: `"user_name"`,
		},
		{
			name:     "mysql simple",
			input:    "user_name",
			dialect:  MySQL,
			expected: "`user_name`",
		},
		{
			name:     "sqlite simple",
			input:    "user_name",
			dialect:  SQLite,
			expected: `"user_name"`,
		},
		{
			name:     "with special chars",
			input:    "user@name!",
			dialect:  Postgres,
			expected: `"username"`,
		},
		{
			name:     "with schema",
			input:    "public.users",
			dialect:  Postgres,
			expected: `"public.users"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIdentifier(tt.input, tt.dialect)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountStatements(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{
			name:     "single statement",
			query:    "SELECT * FROM users",
			expected: 1,
		},
		{
			name:     "two statements",
			query:    "SELECT * FROM users; DELETE FROM users",
			expected: 2,
		},
		{
			name:     "with subquery",
			query:    "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			expected: 1,
		},
		{
			name:     "with string containing semicolon",
			query:    "SELECT 'hello; world' FROM users",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countStatements(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveStringLiteralsAndComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no strings or comments",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "with string literal",
			input:    "SELECT 'test' FROM users",
			expected: "SELECT  FROM users",
		},
		{
			name:     "with line comment",
			input:    "SELECT * FROM users -- comment",
			expected: "SELECT * FROM users ",
		},
		{
			name:     "with block comment",
			input:    "SELECT * /* comment */ FROM users",
			expected: "SELECT *  FROM users",
		},
		{
			name:     "complex query",
			input:    "SELECT 'test'; -- comment\nSELECT /* block */ 'another'",
			expected: "SELECT ; SELECT  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeStringLiteralsAndComments(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecureQueryBuilder(t *testing.T) {
	t.Run("valid query", func(t *testing.T) {
		sqb := NewSecureQueryBuilder("SELECT * FROM users", Postgres)

		where := NewWhereBuilder(Postgres)
		where.Equal("name", "John")
		sqb.Where(where)

		query, params, err := sqb.Build()

		assert.NoError(t, err)
		assert.Contains(t, query, "SELECT * FROM users WHERE")
		assert.Equal(t, []interface{}{"John"}, params)
	})

	t.Run("validation disabled", func(t *testing.T) {
		sqb := NewSecureQueryBuilder("SELECT * FROM users", Postgres)
		sqb.DisableValidation()

		// This should work even with validation disabled
		query, params, err := sqb.Build()

		assert.NoError(t, err)
		assert.Equal(t, "SELECT * FROM users", query)
		assert.Empty(t, params)
	})
}
