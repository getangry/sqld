package sqld_test

import (
	"fmt"
	"time"

	"github.com/getangry/sqld"
)

// Example demonstrates basic usage of sqld with sqlc-generated code
func Example() {
	// This example shows how to use sqld with your existing sqlc-generated queries

	// 1. Your existing sqlc setup (unchanged)
	// db := pgx.Connect(...)
	// queries := yourpackage.New(db)

	// 2. Enhance with dynamic capabilities
	// enhanced := sqld.NewEnhanced(queries, db, sqld.Postgres)

	// 3. Build dynamic WHERE conditions
	where := sqld.NewWhereBuilder(sqld.Postgres)
	where.Equal("status", "active")
	where.ILike("name", "%john%")
	where.In("role", []interface{}{"admin", "user"})

	// 4. Get the generated SQL (for demonstration)
	sql, params := where.Build()
	fmt.Printf("Generated SQL: %s\n", sql)
	fmt.Printf("Parameters: %v\n", params)

	// Output:
	// Generated SQL: status = $1 AND name ILIKE $2 AND role IN ($3, $4)
	// Parameters: [active %john% admin user]
}

// ExampleWhereBuilder_complex demonstrates complex conditions
func ExampleWhereBuilder_complex() {
	where := sqld.NewWhereBuilder(sqld.Postgres)

	// Basic conditions
	where.Equal("company_id", 123)
	where.GreaterThan("created_at", time.Now().AddDate(0, -6, 0))

	// OR grouping
	where.Or(func(or sqld.ConditionBuilder) {
		or.Equal("department", "engineering")
		or.Equal("department", "product")
	})

	// Date range
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	where.Between("hire_date", startDate, endDate)

	sql, params := where.Build()
	fmt.Printf("Complex SQL has %d parameters\n", len(params))
	fmt.Printf("Contains OR grouping: %t\n", contains(sql, "(") && contains(sql, "OR"))

	// Output:
	// Complex SQL has 6 parameters
	// Contains OR grouping: true
}

// ExampleSearchPattern demonstrates search pattern utilities
func ExampleSearchPattern() {
	patterns := []struct {
		name    string
		pattern string
	}{
		{"contains", sqld.SearchPattern("john", "contains")},
		{"exact", sqld.SearchPattern("john", "exact")},
		{"prefix", sqld.SearchPattern("john", "prefix")},
		{"suffix", sqld.SearchPattern("john", "suffix")},
	}

	for _, p := range patterns {
		fmt.Printf("%s: %s\n", p.name, p.pattern)
	}

	// Output:
	// contains: %john%
	// exact: john
	// prefix: john%
	// suffix: %john
}

// ExampleConditionalWhere demonstrates conditional condition building
func ExampleConditionalWhere() {
	where := sqld.NewWhereBuilder(sqld.Postgres)

	// These will be added
	name := "John Doe"
	status := "active"

	// These will be ignored (empty/nil)
	var email string    // empty
	var country *string // nil

	sqld.ConditionalWhere(where, "name", name)
	sqld.ConditionalWhere(where, "status", status)
	sqld.ConditionalWhere(where, "email", email)     // ignored
	sqld.ConditionalWhere(where, "country", country) // ignored

	sql, params := where.Build()
	fmt.Printf("Conditions added: %d\n", len(params))
	fmt.Printf("SQL: %s\n", sql)

	// Output:
	// Conditions added: 2
	// SQL: name = $1 AND status = $2
}

// Example of different database dialects
func Example_dialects() {
	conditions := []struct {
		name    string
		dialect sqld.Dialect
	}{
		{"PostgreSQL", sqld.Postgres},
		{"MySQL", sqld.MySQL},
		{"SQLite", sqld.SQLite},
	}

	for _, cond := range conditions {
		where := sqld.NewWhereBuilder(cond.dialect)
		where.Equal("name", "John")
		where.GreaterThan("age", 25)

		sql, _ := where.Build()
		fmt.Printf("%s: %s\n", cond.name, sql)
	}

	// Output:
	// PostgreSQL: name = $1 AND age > $2
	// MySQL: name = ? AND age > ?
	// SQLite: name = ? AND age > ?
}

// Helper function for example
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}

// UserFilters represents common search criteria
type UserFilters struct {
	Name         string
	Email        string
	Status       string
	Countries    []string
	CreatedAfter *time.Time
	SearchText   string
}

// Example_realWorld shows a realistic usage scenario
func Example_realWorld() {
	// Simulate user search filters
	filters := UserFilters{
		SearchText:   "john",
		Status:       "active",
		Countries:    []string{"US", "CA"},
		CreatedAfter: &time.Time{}, // Some date
	}

	where := sqld.NewWhereBuilder(sqld.Postgres)

	// Text search across multiple columns
	if filters.SearchText != "" {
		where.Or(func(or sqld.ConditionBuilder) {
			or.ILike("name", sqld.SearchPattern(filters.SearchText, "contains"))
			or.ILike("email", sqld.SearchPattern(filters.SearchText, "contains"))
		})
	}

	// Exact filters
	sqld.ConditionalWhere(where, "status", filters.Status)

	// Array filter
	if len(filters.Countries) > 0 {
		countryValues := make([]interface{}, len(filters.Countries))
		for i, country := range filters.Countries {
			countryValues[i] = country
		}
		where.In("country", countryValues)
	}

	// Date filter
	if filters.CreatedAfter != nil {
		where.GreaterThan("created_at", filters.CreatedAfter)
	}

	sql, params := where.Build()
	fmt.Printf("Real-world query has %d conditions\n", len(params))
	fmt.Printf("Has text search: %t\n", contains(sql, "ILIKE"))
	fmt.Printf("Has country filter: %t\n", contains(sql, "IN"))

	// Output:
	// Real-world query has 6 conditions
	// Has text search: true
	// Has country filter: true
}
