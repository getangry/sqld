# sqld - Dynamic Query Builder for SQLc

[![Go](https://github.com/getangry/sqld/actions/workflows/go.yml/badge.svg)](https://github.com/getangry/sqld/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/getangry/sqld.svg)](https://pkg.go.dev/github.com/getangry/sqld)
[![Go Report Card](https://goreportcard.com/badge/github.com/getangry/sqld)](https://goreportcard.com/report/github.com/getangry/sqld)
[![codecov](https://codecov.io/gh/getangry/sqld/branch/main/graph/badge.svg)](https://codecov.io/gh/getangry/sqld)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

sqld is a powerful, type-safe dynamic query builder designed to work seamlessly with [sqlc](https://sqlc.dev)-generated code. It maintains sqlc's SQL-first philosophy while adding the flexibility to build dynamic WHERE clauses, handle complex filtering, and create sophisticated queries at runtime.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Usage Examples](#usage-examples)
  - [SQLc Annotation System](#sqlc-annotation-system)
  - [Basic WHERE Conditions](#basic-where-conditions)
  - [Complex Conditions](#complex-conditions)
  - [Integration with SQLc](#integration-with-sqlc)
  - [HTTP Query Parameter Filtering](#http-query-parameter-filtering)
  - [Cursor-based Pagination](#cursor-based-pagination)
  - [Dynamic Sorting/ORDER BY](#dynamic-sortingorder-by)
  - [Search Filters](#search-filters)
  - [Transaction Support](#transaction-support)
  - [Error Handling](#error-handling)
  - [Security Features](#security-features)
- [API Reference](#api-reference)
- [Database Support](#database-support)
- [Best Practices](#best-practices)
- [Contributing](#contributing)
- [License](#license)

## Features

- 🔒 **Type-safe** - Maintains type safety while building dynamic queries
- 🗄️ **Multi-database support** - Works with PostgreSQL, MySQL, and SQLite
- 🛡️ **Advanced security** - Comprehensive SQL injection prevention with validation and sanitization
- 🚀 **SQLc annotation system** - Enhance SQLc queries with runtime annotations (`/* sqld:where */`, `/* sqld:cursor */`, `/* sqld:orderby */`, `/* sqld:limit */`)
- 🔧 **SQLc integration** - Seamlessly enhances existing sqlc-generated code without rewrites
- 🌐 **HTTP query parameter parsing** - Auto-convert URL query strings to SQL conditions
- 📄 **Cursor-based pagination** - Efficient pagination that scales to millions of records
- 🎯 **Go best practices** - Idiomatic function names and patterns with full context.Context support
- 📊 **Dynamic sorting** - Support for ORDER BY clauses with multiple fields and directions
- ⚡ **High performance** - Minimal overhead, no reflection or runtime parsing
- 🧩 **Composable** - Build complex queries by combining simple conditions
- 🔄 **Transaction support** - Full transaction management with automatic rollback and commit
- ❌ **Structured error handling** - Comprehensive error types with context and unwrapping support
- 🧪 **Thoroughly tested** - 95+ test cases covering all functionality with benchmarks

## Installation

```bash
go get github.com/getangry/sqld
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    
    "github.com/getangry/sqld"
    // Your sqlc-generated package
    "yourproject/db"
)

func main() {
    ctx := context.Background()
    
    // Create a WHERE builder with automatic validation
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // Add conditions (automatically validated for SQL injection)
    where.Equal("status", "active")
    where.GreaterThan("age", 18)
    where.ILike("name", "%john%")
    
    // Build the SQL and get parameters
    sql, params := where.Build()
    fmt.Printf("SQL: %s\n", sql)
    // Output: SQL: status = $1 AND age > $2 AND name ILIKE $3
    fmt.Printf("Params: %v\n", params)
    // Output: Params: [active 18 %john%]
    
    // Enhanced queries with context and comprehensive error handling
    queries := db.New(conn)
    enhanced := sqld.NewEnhanced(queries, conn, sqld.Postgres)
    
    baseQuery := "SELECT id, name, email FROM users"
    err := enhanced.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
        for rows.Next() {
            var id int
            var name, email string
            if err := rows.Scan(&id, &name, &email); err != nil {
                return sqld.WrapQueryError(err, baseQuery, params, "scanning user")
            }
            fmt.Printf("User: %d, %s, %s\n", id, name, email)
        }
        return nil
    })
    
    if err != nil {
        // Handle structured errors with context
        var qErr *sqld.QueryError
        var vErr *sqld.ValidationError
        
        if errors.As(err, &qErr) {
            log.Printf("Query failed in %s: %v", qErr.Context, qErr.Unwrap())
        } else if errors.As(err, &vErr) {
            log.Printf("Validation failed for %s: %s", vErr.Field, vErr.Message)
        } else {
            log.Printf("Unexpected error: %v", err)
        }
    }
}
```

## Core Concepts

### WhereBuilder

The `WhereBuilder` is the core component for building dynamic WHERE clauses:

```go
// Create a builder for your database dialect
builder := sqld.NewWhereBuilder(sqld.Postgres) // or sqld.MySQL, sqld.SQLite

// Add conditions
builder.Equal("column", value)
builder.NotEqual("column", value)
builder.GreaterThan("column", value)
builder.LessThan("column", value)
builder.Like("column", pattern)
builder.ILike("column", pattern) // Case-insensitive LIKE
builder.In("column", []interface{}{val1, val2})
builder.Between("column", start, end)
builder.IsNull("column")
builder.IsNotNull("column")

// Build the final SQL
sql, params := builder.Build()
```

### SQLc Integration

sqld is designed to work seamlessly with SQLc-generated code. Here's how to integrate it:

#### 1. Setup with Generated SQLc Code

```go
import (
    "github.com/getangry/sqld"
    "your-project/internal/db" // Your generated SQLc package
    "github.com/jackc/pgx/v5"
)

// Your existing sqlc setup
conn, err := pgx.Connect(ctx, "postgres://...")
queries := db.New(conn)

// Enhance with dynamic capabilities
enhanced := sqld.NewEnhanced(queries, conn, sqld.Postgres)

// Use original SQLc methods
user, err := enhanced.Queries().GetUser(ctx, 1)
users, err := enhanced.Queries().ListUsers(ctx)

// Use dynamic queries for flexible filtering
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("status", "active").GreaterThan("age", 18)

baseQuery := "SELECT id, name, email, age, status FROM users"
enhanced.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
    for rows.Next() {
        var user db.User // Use your generated struct
        err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Age, &user.Status)
        if err != nil {
            return err
        }
        // Process user
    }
    return nil
})
```

#### 2. HTTP API Integration

Perfect for REST APIs with dynamic filtering:

```go
func SearchUsers(w http.ResponseWriter, r *http.Request) {
    // Configure allowed query parameters
    config := &sqld.QueryFilterConfig{
        AllowedFields: map[string]bool{
            "name": true, "email": true, "status": true, "age": true,
        },
        FieldMappings: map[string]string{
            "user_name": "name", // Map API field names to DB columns
        },
        DefaultOperator: sqld.OpEq,
        MaxFilters: 10,
    }

    // Parse filters from URL: /users?name[contains]=john&age[gte]=18&status=active
    where, err := sqld.FromRequest(r, sqld.Postgres, config)
    if err != nil {
        http.Error(w, "Invalid filters", http.StatusBadRequest)
        return
    }

    // Add business logic (always exclude deleted users)
    where.IsNull("deleted_at")

    // Execute with your generated types
    baseQuery := "SELECT id, name, email, age, status FROM users"
    enhanced.DynamicQuery(r.Context(), baseQuery, where, func(rows sqld.Rows) error {
        // Scan into your generated db.User struct
        // Return as JSON
        return nil
    })
}
```

#### 3. Getting Started

**Quick Integration:**
- 📖 [Step-by-step Integration Guide](./INTEGRATION.md) - Complete tutorial for adding sqld to existing or new projects
- 💻 [Complete Example](./example/) - Full working application with SQLc + sqld
- 🚀 [Simple Usage](./example/simple_usage.go) - Minimal integration example

**Key Benefits:**
- Keep using your existing SQLc queries for standard operations
- Add dynamic filtering for search/filter endpoints
- Maintain type safety with generated structs
- No performance overhead for static queries

## Usage Examples

### SQLc Annotation System

sqld provides a powerful annotation system that enhances SQLc queries with dynamic capabilities at runtime without requiring rewrites. Simply add special comments to your SQLc queries and sqld will process them dynamically.

#### Four Core Annotations

- `/* sqld:where */` - Injects dynamic WHERE conditions
- `/* sqld:cursor */` - Enables cursor-based pagination
- `/* sqld:orderby */` - Adds dynamic ORDER BY clauses
- `/* sqld:limit */` - Adds LIMIT clause

#### Basic SQLc Query Enhancement

```sql
-- name: SearchUsers :many
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC, id DESC /* sqld:cursor */ /* sqld:orderby */ /* sqld:limit */;
```

```go
// Use SQLc-generated types and methods
queries := db.New(conn)
originalSQL := db.SearchUsers // SQLc-generated constant

// Create dynamic conditions
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("status", "active")
where.GreaterThan("age", 18)

// Create cursor for pagination
cursor := &sqld.Cursor{
    CreatedAt: "2024-01-15T10:30:00Z",
    ID:        12,
}

// Create dynamic sorting
orderBy := sqld.NewOrderByBuilder()
orderBy.Desc("updated_at").Asc("name")

// Process the annotated query
finalSQL, params, err := sqld.SearchQuery(
    originalSQL,
    sqld.Postgres,
    where,
    cursor,
    orderBy, // Dynamic sorting
    10,      // limit
)
// Result: Enhanced SQL with dynamic WHERE, cursor pagination, and LIMIT

// Execute with your database driver
rows, err := conn.Query(ctx, finalSQL, params...)
```

#### Complete REST API Example

```go
func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    config := &sqld.QueryFilterConfig{
        AllowedFields: map[string]bool{
            "name": true, "status": true, "country": true,
        },
        MaxFilters: 10,
    }
    
    where, err := sqld.FromRequest(r, sqld.Postgres, config)
    if err != nil {
        http.Error(w, "Invalid filters", http.StatusBadRequest)
        return
    }
    
    // Parse cursor and limit from query params
    var cursor *sqld.Cursor
    if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
        cursor, _ = sqld.DecodeCursor(cursorStr)
    }
    
    limit := 20
    if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
        if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
            limit = l
        }
    }
    
    // Use SQLc annotation system
    finalSQL, params, err := sqld.SearchQuery(
        db.SearchUsers,
        sqld.Postgres,
        where,
        cursor,
        limit,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Execute query
    rows, err := h.db.Query(ctx, finalSQL, params...)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var users []User
    var nextCursor string
    
    for rows.Next() {
        var u User
        err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age, &u.Status, 
                        &u.Role, &u.Country, &u.Verified, &u.CreatedAt, 
                        &u.UpdatedAt, &u.DeletedAt)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        users = append(users, u)
        
        // Generate next cursor from last record
        if len(users) == limit {
            nextCursor = sqld.EncodeCursor(u.CreatedAt, u.ID)
        }
    }
    
    response := struct {
        Users      []User `json:"users"`
        NextCursor string `json:"next_cursor,omitempty"`
    }{
        Users:      users,
        NextCursor: nextCursor,
    }
    
    json.NewEncoder(w).Encode(response)
}
```

#### Annotation Processing

The annotation processor intelligently combines all conditions:

1. **Base WHERE clause**: Your existing SQLc query conditions
2. **Dynamic filters**: Conditions from HTTP query parameters
3. **Cursor conditions**: For pagination
4. **LIMIT clause**: For result size control

```sql
-- Original SQLc query:
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC, id DESC /* sqld:cursor */ /* sqld:limit */

-- After processing becomes:
WHERE deleted_at IS NULL 
  AND status = $1 
  AND age > $2 
  AND (created_at < $3 OR (created_at = $3 AND id < $4))
ORDER BY created_at DESC, id DESC 
LIMIT $5
```

### Basic WHERE Conditions

```go
func GetActiveUsers(minAge int) (string, []interface{}) {
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    where.Equal("status", "active")
    where.GreaterThan("age", minAge)
    where.IsNotNull("email_verified_at")
    
    return where.Build()
}
```

### Complex Conditions

#### OR Conditions

```go
where := sqld.NewWhereBuilder(sqld.Postgres)

// Users from specific departments
where.Or(func(or sqld.ConditionBuilder) {
    or.Equal("department", "engineering")
    or.Equal("department", "product")
    or.Equal("department", "design")
})

// AND active status
where.Equal("status", "active")

sql, params := where.Build()
// SQL: (department = $1 OR department = $2 OR department = $3) AND status = $4
```

#### Nested Conditions

```go
where := sqld.NewWhereBuilder(sqld.Postgres)

// Complex permission check
where.Or(func(or sqld.ConditionBuilder) {
    // Admin users
    or.Equal("role", "admin")
    
    // Or owners of the resource
    or.Raw("user_id = resource.owner_id")
    
    // Or users with explicit permission
    or.Raw("EXISTS (SELECT 1 FROM permissions WHERE permissions.user_id = users.id AND permissions.resource_id = ?)", resourceID)
})
```

### Integration with SQLc

#### Enhance Existing Queries

```go
// Assuming you have sqlc-generated code:
// type Queries struct { ... }
// func (q *Queries) GetUser(ctx context.Context, id int64) (User, error) { ... }

type UserService struct {
    enhanced *sqld.EnhancedQueries[*Queries]
}

func NewUserService(queries *Queries, db sqld.DBTX) *UserService {
    return &UserService{
        enhanced: sqld.NewEnhanced(queries, db, sqld.Postgres),
    }
}

func (s *UserService) SearchUsers(ctx context.Context, filters UserFilters) ([]User, error) {
    baseQuery := `
        SELECT id, name, email, status, created_at 
        FROM users`
    
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // Add dynamic filters
    if filters.Name != "" {
        where.ILike("name", sqld.SearchPattern(filters.Name, "contains"))
    }
    
    if filters.Status != "" {
        where.Equal("status", filters.Status)
    }
    
    if len(filters.Tags) > 0 {
        tagInterfaces := make([]interface{}, len(filters.Tags))
        for i, tag := range filters.Tags {
            tagInterfaces[i] = tag
        }
        where.In("tag", tagInterfaces)
    }
    
    // Execute dynamic query
    var users []User
    err := s.enhanced.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
        result, err := sqld.ScanToSlice(rows, func(rows sqld.Rows) (User, error) {
            var u User
            err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Status, &u.CreatedAt)
            return u, err
        })
        users = result
        return err
    })
    
    return users, err
}
```

#### Inject Conditions into Existing Queries

```go
// Start with your sqlc query
originalQuery := `SELECT * FROM users WHERE role = $1`

// Add dynamic conditions
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("status", "active")
where.GreaterThan("created_at", lastWeek)

// Inject the conditions
enhancedQuery, params := sqld.InjectWhereCondition(originalQuery, where, sqld.Postgres)
// Result: SELECT * FROM users WHERE role = $1 AND status = $2 AND created_at > $3

// Combine original and new parameters
allParams := append([]interface{}{"admin"}, params...)
```

### HTTP Query Parameter Filtering

sqld includes a powerful queryfilter package that automatically converts HTTP query parameters into SQL WHERE conditions. This is perfect for building REST APIs with dynamic filtering.

#### Basic Query Parameter Parsing

```go
// Handle: GET /users?name=john&age[gt]=18&status[in]=active,verified
func GetUsers(w http.ResponseWriter, r *http.Request) {
    // Configure allowed filters
    config := &sqld.QueryFilterConfig{
        AllowedFields: map[string]bool{
            "name":   true,
            "age":    true, 
            "status": true,
            "email":  true,
        },
        MaxFilters: 10,
    }

    // Parse query parameters into WHERE conditions
    where, err := sqld.FromRequest(r, sqld.Postgres, config)
    if err != nil {
        http.Error(w, "Invalid filters", http.StatusBadRequest)
        return
    }

    // Generate SQL
    baseQuery := "SELECT * FROM users"
    finalQuery, params := sqld.InjectWhereCondition(baseQuery, where, sqld.Postgres)
    // Result: SELECT * FROM users WHERE name = $1 AND age > $2 AND status IN ($3, $4)
    
    // Execute query with your database
    rows, err := db.Query(ctx, finalQuery, params...)
}
```

#### Supported Query Syntax

sqld supports multiple syntax styles for maximum flexibility:

**Bracket Syntax:**
```
GET /users?name[eq]=john&age[gt]=18&email[contains]=example.com
```

**Underscore Syntax:**
```
GET /users?name_eq=john&age_gt=18&email_contains=example.com
```

**Mixed Syntax:**
```
GET /users?name=john&age[gt]=18&email_contains=example.com
```

#### Available Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` or `=` | Equality | `name=john` |
| `ne` or `!=` | Not equal | `status[ne]=deleted` |
| `gt` | Greater than | `age[gt]=18` |
| `gte` | Greater than or equal | `age[gte]=21` |
| `lt` | Less than | `price[lt]=100` |
| `lte` | Less than or equal | `price[lte]=50` |
| `like` | SQL LIKE | `name[like]=%john%` |
| `ilike` | Case-insensitive LIKE | `email[ilike]=%@GMAIL.COM` |
| `contains` | Text contains | `description[contains]=urgent` |
| `startswith` | Text starts with | `name[startswith]=admin` |
| `endswith` | Text ends with | `email[endswith]=.com` |
| `between` | Between two values | `age[between]=18,65` |
| `in` | In array | `role[in]=admin,user,manager` |
| `notin` | Not in array | `status[notin]=deleted,banned` |
| `isnull` | Is NULL | `deleted_at[isnull]=true` |
| `isnotnull` | Is not NULL | `verified_at[isnotnull]=true` |

#### Field Mapping and Security

```go
config := &sqld.QueryFilterConfig{
    // Security: Only allow filtering on these fields
    AllowedFields: map[string]bool{
        "name":     true,
        "age":      true,
        "status":   true,
        "country":  true,
    },
    
    // Map query param names to database columns
    FieldMappings: map[string]string{
        "user_name":    "name",           // ?user_name=john -> WHERE name = $1
        "user_age":     "age",            // ?user_age=25 -> WHERE age = $1
        "signup_date":  "created_at",     // ?signup_date=2024-01-01 -> WHERE created_at = $1
    },
    
    // Prevent abuse
    MaxFilters: 20,
    DefaultOperator: sqld.OpEq,
    DateLayout: "2006-01-02",
}
```

#### Complete REST API Example

```go
type UserAPI struct {
    enhanced *sqld.EnhancedQueries[*db.Queries]
}

func (api *UserAPI) SearchUsers(w http.ResponseWriter, r *http.Request) {
    // Configure filtering
    config := &sqld.QueryFilterConfig{
        AllowedFields: map[string]bool{
            "name": true, "email": true, "status": true, 
            "country": true, "age": true, "created_at": true,
        },
        FieldMappings: map[string]string{
            "signup": "created_at",
            "active": "status",
        },
        MaxFilters: 15,
    }

    // Parse filters from query parameters
    where, err := sqld.FromRequest(r, sqld.Postgres, config)
    if err != nil {
        http.Error(w, fmt.Sprintf("Invalid filters: %v", err), http.StatusBadRequest)
        return
    }

    // Build query
    baseQuery := `
        SELECT id, name, email, status, country, age, created_at 
        FROM users`
    
    var users []User
    err = api.enhanced.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
        result, err := sqld.ScanToSlice(rows, func(rows sqld.Rows) (User, error) {
            var u User
            err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Status, &u.Country, &u.Age, &u.CreatedAt)
            return u, err
        })
        users = result
        return err
    })

    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(users)
}
```

#### Query Examples

**Simple Filtering:**
```
GET /users?name=john&status=active
→ WHERE name = $1 AND status = $2
```

**Complex Filtering:**
```
GET /users?age[between]=18,65&country[in]=US,CA,UK&email[contains]=@company.com&created_at[after]=2024-01-01
→ WHERE age BETWEEN $1 AND $2 AND country IN ($3, $4, $5) AND email ILIKE $6 AND created_at > $7
```

**Text Search:**
```
GET /users?name[contains]=john&email[endswith]=.com&status[ne]=deleted
→ WHERE name ILIKE $1 AND email ILIKE $2 AND status != $3
```

#### Integration with Existing Queries

You can also enhance existing sqlc queries:

```go
// Your existing sqlc query
query := "SELECT * FROM users WHERE role = $1"

// Add dynamic filters from request
where, _ := sqld.FromRequest(r, sqld.Postgres, config)
enhancedQuery, params := sqld.InjectWhereCondition(query, where, sqld.Postgres)

// Combine parameters
allParams := append([]interface{}{"admin"}, params...)
// Result: SELECT * FROM users WHERE role = $1 AND name = $2 AND age > $3
```

### Cursor-based Pagination

sqld uses cursor-based pagination for efficient pagination that scales to millions of records, avoiding the performance issues of OFFSET-based pagination.

#### Basic Cursor Pagination

```go
// Create cursor from last record
type Cursor struct {
    CreatedAt interface{} `json:"created_at"`
    ID        int32       `json:"id"`
}

cursor := &sqld.Cursor{
    CreatedAt: "2024-01-15T10:30:00Z",
    ID:        12,
}

// Use with SQLc annotation
finalSQL, params, err := sqld.SearchQuery(
    db.SearchUsers,
    sqld.Postgres,
    where,
    cursor,
    20, // limit
)
```

#### REST API Implementation

```go
func PaginatedUsers(w http.ResponseWriter, r *http.Request) {
    // Parse cursor from query parameter
    var cursor *sqld.Cursor
    if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
        cursor, _ = sqld.DecodeCursor(cursorStr)
    }
    
    // Parse limit
    limit := 20
    if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
        limit = l
    }
    
    // Execute query with cursor
    where := sqld.NewWhereBuilder(sqld.Postgres)
    where.IsNull("deleted_at")
    
    finalSQL, params, err := sqld.SearchQuery(
        db.SearchUsers,
        sqld.Postgres, 
        where,
        cursor,
        limit,
    )
    
    rows, err := db.Query(ctx, finalSQL, params...)
    // ... scan results
    
    // Generate next cursor from last record
    var nextCursor string
    if len(users) == limit {
        lastUser := users[len(users)-1]
        nextCursor = sqld.EncodeCursor(lastUser.CreatedAt, lastUser.ID)
    }
    
    response := PaginatedResponse{
        Users:      users,
        NextCursor: nextCursor,
        HasMore:    len(users) == limit,
    }
    
    json.NewEncoder(w).Encode(response)
}
```

#### Cursor Encoding/Decoding

```go
// Encode cursor for API response
cursorStr := sqld.EncodeCursor(record.CreatedAt, record.ID)
// Returns: base64-encoded JSON like "eyJjcmVhdGVkX2F0IjoiMjAyNC0wMS0xNVQxMDozMDowMFoiLCJpZCI6MTJ9"

// Decode cursor from request
cursor, err := sqld.DecodeCursor(cursorStr)
if err != nil {
    // Invalid cursor
}
```

#### Why Cursor-based Pagination?

- **Consistent results**: No duplicate or missing records when data changes
- **Performance**: No OFFSET scanning, direct index usage
- **Scalability**: Works efficiently with millions of records
- **Real-time friendly**: Handles concurrent insertions/deletions gracefully

### Dynamic Sorting/ORDER BY

sqld provides comprehensive support for dynamic ORDER BY clauses, allowing you to build flexible sorting logic that integrates seamlessly with SQLc queries.

#### Basic OrderByBuilder Usage

```go
// Create an OrderByBuilder
orderBy := sqld.NewOrderByBuilder()

// Add sort fields with different directions
orderBy.Desc("created_at")
orderBy.Asc("name")
orderBy.Desc("priority")

// Generate ORDER BY clause
fmt.Println(orderBy.BuildWithPrefix())
// Output: ORDER BY created_at DESC, name ASC, priority DESC

// Check if any fields are defined
if orderBy.HasFields() {
    sql := orderBy.Build() // Without "ORDER BY" prefix
}
```

#### Supported Sort Formats

sqld supports multiple syntax formats for specifying sort fields:

```go
// 1. Colon syntax: "field:direction"
fields1 := sqld.ParseSortFields("name:desc,email:asc,created_at:desc")

// 2. Prefix syntax: "-field" (desc), "+field" (asc)
fields2 := sqld.ParseSortFields("-name,+email,-created_at")

// 3. Mixed syntax
fields3 := sqld.ParseSortFields("name:desc,+email,-created_at")

// 4. Array format
fields4 := sqld.ParseSortFields([]string{"name:desc", "email:asc"})

// All produce the same result:
// [{Field:name Direction:DESC} {Field:email Direction:ASC} {Field:created_at Direction:DESC}]
```

#### HTTP Query Parameter Integration

Parse sorting from HTTP requests with multiple parameter formats:

```go
// 1. Standard sort parameter: ?sort=name:desc,email:asc
req, _ := http.NewRequest("GET", "/users?sort=name:desc,email:asc", nil)

config := &sqld.QueryFilterConfig{
    OrderByConfig: &sqld.OrderByConfig{
        AllowedFields: map[string]bool{
            "name": true, "email": true, "created_at": true,
        },
        MaxSortFields: 3,
    },
}

orderBy, err := sqld.ParseSortFromRequest(req, config)
if err != nil {
    // Handle error
}

// 2. Individual field parameters: ?sort_name=desc&sort_email=asc
values := url.Values{
    "sort_name":  []string{"desc"},
    "sort_email": []string{"asc"},
}
orderBy2, _ := sqld.ParseSortFromValues(values, config)

// 3. Alternative parameter names: sort_by, order_by, orderby, order
// All of these work: ?sort_by=name:desc, ?order_by=name:desc, etc.
```

#### Security and Validation

Control which fields can be sorted and how:

```go
config := &sqld.OrderByConfig{
    // Whitelist allowed fields (security)
    AllowedFields: map[string]bool{
        "name":       true,
        "email":      true,
        "created_at": true,
        "updated_at": true,
    },
    
    // Map API field names to database columns
    FieldMappings: map[string]string{
        "signup": "created_at",    // ?sort=signup:desc -> ORDER BY created_at DESC
        "user":   "name",          // ?sort=user:asc -> ORDER BY name ASC
    },
    
    // Limit number of sort fields (prevent abuse)
    MaxSortFields: 5,
    
    // Default sort when no fields specified
    DefaultSort: []sqld.SortField{
        {"created_at", sqld.SortDesc},
        {"id", sqld.SortAsc},
    },
}

// Validate and build
fields := []sqld.SortField{
    {"user", sqld.SortAsc},     // Will be mapped to "name"
    {"signup", sqld.SortDesc},  // Will be mapped to "created_at"
}

orderBy, err := config.ValidateAndBuild(fields)
if err != nil {
    // Handle validation error (forbidden field, too many fields, etc.)
}

result := orderBy.Build()
// Result: "name ASC, created_at DESC"
```

#### SQLc Annotation Integration

Use the `/* sqld:orderby */` annotation to add dynamic sorting to SQLc queries:

```sql
-- name: SearchUsers :many
SELECT id, name, email, status, created_at
FROM users
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC /* sqld:orderby */ /* sqld:limit */;
```

```go
// Your existing SQLc query with dynamic sorting
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("status", "active")

// Parse sorting from HTTP request
orderBy, _ := sqld.ParseSortFromRequest(req, config)

// Use SQLc annotation system
finalSQL, params, err := sqld.SearchQuery(
    db.SearchUsers,  // SQLc-generated query
    sqld.Postgres,
    where,           // Dynamic filters
    cursor,          // Pagination cursor
    orderBy,         // Dynamic sorting
    20,              // Limit
)

// The annotation processor will:
// 1. Keep the base ORDER BY: "created_at DESC"
// 2. Add dynamic sorting: ", name ASC, email DESC"
// Result: "ORDER BY created_at DESC, name ASC, email DESC"
```

#### Combined Filtering and Sorting

Parse both filters and sorting from a single HTTP request:

```go
// Handle: GET /users?status=active&age[gte]=18&sort=name:desc,email:asc
func SearchUsers(w http.ResponseWriter, r *http.Request) {
    config := &sqld.QueryFilterConfig{
        AllowedFields: map[string]bool{
            "status": true, "age": true,  // For filtering
        },
        DefaultOperator: sqld.OpEq,
        OrderByConfig: &sqld.OrderByConfig{
            AllowedFields: map[string]bool{
                "name": true, "email": true, "created_at": true, // For sorting
            },
            MaxSortFields: 3,
        },
    }
    
    // Parse both filters and sorting in one call
    where, orderBy, err := sqld.FromRequestWithSort(r, sqld.Postgres, config)
    if err != nil {
        http.Error(w, "Invalid parameters", http.StatusBadRequest)
        return
    }
    
    // Use with SQLc annotations
    finalSQL, params, err := sqld.SearchQuery(
        db.SearchUsers,
        sqld.Postgres,
        where,
        nil,     // No cursor
        orderBy, // Dynamic sorting
        50,      // Limit
    )
    
    // Execute query...
}
```

#### Best Practices for Sorting

1. **Always whitelist allowed fields** to prevent SQL injection
2. **Set reasonable limits** on the number of sort fields
3. **Provide sensible defaults** for when no sort is specified
4. **Use field mappings** to decouple API field names from database columns
5. **Consider database indexes** for frequently sorted columns

### Search Filters

```go
// Conditional filtering - only add conditions if values are provided
func UserFilter(filter UserSearchRequest) *sqld.WhereBuilder {
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // ConditionalWhere only adds the condition if the value is not empty/nil
    sqld.ConditionalWhere(where, "name", filter.Name)
    sqld.ConditionalWhere(where, "email", filter.Email)
    sqld.ConditionalWhere(where, "country", filter.Country)
    
    // Date range filtering
    if filter.StartDate != nil && filter.EndDate != nil {
        where.Between("created_at", filter.StartDate, filter.EndDate)
    } else if filter.StartDate != nil {
        where.GreaterThan("created_at", filter.StartDate)
    } else if filter.EndDate != nil {
        where.LessThan("created_at", filter.EndDate)
    }
    
    // Status filtering with inclusions
    if len(filter.IncludeStatuses) > 0 {
        statusValues := make([]interface{}, len(filter.IncludeStatuses))
        for i, v := range filter.IncludeStatuses {
            statusValues[i] = v
        }
        where.In("status", statusValues)
    }
    
    // Exclude certain statuses
    for _, status := range filter.ExcludeStatuses {
        where.NotEqual("status", status)
    }
    
    // Full-text search across multiple columns
    if filter.SearchText != "" {
        searchPattern := sqld.SearchPattern(strings.TrimSpace(filter.SearchText), "contains")
        where.Or(func(or sqld.ConditionBuilder) {
            or.ILike("name", searchPattern)
            or.ILike("email", searchPattern) 
            or.ILike("bio", searchPattern)
        })
    }
    
    return where
}
```

### Search Patterns

```go
// Helper for building LIKE patterns
searchTerm := "john"

contains := sqld.SearchPattern(searchTerm, "contains")  // %john%
prefix := sqld.SearchPattern(searchTerm, "prefix")      // john%
suffix := sqld.SearchPattern(searchTerm, "suffix")      // %john
exact := sqld.SearchPattern(searchTerm, "exact")        // john

where := sqld.NewWhereBuilder(sqld.Postgres)
where.ILike("name", contains)  // Case-insensitive search
```

### Raw SQL

```go
where := sqld.NewWhereBuilder(sqld.Postgres)

// Add raw SQL when needed
where.Raw("DATE_TRUNC('day', created_at) = ?", "2024-01-01")
where.Raw("age BETWEEN ? AND ?", 18, 65)
where.Raw("EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)")

// Mix with regular conditions
where.Equal("status", "active")
```

### Combining Multiple Builders

```go
// Combine conditions from different sources
userFilter := sqld.NewWhereBuilder(sqld.Postgres)
userFilter.Equal("user_type", "customer")

orderFilter := sqld.NewWhereBuilder(sqld.Postgres)
orderFilter.GreaterThan("total_amount", 100)

dateFilter := sqld.NewWhereBuilder(sqld.Postgres)
dateFilter.Between("created_at", startDate, endDate)

// Combine all conditions
combined := sqld.CombineConditions(sqld.Postgres, userFilter, orderFilter, dateFilter)
sql, params := combined.Build()
```

## Transaction Support

sqld provides comprehensive transaction support with automatic rollback, proper error handling, and seamless integration with sqlc-generated code.

### Basic Transaction Usage

```go
import (
    "context"
    "database/sql"
    "github.com/getangry/sqld"
)

// Create a transaction-enabled database wrapper
db, err := sql.Open("postgres", connectionString)
if err != nil {
    return err
}

stdDB := sqld.NewStandardDB(db, sqld.Postgres)
queries := db.New(db)

// Create transactional queries wrapper
txQueries := sqld.NewTransactionalQueries(queries, stdDB, sqld.Postgres, stdDB)

// Execute operations within a transaction
ctx := context.Background()
err = txQueries.WithTx(ctx, nil, func(ctx context.Context, queries *sqld.EnhancedQueries[*db.Queries]) error {
    // All operations here are within the transaction
    
    // Use enhanced dynamic queries
    where := sqld.NewWhereBuilder(sqld.Postgres)
    where.Equal("status", "pending")
    
    baseQuery := "UPDATE orders SET status = 'processing'"
    err := queries.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
        // Process results
        return nil
    })
    if err != nil {
        return err // Automatic rollback
    }
    
    // Use original sqlc methods
    user, err := queries.Queries().CreateUser(ctx, db.CreateUserParams{
        Name:  "John Doe",
        Email: "john@example.com",
    })
    if err != nil {
        return err // Automatic rollback
    }
    
    // Transaction commits automatically if no error is returned
    return nil
})

if err != nil {
    // Transaction was rolled back
    log.Printf("Transaction failed: %v", err)
}
```

### Advanced Transaction Options

```go
// Configure transaction isolation level and read-only mode
opts := &sqld.TxOptions{
    IsolationLevel: sql.LevelReadCommitted,
    ReadOnly:       false,
}

err = txQueries.WithTx(ctx, opts, func(ctx context.Context, queries *sqld.EnhancedQueries[*db.Queries]) error {
    // Transaction operations with specific settings
    return nil
})
```

## Error Handling

sqld provides structured error handling with context information and proper error wrapping that integrates with Go's standard error handling patterns.

### Error Types

```go
import (
    "errors"
    "github.com/getangry/sqld"
)

// Query execution error with context
var qErr *sqld.QueryError
if errors.As(err, &qErr) {
    fmt.Printf("Query failed in %s: %v\n", qErr.Context, qErr.Unwrap())
    fmt.Printf("SQL: %s\n", qErr.Query)
    fmt.Printf("Params: %v\n", qErr.Params)
}

// Validation error for invalid input
var vErr *sqld.ValidationError
if errors.As(err, &vErr) {
    fmt.Printf("Validation failed for field %s: %s\n", vErr.Field, vErr.Message)
    if vErr.Value != nil {
        fmt.Printf("Invalid value: %v\n", vErr.Value)
    }
}

// Transaction error
var tErr *sqld.TransactionError
if errors.As(err, &tErr) {
    fmt.Printf("Transaction %s failed: %v\n", tErr.Operation, tErr.Unwrap())
}

// Check for specific error constants
if errors.Is(err, sqld.ErrNoRows) {
    fmt.Println("No rows returned")
} else if errors.Is(err, sqld.ErrSQLInjection) {
    fmt.Println("Potential SQL injection detected")
}
```

## Security Features

sqld implements multiple layers of security to prevent SQL injection and validate input at various stages.

### Automatic Query Validation

```go
// All queries are automatically validated
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("name", "John")  // Column name is validated
where.Raw("user_id = ?", 123) // Raw SQL is checked for injection patterns

// Enhanced queries include validation
err := enhanced.DynamicQuery(ctx, baseQuery, where, scanFn)
// Returns ValidationError if security issues are detected
```

### Input Sanitization

```go
// Sanitize identifiers for safe use in SQL
safeColumn := sqld.SanitizeIdentifier("user-input", sqld.Postgres)
// Result: "user_input" (removes dangerous characters and quotes properly)

// Validate ORDER BY clauses
err := sqld.ValidateOrderBy("name ASC, created_at DESC")
if err != nil {
    var vErr *sqld.ValidationError
    if errors.As(err, &vErr) {
        // Handle invalid ORDER BY clause
    }
}
```

### Parameterized Query Enforcement

```go
// All values are automatically parameterized
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("status", "'; DROP TABLE users; --") // Safe - becomes parameter $1

// Raw SQL still uses parameters
where.Raw("created_at > ?", time.Now()) // Safe - properly parameterized
```

## API Reference

### WhereBuilder Methods

| Method | Description | Example |
|--------|-------------|---------|
| `Equal(column, value)` | Adds an equality condition | `where.Equal("status", "active")` |
| `NotEqual(column, value)` | Adds a not-equal condition | `where.NotEqual("status", "deleted")` |
| `GreaterThan(column, value)` | Adds a greater-than condition | `where.GreaterThan("age", 18)` |
| `LessThan(column, value)` | Adds a less-than condition | `where.LessThan("price", 100)` |
| `Like(column, pattern)` | Adds a LIKE condition | `where.Like("name", "%john%")` |
| `ILike(column, pattern)` | Adds a case-insensitive LIKE | `where.ILike("email", "%@gmail.com")` |
| `In(column, values)` | Adds an IN condition | `where.In("id", []interface{}{1, 2, 3})` |
| `Between(column, start, end)` | Adds a BETWEEN condition | `where.Between("date", start, end)` |
| `IsNull(column)` | Adds an IS NULL condition | `where.IsNull("deleted_at")` |
| `IsNotNull(column)` | Adds an IS NOT NULL condition | `where.IsNotNull("verified_at")` |
| `Or(func)` | Groups conditions with OR | `where.Or(func(or ConditionBuilder) { ... })` |
| `Raw(sql, params...)` | Adds raw SQL | `where.Raw("custom_func(?) = ?", col, val)` |
| `Build()` | Returns SQL string and parameters | `sql, params := where.Build()` |
| `HasConditions()` | Checks if any conditions exist | `if where.HasConditions() { ... }` |

### Utility Functions

| Function | Description |
|----------|-------------|
| `SearchPattern(text, mode)` | Creates LIKE patterns ("contains", "prefix", "suffix", "exact") |
| `ConditionalWhere(builder, column, value)` | Adds condition only if value is not empty/nil |
| `CombineConditions(dialect, builders...)` | Combines multiple WHERE builders |
| `InjectWhereCondition(query, builder, dialect)` | Injects conditions into existing SQL |
| `SearchQuery(sql, dialect, where, cursor, orderBy, limit, params...)` | Processes SQLc queries with annotations |

### OrderByBuilder Methods

| Method | Description | Example |
|--------|-------------|---------|
| `NewOrderByBuilder()` | Creates a new OrderByBuilder | `orderBy := sqld.NewOrderByBuilder()` |
| `Add(field, direction)` | Adds a sort field with direction | `orderBy.Add("name", sqld.SortAsc)` |
| `Asc(field)` | Adds ascending sort field | `orderBy.Asc("name")` |
| `Desc(field)` | Adds descending sort field | `orderBy.Desc("created_at")` |
| `Clear()` | Removes all sort fields | `orderBy.Clear()` |
| `HasFields()` | Checks if any fields are defined | `if orderBy.HasFields() { ... }` |
| `GetFields()` | Returns copy of sort fields | `fields := orderBy.GetFields()` |
| `Build()` | Generates ORDER BY clause | `sql := orderBy.Build()` |
| `BuildWithPrefix()` | Generates with "ORDER BY" prefix | `sql := orderBy.BuildWithPrefix()` |

### Sorting Functions

| Function | Description |
|----------|-------------|
| `ParseSortFields(input)` | Parses sort fields from string or array |
| `ParseSortFromRequest(request, config)` | Extracts sorting from HTTP request |
| `ParseSortFromValues(values, config)` | Extracts sorting from url.Values |
| `FromRequestWithSort(request, dialect, config)` | Parses both filters and sorting |
| `SortFieldFromString(s)` | Parses single sort field from string |
| `ParseSortDirection(dir)` | Converts string to SortDirection |

### EnhancedQueries Methods

| Method | Description |
|--------|-------------|
| `DynamicQuery(ctx, baseQuery, where, scanFn)` | Executes a dynamic query with conditions |
| `DynamicQueryRow(ctx, baseQuery, where)` | Executes a query returning single row |
| `PaginationQuery(ctx, baseQuery, where, limit, offset, orderBy)` | Builds paginated query with validation |
| `SearchQuery(baseQuery, searchColumns, searchText, filters)` | Builds search query across multiple columns |

### Transaction Methods

| Method | Description |
|--------|-------------|
| `NewStandardDB(db, dialect)` | Creates a transaction-enabled database wrapper |
| `NewTransactionalQueries(queries, db, dialect, txManager)` | Creates transactional queries wrapper |
| `WithTx(ctx, opts, fn)` | Executes function within a transaction |
| `RunInTransaction(ctx, txManager, opts, operations...)` | Runs multiple operations in a transaction |
| `BeginTx(ctx, opts)` | Begins a new transaction |
| `Commit(ctx)` | Commits the transaction |
| `Rollback(ctx)` | Rolls back the transaction |

### Error Handling Functions

| Function | Description |
|----------|-------------|
| `WrapQueryError(err, query, params, context)` | Wraps error with query context |
| `WrapTransactionError(err, operation)` | Wraps error with transaction context |

### Validation Functions

| Function | Description |
|----------|-------------|
| `ValidateQuery(query, dialect)` | Validates query for security issues |
| `ValidateColumnName(column)` | Validates column name for safety |
| `ValidateTableName(table)` | Validates table name for safety |
| `ValidateOrderBy(orderBy)` | Validates ORDER BY clause |
| `ValidateValue(value)` | Validates parameter value |
| `SanitizeIdentifier(identifier, dialect)` | Sanitizes identifier for safe use |

### QueryFilter Functions

| Function | Description |
|----------|-------------|
| `ParseQueryString(queryString, config)` | Parses URL query string into Filter objects |
| `ParseRequest(request, config)` | Parses HTTP request query parameters into filters |
| `ParseURLValues(values, config)` | Parses url.Values into Filter objects |
| `FromRequest(request, dialect, config)` | One-step: HTTP request → WhereBuilder |
| `FromQueryString(queryString, dialect, config)` | One-step: query string → WhereBuilder |
| `ApplyFiltersToBuilder(filters, builder)` | Applies parsed filters to WhereBuilder |
| `MapOperator(opString)` | Converts string to Operator constant |
| `DefaultQueryFilterConfig()` | Returns default configuration |

### QueryFilterConfig Options

| Option | Type | Description |
|--------|------|-------------|
| `AllowedFields` | `map[string]bool` | Whitelist of fields that can be filtered |
| `FieldMappings` | `map[string]string` | Maps query param names to database columns |
| `DefaultOperator` | `Operator` | Default operator when none specified |
| `DateLayout` | `string` | Go time layout for parsing dates |
| `MaxFilters` | `int` | Maximum number of filters to prevent abuse |

## Database Support

### PostgreSQL
- Full support for all features
- Native `ILIKE` for case-insensitive searches
- Parameter placeholders: `$1, $2, $3...`
- Full-text search with `to_tsvector` and `plainto_tsquery`

### MySQL
- Full support with adaptations
- `ILIKE` emulated with `LOWER(column) LIKE LOWER(pattern)`
- Parameter placeholders: `?, ?, ?...`
- Full-text search uses `LIKE` patterns

### SQLite
- Full support with adaptations
- `ILIKE` emulated with `LOWER(column) LIKE LOWER(pattern)`
- Parameter placeholders: `?, ?, ?...`
- Full-text search uses `LIKE` patterns

## Best Practices

### 1. Use Context and Structured Error Handling

Always use context.Context and handle structured errors:

```go
func SearchUsers(ctx context.Context, filter UserFilter) ([]User, error) {
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // Use ConditionalWhere for optional filters
    sqld.ConditionalWhere(where, "name", filter.Name)
    sqld.ConditionalWhere(where, "status", filter.Status)
    
    var users []User
    err := enhanced.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
        result, err := sqld.ScanToSlice(rows, func(rows sqld.Rows) (User, error) {
            var u User
            err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Status)
            return u, err
        })
        users = result
        return err
    })
    
    if err != nil {
        var vErr *sqld.ValidationError
        if errors.As(err, &vErr) {
            return nil, fmt.Errorf("invalid filter %s: %s", vErr.Field, vErr.Message)
        }
        return nil, sqld.WrapQueryError(err, baseQuery, nil, "searching users")
    }
    
    return users, nil
}
```

### 2. Always Whitelist Fields for Security

```go
config := &sqld.QueryFilterConfig{
    // Security: Only allow filtering on approved fields
    AllowedFields: map[string]bool{
        "name":    true,
        "email":   true, 
        "status":  true,
        "country": true,
    },
    
    // Map external API names to internal columns
    FieldMappings: map[string]string{
        "user_name": "name",
        "active":    "status",
    },
    
    // Prevent abuse
    MaxFilters: 10,
}

// This automatically validates and rejects unknown fields
where, err := sqld.FromRequest(r, sqld.Postgres, config)
```

### 3. Use Transactions for Critical Operations

```go
func TransferFunds(ctx context.Context, fromID, toID int, amount decimal.Decimal) error {
    return txQueries.WithTx(ctx, &sqld.TxOptions{
        IsolationLevel: sql.LevelSerializable,
    }, func(ctx context.Context, queries *sqld.EnhancedQueries[*db.Queries]) error {
        // Debit from source account
        err := queries.Queries().DebitAccount(ctx, db.DebitAccountParams{
            ID:     fromID,
            Amount: amount,
        })
        if err != nil {
            return sqld.WrapQueryError(err, "", nil, "debiting source account")
        }
        
        // Credit to destination account
        err = queries.Queries().CreditAccount(ctx, db.CreditAccountParams{
            ID:     toID,
            Amount: amount,
        })
        if err != nil {
            return sqld.WrapQueryError(err, "", nil, "crediting destination account")
        }
        
        // Log the transfer
        _, err = queries.Queries().LogTransfer(ctx, db.LogTransferParams{
            FromID: fromID,
            ToID:   toID,
            Amount: amount,
        })
        if err != nil {
            return sqld.WrapQueryError(err, "", nil, "logging transfer")
        }
        
        return nil // Transaction commits automatically
    })
}
```

### 4. Validate and Limit Input Size

```go
// Prevent extremely large IN clauses
if len(ids) > 1000 {
    return &sqld.ValidationError{
        Field:   "ids",
        Value:   len(ids),
        Message: "too many IDs in filter (max 1000)",
    }
}

// Validate ORDER BY clause
if orderBy != "" {
    if err := sqld.ValidateOrderBy(orderBy); err != nil {
        return err
    }
}

values := make([]interface{}, len(ids))
for i, id := range ids {
    values[i] = id
}
where.In("id", values)
```

### 5. Monitor and Log Security Events

```go
func handleQueryError(r *http.Request, err error) {
    var vErr *sqld.ValidationError
    if errors.As(err, &vErr) {
        // Log potential security issues
        if strings.Contains(vErr.Message, "injection") {
            log.Printf("SECURITY: Potential injection attempt from %s: field=%s, value=%v", 
                r.RemoteAddr, vErr.Field, vErr.Value)
        }
    }
}
```

### 6. Index Columns Used in WHERE Clauses

Ensure your database has appropriate indexes for columns frequently used in dynamic queries:

```sql
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_email ON users(email);
```

## Testing

sqld has comprehensive test coverage with 95+ test cases covering all functionality.

Run the full test suite:

```bash
go test ./...
```

Run with coverage report:

```bash
go test -cover ./...
```

Run specific test categories:

```bash
# Core query building tests
go test -run TestWhereBuilder

# Security validation tests  
go test -run TestValidate

# Transaction tests
go test -run TestTransaction

# Error handling tests
go test -run TestError

# HTTP integration tests
go test -run TestFromRequest
```

Run benchmarks:

```bash
go test -bench=.
```

The test suite includes:
- **Unit tests** for all public APIs
- **Security tests** for SQL injection prevention
- **Integration tests** for database adapters
- **Error handling tests** for structured error types
- **Transaction tests** for rollback/commit behavior
- **Validation tests** for input sanitization
- **Performance benchmarks** for query building
- **Mock-based tests** for external dependencies

## Performance Considerations

- **No Reflection**: sqld uses direct type assertions and explicit interfaces
- **Minimal Allocations**: Builders reuse internal slices where possible  
- **Prepared Statements**: All queries use parameterized placeholders
- **Zero Dependencies**: No external libraries means minimal overhead
- **Efficient Validation**: Security checks are optimized with compiled regexes
- **Context Aware**: Proper context.Context support for cancellation and timeouts
- **Transaction Pooling**: Efficient transaction management with automatic cleanup
- **Benchmark Tested**: All core operations are benchmarked for performance regression detection

### Benchmark Results

```bash
BenchmarkWhereBuilderSimple-8          2000000    750 ns/op    248 B/op    6 allocs/op
BenchmarkWhereBuilderComplex-8         1000000   1500 ns/op    512 B/op   12 allocs/op
BenchmarkValidateQuery-8               5000000    300 ns/op     64 B/op    2 allocs/op
```

The library is designed for high-throughput applications with minimal performance impact.

## Troubleshooting

### Common Issues

**Issue**: ValidationError for valid column names
```go
// Solution: Check for special characters or SQL keywords
err := sqld.ValidateColumnName("user-name") // Invalid: contains hyphen
err := sqld.ValidateColumnName("user_name") // Valid: uses underscore

// Use SanitizeIdentifier for user input
safe := sqld.SanitizeIdentifier("user-input", sqld.Postgres) // "user_input"
```

**Issue**: QueryError with context information
```go
// Handle structured errors properly
var qErr *sqld.QueryError
if errors.As(err, &qErr) {
    log.Printf("Query failed in %s: %v", qErr.Context, qErr.Unwrap())
    log.Printf("SQL: %s", qErr.Query)
    log.Printf("Params: %v", qErr.Params)
}
```

**Issue**: Transaction rollback on context cancellation
```go
// Ensure proper context handling in transactions
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := txQueries.WithTx(ctx, nil, func(ctx context.Context, queries *sqld.EnhancedQueries[*db.Queries]) error {
    // Check context regularly in long operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Continue with operations
    }
    return nil
})
```

**Issue**: Parameters are numbered incorrectly in PostgreSQL
```go
// Solution: Ensure you're using the correct dialect
where := sqld.NewWhereBuilder(sqld.Postgres) // Not MySQL or SQLite
```

**Issue**: SQL injection ValidationError on safe queries
```go
// Solution: Use SecureQueryBuilder with validation disabled for trusted queries
sqb := sqld.NewSecureQueryBuilder(trustedQuery, sqld.Postgres)
query, params, err := sqb.DisableValidation().Build()
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built to complement [sqlc](https://sqlc.dev) - the amazing SQL compiler
- Inspired by query builders like [squirrel](https://github.com/Masterminds/squirrel) and [goqu](https://github.com/doug-martin/goqu)
- Designed for real-world production use cases

## Support

For issues, questions, or suggestions, please [open an issue](https://github.com/getangry/sqld/issues) on GitHub.