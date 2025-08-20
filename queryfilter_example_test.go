package sqld_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/getangry/sqld"
)

// ExampleBuildFromRequest demonstrates parsing HTTP query parameters into SQL conditions
func ExampleBuildFromRequest() {
	// Simulate an HTTP request with query parameters
	req := httptest.NewRequest("GET", "/users?name=john&age[gt]=18&status[in]=active,pending", nil)

	// Configure which fields are allowed to be filtered
	config := &sqld.QueryFilterConfig{
		AllowedFields: map[string]bool{
			"name":   true,
			"age":    true,
			"status": true,
		},
		DefaultOperator: sqld.OpEq,
		MaxFilters:      10,
	}

	// Build WHERE conditions from the request
	where, err := sqld.BuildFromRequest(req, sqld.Postgres, config)
	if err != nil {
		panic(err)
	}

	// Generate SQL
	sql, params := where.Build()
	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Params: %v\n", params)

	// Output:
	// SQL: name = $1 AND age > $2 AND status IN ($3, $4)
	// Params: [john 18 active pending]
}

// Example_userAPI demonstrates a complete REST API handler using queryfilter
func Example_userAPI() {
	// HTTP handler that uses queryfilter for dynamic filtering
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Configure allowed filters with field mappings
		config := &sqld.QueryFilterConfig{
			AllowedFields: map[string]bool{
				"name":       true,
				"email":      true,
				"age":        true,
				"status":     true,
				"created_at": true,
				"country":    true,
			},
			FieldMappings: map[string]string{
				"user_name": "name",        // Map user_name -> name
				"user_age":  "age",         // Map user_age -> age
				"signup":    "created_at",  // Map signup -> created_at
			},
			DefaultOperator: sqld.OpEq,
			DateLayout:      "2006-01-02",
			MaxFilters:      20,
		}

		// Parse filters from request
		where, err := sqld.BuildFromRequest(r, sqld.Postgres, config)
		if err != nil {
			http.Error(w, "Invalid filters: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Build the complete query
		baseQuery := "SELECT id, name, email, age, status, created_at FROM users"
		query, params := sqld.InjectWhereCondition(baseQuery, where, sqld.Postgres)

		// In a real application, you would execute this query
		fmt.Printf("Query: %s\n", query)
		fmt.Printf("Params: %v\n", params)
	}

	// Test the handler with various query patterns
	testRequests := []string{
		"/users?name=john&age[gt]=18",
		"/users?email[contains]=example.com&status[in]=active,verified",
		"/users?signup[after]=2024-01-01&country[eq]=US",
		"/users?user_name[startswith]=admin&age[between]=25,65",
	}

	for i, url := range testRequests {
		fmt.Printf("\n--- Request %d: %s ---\n", i+1, url)
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}

	// Output:
	// --- Request 1: /users?name=john&age[gt]=18 ---
	// Query: SELECT id, name, email, age, status, created_at FROM users WHERE name = $1 AND age > $2
	// Params: [john 18]
	//
	// --- Request 2: /users?email[contains]=example.com&status[in]=active,verified ---
	// Query: SELECT id, name, email, age, status, created_at FROM users WHERE email ILIKE $1 AND status IN ($2, $3)
	// Params: [%example.com% active verified]
	//
	// --- Request 3: /users?signup[after]=2024-01-01&country[eq]=US ---
	// Query: SELECT id, name, email, age, status, created_at FROM users WHERE created_at > $1 AND country = $2
	// Params: [2024-01-01 US]
	//
	// --- Request 4: /users?user_name[startswith]=admin&age[between]=25,65 ---
	// Query: SELECT id, name, email, age, status, created_at FROM users WHERE name ILIKE $1 AND age BETWEEN $2 AND $3
	// Params: [admin% 25 65]
}

// Example_advancedFiltering demonstrates complex filtering scenarios
func Example_advancedFiltering() {
	// Complex query with multiple filter types
	queryString := "name[contains]=john&age[between]=25,65&status[in]=active,verified&created_at[after]=2024-01-01&deleted_at[isnull]=true&country[ne]=US"

	config := &sqld.QueryFilterConfig{
		AllowedFields: map[string]bool{
			"name":       true,
			"age":        true,
			"status":     true,
			"created_at": true,
			"deleted_at": true,
			"country":    true,
		},
		DefaultOperator: sqld.OpEq,
		DateLayout:      "2006-01-02",
		MaxFilters:      20,
	}

	where, err := sqld.BuildFromQueryString(queryString, sqld.Postgres, config)
	if err != nil {
		panic(err)
	}

	sql, params := where.Build()
	fmt.Printf("Complex SQL: %s\n", sql)
	fmt.Printf("Param count: %d\n", len(params))

	// Output:
	// Complex SQL: name ILIKE $1 AND age BETWEEN $2 AND $3 AND status IN ($4, $5) AND created_at > $6 AND deleted_at IS NULL AND country != $7
	// Param count: 7
}

// Example_serviceIntegration shows how to integrate with a service layer
func Example_serviceIntegration() {
	type UserService struct {
		enhanced *sqld.EnhancedQueries[interface{}] // Replace with your actual Queries type
	}

	// Service method that uses queryfilter
	searchUsers := func(ctx context.Context, r *http.Request) ([]interface{}, error) {
		// Configure allowed search fields
		config := &sqld.QueryFilterConfig{
			AllowedFields: map[string]bool{
				"name":     true,
				"email":    true,
				"status":   true,
				"role":     true,
				"country":  true,
				"age":      true,
				"verified": true,
			},
			FieldMappings: map[string]string{
				"is_verified": "verified",
			},
			DefaultOperator: sqld.OpEq,
			MaxFilters:      15,
		}

		// Parse filters from request
		where, err := sqld.BuildFromRequest(r, sqld.Postgres, config)
		if err != nil {
			return nil, fmt.Errorf("invalid filters: %w", err)
		}

		// Base query for user search
		baseQuery := `
			SELECT id, name, email, status, role, country, age, verified, created_at
			FROM users`

		// Add WHERE conditions if any exist
		var query string
		var params []interface{}
		
		if where.HasConditions() {
			query, params = sqld.InjectWhereCondition(baseQuery, where, sqld.Postgres)
		} else {
			query = baseQuery
		}

		// Add ordering and pagination
		query += " ORDER BY created_at DESC LIMIT 50"

		fmt.Printf("Service Query: %s\n", query)
		fmt.Printf("Service Params: %v\n", params)

		// In real code, execute the query here
		return nil, nil
	}

	// Test the service
	req := httptest.NewRequest("GET", "/api/users?status[in]=active,verified&country=US&age[gte]=18&is_verified=true", nil)
	_, err := searchUsers(context.Background(), req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Output:
	// Service Query: SELECT id, name, email, status, role, country, age, verified, created_at
	// FROM users WHERE status IN ($1, $2) AND country = $3 AND age >= $4 AND verified = $5 ORDER BY created_at DESC LIMIT 50
	// Service Params: [active verified US 18 true]
}

// Example_filterSyntax demonstrates different query syntax options
func Example_filterSyntax() {
	examples := []string{
		// Bracket syntax
		"name[eq]=john&age[gt]=18&email[contains]=example",
		// Underscore syntax  
		"name_eq=john&age_gt=18&email_contains=example",
		// Mixed syntax
		"name=john&age[gt]=18&email_contains=example",
		// Complex operators
		"age[between]=18,65&role[in]=admin,user&created_at[after]=2024-01-01",
	}

	config := sqld.DefaultQueryFilterConfig()
	config.AllowedFields = map[string]bool{
		"name": true, "age": true, "email": true, "role": true, "created_at": true,
	}

	for i, queryString := range examples {
		fmt.Printf("\n--- Syntax Example %d ---\n", i+1)
		fmt.Printf("Query: %s\n", queryString)
		
		where, err := sqld.BuildFromQueryString(queryString, sqld.Postgres, config)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		sql, params := where.Build()
		fmt.Printf("SQL: %s\n", sql)
		fmt.Printf("Params: %v\n", params)
	}

	// Output will show the different syntax options and their SQL translations
}