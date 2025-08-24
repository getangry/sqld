package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/getangry/sqld"
	pgxadapter "github.com/getangry/sqld/adapters/pgx"
	"github.com/getangry/sqld/example/generated/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserService demonstrates how to integrate sqld with SQLc-generated code
// This showcases the best of both worlds:
// - SQLc provides compile-time safety and optimal performance for static queries
// - sqld adds dynamic filtering capabilities with automatic reflection-based scanning
type UserService struct {
	users   *sqld.Executor[db.User]  // Typed executor for User queries
	queries *db.Queries              // Original SQLc queries for fallback
}

// NewUserService creates a new user service with SQLc integration
func NewUserService(conn *pgx.Conn) *UserService {
	queries := db.New(conn)
	// Use adapter to make pgx.Conn compatible with sqld.DBTX
	adapter := pgxadapter.NewPgxAdapter(conn)
	sqldQueries := sqld.New(adapter, sqld.Postgres)
	
	return &UserService{
		users:   sqld.NewExecutor[db.User](sqldQueries),
		queries: queries,
	}
}

// respondWithPaginatedUsers handles the pagination logic and response
func (s *UserService) respondWithPaginatedUsers(c *gin.Context, users []db.User, limit int) {
	// Check if there are more results
	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit] // Remove the extra item
	}

	// Generate next cursor
	var nextCursor *string
	if hasMore && len(users) > 0 {
		lastUser := users[len(users)-1]
		nextCursorData := APICursor{
			CreatedAt: lastUser.CreatedAt.Time,
			ID:        lastUser.ID,
		}
		nextCursorStr := nextCursorData.Encode()
		nextCursor = &nextCursorStr
	}

	c.JSON(http.StatusOK, SearchUsersResponse{
		Users:      users,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Limit:      limit,
	})
}

// Cursor represents a pagination cursor for the API
type APICursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        int32     `json:"id"`
}

// EncodeCursor encodes a cursor to a base64 string
func (c APICursor) Encode() string {
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64 cursor string
func DecodeCursor(encoded string) (*APICursor, error) {
	if encoded == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var cursor APICursor
	err = json.Unmarshal(data, &cursor)
	if err != nil {
		return nil, err
	}
	return &cursor, nil
}

// SearchUsersResponse represents the API response for user search with cursor pagination
type SearchUsersResponse struct {
	Users      []db.User `json:"users"`
	HasMore    bool      `json:"has_more"`
	NextCursor *string   `json:"next_cursor,omitempty"`
	Limit      int       `json:"limit"`
}

// SearchUsers demonstrates the SQLc + sqld annotation system with automatic scanning
//
// Key improvements:
// 1. No manual scanning code - uses reflection automatically
// 2. SQLc queries with annotations for dynamic enhancement
// 3. One-liner execution with QueryAll
// 4. Maintains all SQLc type safety and compile-time verification
func (s *UserService) SearchUsers(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Parse cursor
	cursorStr := c.Query("cursor")
	apiCursor, err := DecodeCursor(cursorStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cursor: " + err.Error()})
		return
	}

	// Convert API cursor to sqld cursor
	var cursor *sqld.Cursor
	if apiCursor != nil {
		cursor = &sqld.Cursor{
			CreatedAt: apiCursor.CreatedAt,
			ID:        apiCursor.ID,
		}
	}

	// Check if we have any query parameters for filtering (excluding pagination params)
	queryParams := c.Request.URL.Query()
	delete(queryParams, "limit")
	delete(queryParams, "cursor")
	hasFilters := len(queryParams) > 0

	if !hasFilters {
		// No filters - use SQLc SearchUsers with automatic scanning
		users, err := s.users.QueryAll(
			ctx,
			db.SearchUsers,
			nil,     // No additional filters
			cursor,  // Cursor for pagination
			nil,     // No custom ordering
			limit+1, // +1 to check for more results
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute query: " + err.Error()})
			return
		}

		s.respondWithPaginatedUsers(c, users, limit)
		return
	}

	// Complex filters and sorting - unified configuration
	config := sqld.DefaultConfig().
		WithAllowedFields(map[string]bool{
			"id":         true,
			"name":       true,
			"email":      true,
			"age":        true,
			"status":     true,
			"role":       true,
			"country":    true,
			"verified":   true,
			"created_at": true,
			"updated_at": true,
		}).
		WithFieldMappings(map[string]string{
			"user_name":    "name",
			"user_email":   "email",
			"signup_date":  "created_at",
			"signup":       "created_at",
			"last_update":  "updated_at",
			"is_verified":  "verified",
			"user_status":  "status",
			"user_country": "country",
		}).
		WithDefaultOperator(sqld.OpEq).
		WithDateLayout("2006-01-02").
		WithMaxFilters(20).
		WithMaxSortFields(3).
		WithDefaultSort([]sqld.SortField{
			{Field: "created_at", Direction: sqld.SortDesc},
			{Field: "id", Direction: sqld.SortAsc},
		})

	// Parse filters and sorting from query parameters
	where, orderBy, err := sqld.FromRequestWithSort(c.Request, sqld.Postgres, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filters or sorting: " + err.Error()})
		return
	}

	// Execute with dynamic filters and sorting using automatic reflection-based scanning
	users, err := s.users.QueryAll(
		ctx,
		db.SearchUsers,
		where,   // Dynamic filters from query parameters
		cursor,  // Cursor for pagination
		orderBy, // Dynamic sorting from query parameters
		limit+1, // +1 to check for more results
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute query: " + err.Error()})
		return
	}

	s.respondWithPaginatedUsers(c, users, limit)
}

// GetUserByIDOrEmail demonstrates mixing static SQLc queries with dynamic ones
func (s *UserService) GetUserByIDOrEmail(c *gin.Context) {
	ctx := c.Request.Context()
	identifier := c.Param("identifier")

	var user db.User
	var err error

	// Try to parse as ID first
	if id, parseErr := strconv.Atoi(identifier); parseErr == nil {
		// Use the original SQLc-generated query for ID lookup
		user, err = s.queries.GetUser(ctx, int32(id))
	} else {
		// Use the original SQLc-generated query for email lookup
		user, err = s.queries.GetUserByEmail(ctx, identifier)
	}

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUserWithFilters demonstrates using SQLc's type-safe UpdateUser
func (s *UserService) UpdateUserWithFilters(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// First, get the current user to merge with updates
	currentUser, err := s.queries.GetUser(ctx, int32(userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		return
	}

	// Parse the JSON body for update fields
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// Prepare SQLc UpdateUserParams with current values as defaults
	updateParams := db.UpdateUserParams{
		ID:       int32(userID),
		Name:     currentUser.Name,
		Email:    currentUser.Email,
		Age:      currentUser.Age,
		Status:   currentUser.Status,
		Role:     currentUser.Role,
		Country:  currentUser.Country,
		Verified: currentUser.Verified,
	}

	// Apply updates with validation
	for field, value := range updates {
		switch field {
		case "name":
			if name, ok := value.(string); ok && name != "" {
				updateParams.Name = name
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be a non-empty string"})
				return
			}
		case "email":
			if email, ok := value.(string); ok && email != "" {
				updateParams.Email = email
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email must be a non-empty string"})
				return
			}
		case "age":
			if age, ok := value.(float64); ok {
				updateParams.Age = pgtype.Int4{Int32: int32(age), Valid: true}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Age must be a number"})
				return
			}
		case "status":
			if status, ok := value.(string); ok {
				updateParams.Status = pgtype.Text{String: status, Valid: status != ""}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be a string"})
				return
			}
		case "role":
			if role, ok := value.(string); ok {
				updateParams.Role = pgtype.Text{String: role, Valid: role != ""}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be a string"})
				return
			}
		case "country":
			if country, ok := value.(string); ok {
				updateParams.Country = pgtype.Text{String: country, Valid: country != ""}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Country must be a string"})
				return
			}
		case "verified":
			if verified, ok := value.(bool); ok {
				updateParams.Verified = pgtype.Bool{Bool: verified, Valid: true}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Verified must be a boolean"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field not allowed: " + field})
			return
		}
	}

	// Use SQLc's type-safe UpdateUser query
	updatedUser, err := s.queries.UpdateUser(ctx, updateParams)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found or has been deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// CreateUserFromRequest demonstrates using SQLc for creation with validation
func (s *UserService) CreateUserFromRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Age      *int32 `json:"age"`
		Status   string `json:"status"`
		Role     string `json:"role"`
		Country  string `json:"country"`
		Verified bool   `json:"verified"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use the original SQLc-generated CreateUser method
	params := db.CreateUserParams{
		Name:     req.Name,
		Email:    req.Email,
		Age:      pgtype.Int4{Int32: *req.Age, Valid: req.Age != nil},
		Status:   pgtype.Text{String: req.Status, Valid: req.Status != ""},
		Role:     pgtype.Text{String: req.Role, Valid: req.Role != ""},
		Country:  pgtype.Text{String: req.Country, Valid: req.Country != ""},
		Verified: pgtype.Bool{Bool: req.Verified, Valid: true},
	}

	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// SearchUsersSorted demonstrates dedicated sorting with different syntax options
func (s *UserService) SearchUsersSorted(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse limit
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Configuration for sorting-focused endpoint
	config := sqld.DefaultConfig().
		WithAllowedFields(map[string]bool{
			"id":         true,
			"name":       true,
			"email":      true,
			"age":        true,
			"status":     true,
			"role":       true,
			"country":    true,
			"verified":   true,
			"created_at": true,
			"updated_at": true,
		}).
		WithFieldMappings(map[string]string{
			"signup":      "created_at",
			"last_update": "updated_at",
			"user_name":   "name",
		}).
		WithMaxSortFields(3).
		WithDefaultSort([]sqld.SortField{
			{Field: "name", Direction: sqld.SortAsc}, // Default: sort by name ascending
		})

	// Parse sorting from multiple possible parameter formats
	orderBy, err := sqld.ParseSortFromRequest(c.Request, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sorting: " + err.Error()})
		return
	}

	// Execute with sorting only (no filters)
	users, err := s.users.QueryAll(
		ctx,
		db.SearchUsers,
		nil,     // No filters
		nil,     // No cursor
		orderBy, // Dynamic sorting
		limit,   // Limit results
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute query: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
		"limit": limit,
		"sorting_examples": gin.H{
			"colon_syntax":      "?sort=name:desc,age:asc",
			"prefix_syntax":     "?sort=-name,+age",
			"mixed_syntax":      "?sort=name:desc,+age",
			"individual_params": "?sort_name=desc&sort_age=asc",
			"field_mappings":    "?sort=user_name:desc,signup:asc",
		},
	})
}

// SetupRoutes configures the HTTP routes
func SetupRoutes(userService *UserService) *gin.Engine {
	r := gin.Default()

	// User routes
	users := r.Group("/users")
	{
		users.GET("", userService.SearchUsers)                    // GET /users?name=john&age[gt]=18&status[in]=active,verified&sort=name:desc
		users.GET("/sorted", userService.SearchUsersSorted)       // GET /users/sorted?sort=name:desc,age:asc (sorting examples)
		users.GET("/:identifier", userService.GetUserByIDOrEmail) // GET /users/123 or GET /users/john@example.com
		users.POST("", userService.CreateUserFromRequest)         // POST /users
		users.PATCH("/:id", userService.UpdateUserWithFilters)    // PATCH /users/123
	}

	return r
}

func main() {
	// Database connection
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://sqld_user:sqld_password@localhost:5432/sqld_db?sslmode=disable"
	}

	conn, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer conn.Close(context.Background())

	// Initialize service
	userService := NewUserService(conn)

	// Setup routes
	r := SetupRoutes(userService)

	// Example usage URLs:
	fmt.Println("Server starting on :8080")
	fmt.Println()
	fmt.Println("=== FILTERING EXAMPLES ===")
	fmt.Println("  GET /users - List all users")
	fmt.Println("  GET /users?name[contains]=john - Search users by name containing 'john'")
	fmt.Println("  GET /users?age[gte]=18&status=active - Search active users 18 or older")
	fmt.Println("  GET /users?role[in]=admin,manager&country=US - Search US admins/managers")
	fmt.Println("  GET /users?created_at[after]=2024-01-01&verified=true - Search verified users since 2024")
	fmt.Println()
	fmt.Println("=== SORTING EXAMPLES ===")
	fmt.Println("  GET /users?sort=name:desc - Sort by name descending")
	fmt.Println("  GET /users?sort=name:desc,age:asc - Sort by name desc, then age asc")
	fmt.Println("  GET /users?sort=-name,+age - Prefix syntax: - for desc, + for asc")
	fmt.Println("  GET /users?sort_name=desc&sort_age=asc - Individual sort parameters")
	fmt.Println("  GET /users?sort=user_name:desc,signup:asc - Field mappings (user_name->name, signup->created_at)")
	fmt.Println()
	fmt.Println("=== COMBINED FILTERING + SORTING ===")
	fmt.Println("  GET /users?status=active&sort=name:desc - Filter active users, sort by name")
	fmt.Println("  GET /users?age[gte]=18&country=US&sort=-created_at,+name - Complex filter + sort")
	fmt.Println()
	fmt.Println("=== SORTING-FOCUSED ENDPOINT ===")
	fmt.Println("  GET /users/sorted - Dedicated sorting endpoint with examples in response")
	fmt.Println("  GET /users/sorted?sort=name:desc,age:asc - Try different sorting combinations")
	fmt.Println()
	fmt.Println("=== OTHER ENDPOINTS ===")
	fmt.Println("  GET /users/123 - Get user by ID")
	fmt.Println("  GET /users/john@example.com - Get user by email")
	fmt.Println("  POST /users - Create new user")
	fmt.Println("  PATCH /users/123 - Update user")

	log.Fatal(r.Run(":8080"))
}
