package main

import (
	"context"
	"log"
	"net/http"

	"github.com/getangry/sqld"
	"github.com/getangry/sqld/example/generated/db"
	"github.com/jackc/pgx/v5"
)

func main() {
	// Your existing sqlc setup
	conn, err := pgx.Connect(context.Background(), "postgres://user:password@localhost/mydb?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer conn.Close(context.Background())

	// Create SQLc queries instance
	queries := db.New(conn)

	// Enhance with dynamic capabilities
	enhanced := sqld.NewEnhanced(queries, conn, sqld.Postgres)

	// Example 1: Use original SQLc queries
	ctx := context.Background()
	user, err := enhanced.Queries().GetUser(ctx, 1)
	if err != nil {
		log.Printf("Error getting user: %v", err)
	} else {
		log.Printf("Found user: %+v", user)
	}

	// Example 2: Use dynamic queries alongside your generated ones
	http.HandleFunc("/users/search", func(w http.ResponseWriter, r *http.Request) {
		// Configure which fields can be filtered
		config := &sqld.QueryFilterConfig{
			AllowedFields: map[string]bool{
				"name":    true,
				"email":   true,
				"status":  true,
				"role":    true,
				"country": true,
				"age":     true,
			},
			FieldMappings: map[string]string{
				"user_name": "name", // Map user_name query param to name column
			},
			DefaultOperator: sqld.OpEq,
			MaxFilters:      10,
		}

		// Parse filters from URL query parameters
		where, err := sqld.BuildFromRequest(r, sqld.Postgres, config)
		if err != nil {
			http.Error(w, "Invalid filters: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Always exclude deleted users
		where.IsNull("deleted_at")

		// Build the query with your base SELECT
		baseQuery := `
			SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
			FROM users`

		// Execute dynamic query
		err = enhanced.DynamicQuery(r.Context(), baseQuery, where, func(rows sqld.Rows) error {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("["))
			first := true

			for rows.Next() {
				var user db.User
				err := rows.Scan(
					&user.ID, &user.Name, &user.Email, &user.Age,
					&user.Status, &user.Role, &user.Country, &user.Verified,
					&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
				)
				if err != nil {
					return err
				}

				if !first {
					w.Write([]byte(","))
				}
				first = false

				// In a real app, use proper JSON marshaling
				w.Write([]byte(`{"id":` + string(rune(user.ID)) + `,"name":"` + user.Name + `"}`))
			}
			w.Write([]byte("]"))
			return nil
		})

		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Example 3: Combine static and dynamic queries
	http.HandleFunc("/users/advanced-search", func(w http.ResponseWriter, r *http.Request) {
		// Use static SQLc query for initial data
		allUsers, err := enhanced.Queries().ListUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Then apply dynamic filtering on top
		config := &sqld.QueryFilterConfig{
			AllowedFields: map[string]bool{
				"status": true,
				"role":   true,
				"age":    true,
			},
			DefaultOperator: sqld.OpEq,
		}

		where, err := sqld.BuildFromRequest(r, sqld.Postgres, config)
		if err != nil {
			http.Error(w, "Invalid filters", http.StatusBadRequest)
			return
		}

		// If no dynamic filters, return all users
		if !where.HasConditions() {
			// Return allUsers as JSON (simplified)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"users": [], "message": "No filters applied, showing all users"}`))
			return
		}

		// Apply dynamic query with filters
		baseQuery := `
			SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
			FROM users`

		// Execute with dynamic conditions
		err = enhanced.DynamicQuery(r.Context(), baseQuery, where, func(rows sqld.Rows) error {
			// Process results...
			return nil
		})

		if err != nil {
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}
	})

	log.Println("Server starting on :8080")
	log.Println("Try these URLs:")
	log.Println("  http://localhost:8080/users/search?name=john")
	log.Println("  http://localhost:8080/users/search?status=active&age[gt]=18")
	log.Println("  http://localhost:8080/users/search?role[in]=admin,manager&country=US")
	log.Println("  http://localhost:8080/users/search?user_name[contains]=smith&age[between]=25,65")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
