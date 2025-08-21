package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/getangry/sqld"
	"github.com/getangry/sqld/example/generated/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserService demonstrates how to integrate sqld with SQLc-generated code
type UserService struct {
	enhanced *sqld.EnhancedQueries[*db.Queries]
	queries  *db.Queries
}

// NewUserService creates a new user service with SQLc integration
func NewUserService(conn *pgx.Conn) *UserService {
	queries := db.New(conn)
	enhanced := sqld.NewEnhanced(queries, conn, sqld.Postgres)

	return &UserService{
		enhanced: enhanced,
		queries:  queries,
	}
}

// SearchUsersResponse represents the API response for user search
type SearchUsersResponse struct {
	Users []db.User `json:"users"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

// SearchUsers handles dynamic user search with filters
func (s *UserService) SearchUsers(c *gin.Context) {
	ctx := c.Request.Context()

	// Configure allowed fields and mappings for user search
	config := &sqld.QueryFilterConfig{
		AllowedFields: map[string]bool{
			"name":       true,
			"email":      true,
			"age":        true,
			"status":     true,
			"role":       true,
			"country":    true,
			"verified":   true,
			"created_at": true,
		},
		FieldMappings: map[string]string{
			"user_name":    "name",
			"user_email":   "email",
			"signup_date":  "created_at",
			"is_verified":  "verified",
			"user_status":  "status",
			"user_country": "country",
		},
		DefaultOperator: sqld.OpEq,
		DateLayout:      "2006-01-02",
		MaxFilters:      20,
	}

	// Parse filters from query parameters
	where, err := sqld.BuildFromRequest(c.Request, sqld.Postgres, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filters: " + err.Error()})
		return
	}

	// Always filter out deleted users
	where.IsNull("deleted_at")

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build the dynamic query
	baseQuery := `
		SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
		FROM users`

	query, params := s.enhanced.PaginationQuery(baseQuery, where, limit, offset, "created_at DESC")

	// Execute the query using the enhanced queries
	var users []db.User
	err = s.enhanced.DynamicQuery(ctx, query, nil, func(rows sqld.Rows) error {
		for rows.Next() {
			var user db.User
			err := rows.Scan(
				&user.ID,
				&user.Name,
				&user.Email,
				&user.Age,
				&user.Status,
				&user.Role,
				&user.Country,
				&user.Verified,
				&user.CreatedAt,
				&user.UpdatedAt,
				&user.DeletedAt,
			)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search users: " + err.Error()})
		return
	}

	// Get total count (in a real app, you'd want to optimize this)
	countQuery := "SELECT COUNT(*) FROM users"
	if where.HasConditions() {
		whereSQL, whereParams := where.Build()
		countQuery += " WHERE " + whereSQL
		params = whereParams
	}

	var total int
	row := s.enhanced.DynamicQueryRow(ctx, countQuery, nil)
	if err := row.Scan(&total); err != nil {
		log.Printf("Failed to get user count: %v", err)
		total = len(users) // Fallback
	}

	response := SearchUsersResponse{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserByIDOrEmail demonstrates mixing dynamic and static queries
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

// UpdateUserWithFilters demonstrates dynamic updates with validation
func (s *UserService) UpdateUserWithFilters(c *gin.Context) {
	ctx := c.Request.Context()
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse the JSON body for update fields
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// Build dynamic update using sqld
	updateBuilder := sqld.NewWhereBuilder(sqld.Postgres)
	var setParts []string
	var params []interface{}
	paramIndex := 0

	// Validate and build SET clauses
	allowedFields := map[string]bool{
		"name": true, "email": true, "age": true, "status": true,
		"role": true, "country": true, "verified": true,
	}

	for field, value := range updates {
		if !allowedFields[field] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field not allowed: " + field})
			return
		}

		paramIndex++
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, paramIndex))
		params = append(params, value)
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	// Build the update query
	paramIndex++
	updateQuery := fmt.Sprintf(
		"UPDATE users SET %s, updated_at = NOW() WHERE id = $%d AND deleted_at IS NULL RETURNING id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at",
		fmt.Sprintf("%s", setParts),
		paramIndex,
	)
	params = append(params, userID)

	// Execute the update
	var updatedUser db.User
	row := s.enhanced.DynamicQueryRow(ctx, updateQuery, nil)
	err = row.Scan(
		&updatedUser.ID,
		&updatedUser.Name,
		&updatedUser.Email,
		&updatedUser.Age,
		&updatedUser.Status,
		&updatedUser.Role,
		&updatedUser.Country,
		&updatedUser.Verified,
		&updatedUser.CreatedAt,
		&updatedUser.UpdatedAt,
		&updatedUser.DeletedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
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

// SetupRoutes configures the HTTP routes
func SetupRoutes(userService *UserService) *gin.Engine {
	r := gin.Default()

	// User routes
	users := r.Group("/users")
	{
		users.GET("", userService.SearchUsers)                    // GET /users?name=john&age[gt]=18&status[in]=active,verified
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
	fmt.Println("Example URLs:")
	fmt.Println("  GET /users - List all users")
	fmt.Println("  GET /users?name[contains]=john - Search users by name containing 'john'")
	fmt.Println("  GET /users?age[gte]=18&status=active - Search active users 18 or older")
	fmt.Println("  GET /users?role[in]=admin,manager&country=US - Search US admins/managers")
	fmt.Println("  GET /users?created_at[after]=2024-01-01&verified=true - Search verified users since 2024")
	fmt.Println("  GET /users/123 - Get user by ID")
	fmt.Println("  GET /users/john@example.com - Get user by email")
	fmt.Println("  POST /users - Create new user")
	fmt.Println("  PATCH /users/123 - Update user")

	log.Fatal(r.Run(":8080"))
}
