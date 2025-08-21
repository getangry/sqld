package sqld

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhereBuilder_PostgreSQL(t *testing.T) {
	tests := []struct {
		name           string
		buildCondition func(*WhereBuilder)
		expectedSQL    string
		expectedParams []interface{}
	}{
		{
			name: "single equal condition",
			buildCondition: func(b *WhereBuilder) {
				b.Equal("name", "John")
			},
			expectedSQL:    "name = $1",
			expectedParams: []interface{}{"John"},
		},
		{
			name: "multiple AND conditions",
			buildCondition: func(b *WhereBuilder) {
				b.Equal("name", "John")
				b.GreaterThan("age", 18)
				b.Equal("status", "active")
			},
			expectedSQL:    "name = $1 AND age > $2 AND status = $3",
			expectedParams: []interface{}{"John", 18, "active"},
		},
		{
			name: "IN condition with slice",
			buildCondition: func(b *WhereBuilder) {
				b.In("role", []interface{}{"admin", "user", "manager"})
			},
			expectedSQL:    "role IN ($1, $2, $3)",
			expectedParams: []interface{}{"admin", "user", "manager"},
		},
		{
			name: "BETWEEN condition",
			buildCondition: func(b *WhereBuilder) {
				b.Between("created_at", "2024-01-01", "2024-12-31")
			},
			expectedSQL:    "created_at BETWEEN $1 AND $2",
			expectedParams: []interface{}{"2024-01-01", "2024-12-31"},
		},
		{
			name: "LIKE and ILIKE conditions",
			buildCondition: func(b *WhereBuilder) {
				b.Like("email", "%@example.com")
				b.ILike("name", "%john%")
			},
			expectedSQL:    "email LIKE $1 AND name ILIKE $2",
			expectedParams: []interface{}{"%@example.com", "%john%"},
		},
		{
			name: "NULL conditions",
			buildCondition: func(b *WhereBuilder) {
				b.IsNull("deleted_at")
				b.IsNotNull("confirmed_at")
			},
			expectedSQL:    "deleted_at IS NULL AND confirmed_at IS NOT NULL",
			expectedParams: []interface{}{},
		},
		{
			name: "OR conditions",
			buildCondition: func(b *WhereBuilder) {
				b.Equal("status", "active")
				b.Or(func(or ConditionBuilder) {
					or.Equal("role", "admin")
					or.Equal("role", "manager")
				})
			},
			expectedSQL:    "status = $1 AND (role = $2 OR role = $3)",
			expectedParams: []interface{}{"active", "admin", "manager"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewWhereBuilder(Postgres)
			tt.buildCondition(builder)

			sql, params := builder.Build()
			assert.Equal(t, tt.expectedSQL, sql)
			assert.Equal(t, tt.expectedParams, params)
		})
	}
}

func TestWhereBuilder_MySQL(t *testing.T) {
	builder := NewWhereBuilder(MySQL)
	builder.Equal("name", "John")
	builder.GreaterThan("age", 18)

	sql, params := builder.Build()
	assert.Equal(t, "name = ? AND age > ?", sql)
	assert.Equal(t, []interface{}{"John", 18}, params)
}

func TestWhereBuilder_SQLite(t *testing.T) {
	builder := NewWhereBuilder(SQLite)
	builder.Equal("name", "John")
	builder.ILike("email", "%test%")

	sql, params := builder.Build()
	assert.Equal(t, "name = ? AND LOWER(email) LIKE LOWER(?)", sql)
	assert.Equal(t, []interface{}{"John", "%test%"}, params)
}

func TestEmptyConditions(t *testing.T) {
	builder := NewWhereBuilder(Postgres)

	sql, params := builder.Build()
	assert.Empty(t, sql)
	assert.Empty(t, params)
	assert.False(t, builder.HasConditions())
}

func TestNilValueHandling(t *testing.T) {
	builder := NewWhereBuilder(Postgres)

	// These should be ignored
	builder.Equal("name", nil)
	builder.NotEqual("email", nil)
	builder.GreaterThan("age", nil)
	builder.Between("date", nil, "2024-12-31")
	builder.Between("date", "2024-01-01", nil)
	builder.In("role", []interface{}{})
	builder.Like("text", "")
	builder.ILike("text", "")

	// Only this should be added
	builder.Equal("status", "active")

	sql, params := builder.Build()
	assert.Equal(t, "status = $1", sql)
	assert.Equal(t, []interface{}{"active"}, params)
}

func TestQueryBuilder(t *testing.T) {
	baseQuery := "SELECT * FROM users"

	where := NewWhereBuilder(Postgres)
	where.Equal("status", "active")
	where.GreaterThan("age", 18)

	qb := NewQueryBuilder(baseQuery, Postgres)
	qb.Where(where)

	query, params := qb.Build()

	expectedQuery := "SELECT * FROM users WHERE status = $1 AND age > $2"
	assert.Equal(t, expectedQuery, query)
	assert.Equal(t, []interface{}{"active", 18}, params)
}

func TestSearchPattern(t *testing.T) {
	tests := []struct {
		text     string
		mode     string
		expected string
	}{
		{"john", "contains", "%john%"},
		{"john", "prefix", "john%"},
		{"john", "suffix", "%john"},
		{"john", "exact", "john"},
		{"john", "unknown", "%john%"}, // defaults to contains
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result := SearchPattern(tt.text, tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionalWhere(t *testing.T) {
	builder := NewWhereBuilder(Postgres)

	// Test different types
	ConditionalWhere(builder, "name", "John") // string - should add
	ConditionalWhere(builder, "email", "")    // empty string - should skip
	ConditionalWhere(builder, "age", 25)      // int - should add
	ConditionalWhere(builder, "count", 0)     // zero int - should skip

	var nilString *string
	validString := "US"
	ConditionalWhere(builder, "country1", nilString)    // nil pointer - should skip
	ConditionalWhere(builder, "country2", &validString) // valid pointer - should add

	sql, params := builder.Build()

	// Should only have name, age, and country2
	assert.Equal(t, 3, len(params))
	assert.Equal(t, "John", params[0])
	assert.Equal(t, 25, params[1])
	assert.Equal(t, "US", params[2])

	assert.Contains(t, sql, "name = $1")
	assert.Contains(t, sql, "age = $2")
	assert.Contains(t, sql, "country2 = $3")
}

func TestCombineConditions(t *testing.T) {
	where1 := NewWhereBuilder(Postgres)
	where1.Equal("name", "John")

	where2 := NewWhereBuilder(Postgres)
	where2.Equal("status", "active")

	where3 := NewWhereBuilder(Postgres) // empty

	combined := CombineConditions(Postgres, where1, where2, where3)

	sql, params := combined.Build()

	// Should contain both conditions
	assert.Equal(t, 2, len(params)) // One from each where builder
	assert.Equal(t, []interface{}{"John", "active"}, params)
	assert.Contains(t, sql, "name = $1")
	assert.Contains(t, sql, "status = $2")
}

func TestRawSQL(t *testing.T) {
	builder := NewWhereBuilder(Postgres)
	builder.Raw("DATE_TRUNC('day', created_at) = ?", "2024-01-01")
	builder.Equal("status", "active")

	sql, params := builder.Build()

	assert.Contains(t, sql, "DATE_TRUNC('day', created_at) = $1")
	assert.Contains(t, sql, "status = $2")
	assert.Equal(t, []interface{}{"2024-01-01", "active"}, params)
}

func TestParameterAdjuster(t *testing.T) {
	adjuster := NewParameterAdjuster(Postgres)

	originalSQL := "name = $1 AND age = $2"
	adjustedSQL := adjuster.AdjustSQL(originalSQL, 5)

	// Should renumber parameters starting from offset
	assert.Contains(t, adjustedSQL, "$6") // $1 + 5
	assert.Contains(t, adjustedSQL, "$7") // $2 + 5
}

func TestDialectSpecificFeatures(t *testing.T) {
	t.Run("PostgreSQL ILIKE", func(t *testing.T) {
		builder := NewWhereBuilder(Postgres)
		builder.ILike("name", "%john%")

		sql, _ := builder.Build()
		assert.Contains(t, sql, "ILIKE")
	})

	t.Run("MySQL ILIKE fallback", func(t *testing.T) {
		builder := NewWhereBuilder(MySQL)
		builder.ILike("name", "%john%")

		sql, _ := builder.Build()
		assert.Contains(t, strings.ToUpper(sql), "LOWER")
		assert.Contains(t, strings.ToUpper(sql), "LIKE")
		assert.NotContains(t, sql, "ILIKE")
	})
}
