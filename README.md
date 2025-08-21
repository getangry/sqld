# sqld - Dynamic Query Enhancement for SQLc

[![Go](https://github.com/getangry/sqld/actions/workflows/go.yml/badge.svg)](https://github.com/getangry/sqld/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/getangry/sqld.svg)](https://pkg.go.dev/github.com/getangry/sqld)
[![Go Report Card](https://goreportcard.com/badge/github.com/getangry/sqld)](https://goreportcard.com/report/github.com/getangry/sqld)
[![codecov](https://codecov.io/gh/getangry/sqld/branch/main/graph/badge.svg)](https://codecov.io/gh/getangry/sqld)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**sqld** enhances [SQLc](https://sqlc.dev)-generated code with dynamic query capabilities while preserving SQLc's SQL-first philosophy and compile-time safety. Add runtime filtering, sorting, pagination, and parameter parsing to your existing SQLc queries without rewrites.

---

## >> Quick Start

Transform your static SQLc queries into dynamic powerhouses:

**Before (Static SQLc):**
```sql
-- Only static queries
SELECT * FROM users WHERE status = 'active' ORDER BY created_at DESC LIMIT 20;
```

**After (SQLc + sqld):**
```sql
-- Dynamic SQLc query with annotations
-- name: SearchUsers :many
SELECT * FROM users 
WHERE status = 'active' /* sqld:where */
ORDER BY created_at DESC /* sqld:orderby */ /* sqld:limit */;
```

```go
// Execute with dynamic filters, sorting, and pagination
users, err := sqld.QueryAll[db.User](
    ctx, db, db.SearchUsers, sqld.Postgres,
    where,   // ?name[contains]=john&age[gte]=18
    cursor,  // Pagination cursor
    orderBy, // ?sort=name:desc,created_at:asc
    limit,   // Dynamic limit
)
```

[**See full example →**](#sqlc-annotation-system)

---

## Table of Contents

### Core Features
- [SQLc Annotation System](#sqlc-annotation-system) - Enhance SQLc queries with runtime annotations
- [HTTP Query Parameter Parsing](#http-query-parameter-parsing) - Auto-convert URL params to SQL conditions
- [Dynamic Filtering & WHERE Clauses](#dynamic-filtering--where-clauses) - Build complex conditions at runtime
- [Dynamic Sorting & ORDER BY](#dynamic-sorting--order-by) - Multi-field sorting with direction control
- [Cursor-Based Pagination](#cursor-based-pagination) - Efficient pagination for large datasets
- [Automatic Result Scanning](#automatic-result-scanning) - Zero-code result scanning with reflection

### Security & Performance
- [Security Features](#security-features) - SQL injection prevention and field validation
- [Type Safety](#type-safety) - Maintain compile-time safety with runtime flexibility
- [Performance Optimizations](#performance-optimizations) - Minimal overhead with smart caching

### Configuration & Integration
- [Configuration System](#configuration-system) - Unified config for filtering, sorting, and security
- [Database Support](#database-support) - PostgreSQL, MySQL, SQLite compatibility
- [Error Handling](#error-handling) - Structured errors with context and wrapping

### Guides & Examples
- [Installation](#installation)
- [Integration with Existing SQLc Code](#integration-with-existing-sqlc-code)
- [Best Practices](#best-practices)
- [Common Patterns](#common-patterns)
- [API Reference](#api-reference)

---

## Key Features Overview

### Perfect SQLc Integration
- **Zero rewrites** - Works with existing SQLc-generated code
- **SQL-first** - Maintains SQLc's philosophy of writing real SQL
- **Type-safe** - Preserves compile-time type safety
- **Annotation-based** - Add `/* sqld:where */` comments to enable dynamic features

### HTTP-First Design
- **URL Query Parsing** - `?name[contains]=john&age[gte]=18&sort=name:desc`
- **RESTful Filtering** - Standard HTTP query parameter conventions
- **Pagination Ready** - Cursor-based pagination with `?cursor=base64token`
- **Security Built-in** - Field whitelisting and parameter validation

### Performance & Developer Experience
- **Reflection-Based Scanning** - No manual result scanning code
- **Automatic Type Conversion** - Smart parameter type inference
- **Minimal Overhead** - Efficient query building and caching
- **Rich Error Messages** - Detailed context for debugging

---

## Installation

```bash
go get github.com/getangry/sqld
```

**Requirements:**
- Go 1.21+ (for generics)
- [SQLc](https://sqlc.dev) for generating base queries

---

## SQLc Annotation System

The heart of sqld is the annotation system that enhances your SQLc queries with dynamic capabilities.

### Basic Annotations

Add comments to your SQLc queries to enable dynamic features:

```sql
-- name: SearchUsers :many
SELECT id, name, email, age, status, created_at
FROM users
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC /* sqld:orderby */ /* sqld:limit */;
```

**Available Annotations:**
- `/* sqld:where */` - Inject dynamic WHERE conditions
- `/* sqld:orderby */` - Inject dynamic ORDER BY clauses  
- `/* sqld:limit */` - Inject dynamic LIMIT and OFFSET
- `/* sqld:cursor */` - Inject cursor-based pagination conditions

### Usage with sqld

```go
// Build dynamic query from SQLc base query
users, err := sqld.QueryAll[db.User](
    ctx,
    database,
    db.SearchUsers,    // SQLc-generated query constant
    sqld.Postgres,
    whereBuilder,      // Dynamic WHERE conditions
    cursor,            // Pagination cursor  
    orderBy,           // Dynamic sorting
    limit,             // Result limit
)
```

### How It Works

1. **SQLc generates** the base query constant (`db.SearchUsers`)
2. **sqld processes** annotation comments and injects dynamic SQL
3. **Result** is executed with full type safety maintained

**Example transformation:**
```sql
-- Input SQLc query:
SELECT * FROM users WHERE active = true /* sqld:where */ /* sqld:orderby */ /* sqld:limit */

-- Runtime result with dynamic conditions:
SELECT * FROM users WHERE active = true AND name ILIKE $1 AND age >= $2 ORDER BY name DESC, created_at ASC LIMIT $3 OFFSET $4
```

[**Learn more about SQLc integration →**](#integration-with-existing-sqlc-code)

---

## HTTP Query Parameter Parsing

sqld automatically converts HTTP query parameters into SQL conditions using intuitive syntax.

### Basic Filtering Syntax

```http
GET /users?name=john&status=active&age=25
```

```go
// Parse HTTP request into SQL conditions
config := sqld.DefaultConfig().WithAllowedFields(map[string]bool{
    "name": true, "status": true, "age": true,
})

where, err := sqld.FromRequest(request, sqld.Postgres, config)
// Generates: WHERE name = $1 AND status = $2 AND age = $3
```

### Advanced Operator Syntax

sqld supports rich query operators using bracket notation:

```http
# Text operations
GET /users?name[contains]=john          # name ILIKE '%john%'
GET /users?email[startsWith]=admin      # email ILIKE 'admin%'
GET /users?name[endsWith]=.com          # name ILIKE '%.com'

# Numeric comparisons  
GET /users?age[gte]=18                  # age >= 18
GET /users?age[lt]=65                   # age < 65
GET /users?created_at[after]=2024-01-01 # created_at > '2024-01-01'

# Array operations
GET /users?status[in]=active,verified   # status IN ('active', 'verified')
GET /users?role[notIn]=banned,deleted   # role NOT IN ('banned', 'deleted')

# Null checks
GET /users?last_login[isNull]=true      # last_login IS NULL
GET /users?profile[isNotNull]=true      # profile IS NOT NULL

# Range operations
GET /users?age[between]=18,65           # age BETWEEN 18 AND 65
```

### Alternative Syntax Support

sqld supports multiple syntax styles for flexibility:

```http
# Bracket notation (recommended)
GET /users?name[contains]=john&age[gte]=18

# Underscore notation  
GET /users?name_contains=john&age_gte=18

# Mixed usage
GET /users?name[contains]=john&age_gte=18&status=active
```

### Field Mapping and Validation

```go
config := sqld.DefaultConfig().
    WithAllowedFields(map[string]bool{
        "name": true, "email": true, "age": true,
    }).
    WithFieldMappings(map[string]string{
        "user_name": "name",        // ?user_name=john → name = 'john'
        "signup_date": "created_at", // ?signup_date[after]=2024-01-01
    }).
    WithMaxFilters(10)              // Prevent abuse

where, err := sqld.FromRequest(request, sqld.Postgres, config)
```

**Security Features:**
- **Field Whitelisting** - Only allowed fields can be filtered
- **Parameter Limits** - Prevent DoS attacks with too many filters
- **SQL Injection Prevention** - All values are properly parameterized
- **Type Validation** - Automatic type checking and conversion

[**See complete operator reference →**](#filtering-operators-reference)

---

## Dynamic Filtering & WHERE Clauses

Build complex WHERE conditions programmatically with type safety and security.

### WhereBuilder API

```go
// Create a WHERE builder for your database dialect
where := sqld.NewWhereBuilder(sqld.Postgres)

// Basic conditions
where.Equal("status", "active")
where.GreaterThan("age", 18)
where.Like("name", "john%")
where.In("role", []string{"admin", "manager"})

// Complex conditions with AND/OR logic
where.Group(func(w *sqld.WhereBuilder) {
    w.Equal("department", "engineering")
    w.Or()
    w.Equal("department", "product")
})

// Generate SQL
sql, params := where.Build()
// Result: WHERE status = $1 AND age > $2 AND name LIKE $3 AND role IN ($4, $5) AND (department = $6 OR department = $7)
```

### Supported Conditions

| Method | SQL Output | Use Case |
|--------|------------|----------|
| `Equal(field, value)` | `field = $1` | Exact matches |
| `NotEqual(field, value)` | `field != $1` | Exclusions |
| `GreaterThan(field, value)` | `field > $1` | Numeric/date ranges |
| `LessThan(field, value)` | `field < $1` | Numeric/date ranges |
| `Like(field, pattern)` | `field LIKE $1` | Pattern matching |
| `ILike(field, pattern)` | `field ILIKE $1` | Case-insensitive patterns |
| `In(field, values)` | `field IN ($1, $2, ...)` | Multiple value matching |
| `Between(field, min, max)` | `field BETWEEN $1 AND $2` | Range queries |
| `IsNull(field)` | `field IS NULL` | Null checks |
| `IsNotNull(field)` | `field IS NOT NULL` | Non-null checks |

### Combining with SQLc

```go
// Your SQLc query
// -- name: GetUsers :many
// SELECT * FROM users WHERE active = true /* sqld:where */

// Build dynamic conditions
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("department", "engineering")
where.GreaterThan("age", 25)

// Execute with SQLc integration
users, err := sqld.QueryAll[db.User](
    ctx, database, db.GetUsers, sqld.Postgres,
    where, nil, nil, 0,
)
```

### Advanced Grouping and Logic

```go
where := sqld.NewWhereBuilder(sqld.Postgres)

// Complex condition: (status = 'active' AND role IN ('admin', 'manager')) OR (status = 'pending' AND age >= 21)
where.Group(func(w *sqld.WhereBuilder) {
    w.Equal("status", "active")
    w.In("role", []string{"admin", "manager"})
}).Or().Group(func(w *sqld.WhereBuilder) {
    w.Equal("status", "pending")
    w.GreaterThanOrEqual("age", 21)
})
```

[**Learn more about complex conditions →**](#advanced-where-building)

---

## Dynamic Sorting & ORDER BY

Add flexible sorting capabilities to your SQLc queries with multiple syntax options.

### Basic Sorting from HTTP

```http
# Single field sorting
GET /users?sort=name:desc                    # ORDER BY name DESC

# Multiple field sorting  
GET /users?sort=name:asc,created_at:desc     # ORDER BY name ASC, created_at DESC

# Prefix notation
GET /users?sort=-name,+created_at            # ORDER BY name DESC, created_at ASC

# Individual parameters
GET /users?sort_name=desc&sort_age=asc       # ORDER BY name DESC, age ASC
```

### Programmatic Sorting

```go
// Build ORDER BY clauses
orderBy := sqld.NewOrderByBuilder()
orderBy.Desc("created_at")
orderBy.Asc("name") 
orderBy.Add("priority", sqld.SortDesc)

sql := orderBy.Build()
// Result: "created_at DESC, name ASC, priority DESC"
```

### Integration with HTTP Parsing

```go
config := sqld.DefaultConfig().
    WithAllowedFields(map[string]bool{
        "name": true, "created_at": true, "age": true,
    }).
    WithMaxSortFields(3).                    // Security: limit sort complexity
    WithDefaultSort([]sqld.SortField{        // Fallback when no sort specified
        {Field: "created_at", Direction: sqld.SortDesc},
    }).
    WithFieldMappings(map[string]string{     // URL param → DB column mapping
        "signup_date": "created_at",
        "user_name": "name",
    })

// Parse sorting from HTTP request
orderBy, err := sqld.ParseSortFromRequest(request, config)

// Use with SQLc
users, err := sqld.QueryAll[db.User](
    ctx, database, db.SearchUsers, sqld.Postgres,
    nil, nil, orderBy, 20,
)
```

### Supported Sort Syntaxes

| HTTP Parameter | Parsed Result | Description |
|----------------|---------------|-------------|
| `?sort=name:desc` | `name DESC` | Colon notation |
| `?sort=-name` | `name DESC` | Prefix notation (- = desc) |
| `?sort=+name` | `name ASC` | Prefix notation (+ = asc) |
| `?sort_name=desc` | `name DESC` | Individual parameters |
| `?sort=name:desc,age:asc` | `name DESC, age ASC` | Multiple fields |

### Field Mapping Example

```go
// Configure field mappings for user-friendly URLs
config := sqld.DefaultConfig().
    WithFieldMappings(map[string]string{
        "signup": "created_at",      // ?sort=signup:desc → ORDER BY created_at DESC  
        "username": "name",          // ?sort=username:asc → ORDER BY name ASC
        "last_seen": "last_login",   // ?sort=last_seen:desc → ORDER BY last_login DESC
    })

// URL: /users?sort=signup:desc,username:asc
// SQL: ORDER BY created_at DESC, name ASC
```

[**See complete sorting guide →**](#advanced-sorting-patterns)

---

## Cursor-Based Pagination

Efficient pagination that scales to millions of records without OFFSET performance penalties.

### How Cursor Pagination Works

Instead of `OFFSET/LIMIT`, cursor pagination uses the last seen record as a reference point:

```sql
-- Traditional OFFSET (slow on large datasets)
SELECT * FROM users ORDER BY created_at DESC LIMIT 20 OFFSET 10000;

-- Cursor-based (fast regardless of dataset size)  
SELECT * FROM users 
WHERE created_at < $1 OR (created_at = $1 AND id < $2)
ORDER BY created_at DESC, id DESC 
LIMIT 20;
```

### SQLc Integration

```sql
-- name: SearchUsers :many
-- Add cursor injection point
SELECT id, name, email, created_at
FROM users  
WHERE deleted_at IS NULL /* sqld:cursor */ /* sqld:where */
ORDER BY created_at DESC, id DESC /* sqld:limit */;
```

### HTTP API Usage

```http
# First page - no cursor
GET /users?limit=20

# Response includes next_cursor for subsequent pages
{
  "users": [...],
  "has_more": true,
  "next_cursor": "eyJjcmVhdGVkX2F0IjoiMjAyNC0wMS0xNSIsImlkIjoxMjN9"
}

# Next page - use cursor from previous response
GET /users?limit=20&cursor=eyJjcmVhdGVkX2F0IjoiMjAyNC0wMS0xNSIsImlkIjoxMjN9
```

### Implementation Example

```go
func PaginateUsers(w http.ResponseWriter, r *http.Request) {
    // Parse pagination parameters
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 || limit > 100 { limit = 20 }
    
    // Parse cursor
    cursorStr := r.URL.Query().Get("cursor")
    cursor, err := sqld.DecodeCursor(cursorStr)
    if err != nil {
        http.Error(w, "Invalid cursor", http.StatusBadRequest)
        return
    }
    
    // Execute paginated query
    users, err := sqld.QueryAll[db.User](
        ctx, database, db.SearchUsers, sqld.Postgres,
        nil,      // No additional filters
        cursor,   // Pagination cursor
        nil,      // Default sorting
        limit+1,  // +1 to check for more results
    )
    if err != nil {
        http.Error(w, "Query failed", http.StatusInternalServerError)
        return
    }
    
    // Build response with pagination metadata
    result := PaginatedResponse{
        Users:   users[:min(len(users), limit)],
        HasMore: len(users) > limit,
        Limit:   limit,
    }
    
    // Generate next cursor if there are more results
    if result.HasMore {
        lastUser := users[limit-1]
        result.NextCursor = sqld.EncodeCursor(lastUser.CreatedAt, lastUser.ID)
    }
    
    json.NewEncoder(w).Encode(result)
}
```

### Cursor Structure

```go
type Cursor struct {
    CreatedAt interface{} `json:"created_at"`  // Timestamp for ordering
    ID        int32       `json:"id"`          // Unique ID for tie-breaking
}

// Cursors are base64-encoded JSON for URL safety
cursor := sqld.EncodeCursor(user.CreatedAt, user.ID)
// Result: "eyJjcmVhdGVkX2F0IjoiMjAyNC0wMS0xNSIsImlkIjoxMjN9"
```

**Benefits of Cursor Pagination:**
- [+] **Consistent Performance** - O(log n) regardless of page number
- [+] **Real-time Safe** - New records don't affect pagination
- [+] **Scalable** - Works with millions of records
- [+] **Deterministic** - Same results when data changes

[**Learn more about pagination strategies →**](#pagination-best-practices)

---

## Automatic Result Scanning

Eliminate boilerplate scanning code with reflection-based automatic result binding.

### Traditional SQLc Approach

```go
// Traditional SQLc requires manual scanning
rows, err := db.Query(ctx, query, params...)
if err != nil { return nil, err }
defer rows.Close()

var users []User
for rows.Next() {
    var user User
    err := rows.Scan(
        &user.ID, &user.Name, &user.Email, 
        &user.Age, &user.Status, &user.CreatedAt,
    )
    if err != nil { return nil, err }
    users = append(users, user)
}
return users, rows.Err()
```

### sqld Automatic Scanning

```go
// sqld handles all scanning automatically
users, err := sqld.QueryAll[db.User](
    ctx, database, db.SearchUsers, sqld.Postgres,
    where, cursor, orderBy, limit,
)
// Done! Results are fully scanned into []db.User
```

### How It Works

sqld uses Go generics and reflection to automatically:

1. **Analyze the target struct** (`db.User`) at runtime
2. **Create scan destinations** for each struct field  
3. **Execute the query** with proper parameter binding
4. **Scan results** directly into typed structs
5. **Handle errors** with detailed context

### Supported Functions

| Function | Purpose | Returns |
|----------|---------|---------|
| `QueryAll[T]()` | Execute query, scan all results | `[]T, error` |
| `QueryOne[T]()` | Execute query, scan single result | `T, error` |
| `QueryPaginated[T]()` | Execute with pagination metadata | `*PaginatedResult[T], error` |

### Type Safety Maintained

```go
// Compile-time type safety with runtime flexibility
users, err := sqld.QueryAll[db.User](...)        // Returns []db.User
orders, err := sqld.QueryAll[db.Order](...)      // Returns []db.Order  
products, err := sqld.QueryAll[db.Product](...)  // Returns []db.Product

// Compiler catches type mismatches
var wrongType []string = users  // ❌ Compile error
```

### Error Context and Debugging

sqld provides detailed error information for debugging:

```go
users, err := sqld.QueryAll[db.User](ctx, db, query, ...)
if err != nil {
    var queryErr *sqld.QueryError
    if errors.As(err, &queryErr) {
        log.Printf("Query failed: %s", queryErr.Query)
        log.Printf("Parameters: %+v", queryErr.Parameters) 
        log.Printf("Context: %s", queryErr.Context)
        log.Printf("Original error: %v", queryErr.Unwrap())
    }
}
```

### Compatibility with SQLc Types

sqld works seamlessly with all SQLc-generated types including:

- [+] **Basic types** - `string`, `int32`, `bool`, etc.
- [+] **Nullable types** - `pgtype.Text`, `pgtype.Int4`, etc.
- [+] **JSON types** - `pgtype.JSONB`, custom JSON structs
- [+] **Time types** - `time.Time`, `pgtype.Timestamptz`
- [+] **Array types** - `pgtype.TextArray`, etc.
- [+] **Custom types** - Enums, custom SQLc types

[**Learn more about type handling →**](#type-conversion-guide)

---

## Security Features

sqld provides comprehensive protection against common web application vulnerabilities.

### SQL Injection Prevention

**Built-in Protection:**
- [+] **Parameterized queries** - All user input is properly escaped
- [+] **Query validation** - Syntax checking before execution
- [+] **No string concatenation** - Parameters are bound safely
- [+] **Comment sanitization** - Removes dangerous SQL comments

```go
// Safe - parameters are properly bound
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("name", userInput)  // Becomes: name = $1 with parameter binding

// sqld prevents dangerous patterns
where.Raw("name = " + userInput)  // ❌ Detected and rejected
```

### Field Whitelisting

Prevent unauthorized data access by controlling which database fields can be filtered or sorted:

```go
config := sqld.DefaultConfig().
    WithAllowedFields(map[string]bool{
        "name":       true,   // ✅ Allowed
        "email":      true,   // ✅ Allowed  
        "created_at": true,   // ✅ Allowed
        // "password" not listed = ❌ Blocked
        // "internal_notes" not listed = ❌ Blocked
    })

// Only allowed fields can be used in queries
where, err := sqld.FromRequest(request, sqld.Postgres, config)
// URL: ?name=john&password=hack → Only name filter applied, password ignored
```

### Rate Limiting and Abuse Prevention

```go
config := sqld.DefaultConfig().
    WithMaxFilters(10).      // Prevent too many WHERE conditions
    WithMaxSortFields(3).    // Limit ORDER BY complexity
    WithDefaultOperator(sqld.OpEq)  // Safe default for unspecified operators

// Requests exceeding limits are rejected with clear error messages
```

### Input Validation and Sanitization

```go
// Automatic type validation
config := sqld.DefaultConfig().
    WithDateLayout("2006-01-02").  // Strict date format validation
    WithFieldMappings(map[string]string{
        "user_id": "id",  // Control exposed vs internal field names
    })

// Invalid inputs are rejected before reaching the database
// ?age=not_a_number → Error: "invalid value for field age"
// ?date=invalid → Error: "invalid date format"
```

### Structured Error Handling

sqld provides detailed, safe error messages that help developers debug without exposing sensitive information:

```go
type QueryError struct {
    Query      string        // SQL query (sanitized)
    Parameters []interface{} // Parameters (values hidden in production)
    Context    string        // Operation context
    Dialect    Dialect       // Database dialect
    err        error         // Original error (wrapped)
}

// Safe error exposure
if err != nil {
    // Detailed logging for developers
    log.Printf("Query error: %+v", err)
    
    // Safe message for users
    http.Error(w, "Invalid query parameters", http.StatusBadRequest)
}
```

### Production Security Checklist

- [+] **Always use field whitelisting** with `WithAllowedFields()`
- [+] **Set parameter limits** with `WithMaxFilters()` and `WithMaxSortFields()`
- [+] **Validate date formats** with `WithDateLayout()`
- [+] **Use field mappings** to hide internal column names
- [+] **Log errors securely** - full details in logs, safe messages to users
- [+] **Test with malicious input** - SQL injection, oversized requests, etc.

[**Complete security guide →**](#production-security-guide)

---

## Configuration System

sqld uses a unified configuration system that controls filtering, sorting, field mapping, and security settings.

### DefaultConfig()

Start with sensible defaults and customize as needed:

```go
config := sqld.DefaultConfig()
// Default settings:
// - Empty AllowedFields (allows all fields - not recommended for production)
// - Empty FieldMappings
// - DefaultOperator: OpEq (=)
// - DateLayout: "2006-01-02"
// - MaxFilters: 50
// - MaxSortFields: 5
// - DefaultSort: empty (no default sorting)
```

### Fluent Configuration API

Chain configuration methods for readable setup:

```go
config := sqld.DefaultConfig().
    WithAllowedFields(map[string]bool{
        "id": true, "name": true, "email": true, "status": true,
    }).
    WithFieldMappings(map[string]string{
        "user_name": "name",
        "signup_date": "created_at",
    }).
    WithDefaultOperator(sqld.OpEq).
    WithDateLayout("2006-01-02T15:04:05Z07:00").
    WithMaxFilters(20).
    WithMaxSortFields(3).
    WithDefaultSort([]sqld.SortField{
        {Field: "created_at", Direction: sqld.SortDesc},
        {Field: "id", Direction: sqld.SortAsc},
    })
```

### Configuration Options Reference

| Method | Purpose | Example | Security Impact |
|--------|---------|---------|-----------------|
| `WithAllowedFields()` | Whitelist filterable/sortable fields | `{"name": true, "email": true}` | 🔒 **Critical** - Prevents unauthorized field access |
| `WithFieldMappings()` | Map URL params to DB columns | `{"user_name": "name"}` | 🔒 **High** - Hides internal schema |
| `WithMaxFilters()` | Limit number of WHERE conditions | `20` | 🛡️ **Medium** - Prevents DoS attacks |
| `WithMaxSortFields()` | Limit ORDER BY complexity | `3` | 🛡️ **Medium** - Prevents performance abuse |
| `WithDefaultOperator()` | Default comparison operator | `sqld.OpEq` | 🔒 **Low** - Controls unspecified comparisons |
| `WithDateLayout()` | Date parsing format | `"2006-01-02"` | ✅ **Low** - Input validation |
| `WithDefaultSort()` | Fallback sorting | `[{Field: "id", Direction: SortAsc}]` | ✅ **Info** - UX improvement |

### Environment-Specific Configurations

```go
// Development - permissive for debugging
func DevConfig() *sqld.Config {
    return sqld.DefaultConfig().
        WithMaxFilters(100).           // High limits for testing
        WithMaxSortFields(10).
        WithAllowedFields(map[string]bool{
            // Allow more fields for development
        })
}

// Production - strict security
func ProdConfig() *sqld.Config {
    return sqld.DefaultConfig().
        WithAllowedFields(map[string]bool{
            // Explicitly whitelist only necessary fields
            "name": true, "status": true, "created_at": true,
        }).
        WithMaxFilters(10).            // Conservative limits
        WithMaxSortFields(2).
        WithFieldMappings(map[string]string{
            // Hide internal column names
            "signup": "created_at",
        })
}
```

### Reusing Configurations

```go
// Create base configuration
baseConfig := sqld.DefaultConfig().
    WithMaxFilters(20).
    WithMaxSortFields(3)

// Extend for specific endpoints
userConfig := baseConfig.
    WithAllowedFields(map[string]bool{
        "name": true, "email": true, "created_at": true,
    })

orderConfig := baseConfig.
    WithAllowedFields(map[string]bool{
        "total": true, "status": true, "created_at": true,
    })
```

[**Advanced configuration patterns →**](#configuration-best-practices)

---

## Database Support

sqld supports multiple database systems with dialect-specific optimizations.

### Supported Databases

| Database | Status | Dialect Constant | Notes |
|----------|--------|------------------|-------|
| **PostgreSQL** | ✅ Full Support | `sqld.Postgres` | Recommended, most features |
| **MySQL** | ✅ Full Support | `sqld.MySQL` | Complete compatibility |
| **SQLite** | ✅ Full Support | `sqld.SQLite` | Great for testing/development |

### Dialect-Specific Features

```go
// PostgreSQL - supports advanced features
where := sqld.NewWhereBuilder(sqld.Postgres)
where.ILike("name", "john%")           // Case-insensitive LIKE
where.Raw("data->>'key' = ?", "value") // JSON operations

// MySQL - standard SQL features
where := sqld.NewWhereBuilder(sqld.MySQL)  
where.Like("name", "john%")            // Standard LIKE

// SQLite - lightweight, perfect for testing
where := sqld.NewWhereBuilder(sqld.SQLite)
where.Equal("name", "john")            // Basic operations
```

### Connection Examples

**PostgreSQL with pgx:**
```go
import (
    "github.com/jackc/pgx/v5"
    "github.com/getangry/sqld/adapters/pgx"
)

conn, err := pgx.Connect(ctx, databaseURL)
adapter := pgxadapter.NewPgxAdapter(conn)

users, err := sqld.QueryAll[db.User](
    ctx, adapter, db.SearchUsers, sqld.Postgres,
    where, cursor, orderBy, limit,
)
```

**MySQL with database/sql:**
```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

db, err := sql.Open("mysql", connectionString)

users, err := sqld.QueryAll[db.User](
    ctx, db, db.SearchUsers, sqld.MySQL,
    where, cursor, orderBy, limit,
)
```

**SQLite for testing:**
```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

db, err := sql.Open("sqlite3", ":memory:")

users, err := sqld.QueryAll[db.User](
    ctx, db, db.SearchUsers, sqld.SQLite,
    where, cursor, orderBy, limit,
)
```

### Parameter Binding by Dialect

sqld automatically handles parameter placeholder differences:

```go
// Same Go code works across all databases
where := sqld.NewWhereBuilder(dialect)  // dialect determines placeholder style
where.Equal("name", "john")
where.GreaterThan("age", 18)

sql, params := where.Build()

// PostgreSQL: WHERE name = $1 AND age > $2
// MySQL/SQLite: WHERE name = ? AND age > ?
```

### Performance Considerations

| Database | Best For | Performance Notes |
|----------|----------|-------------------|
| **PostgreSQL** | Production apps, complex queries | Excellent JSONB support, advanced indexing |
| **MySQL** | Web applications, read-heavy workloads | Great replication, mature ecosystem |
| **SQLite** | Development, testing, embedded apps | No network overhead, simple deployment |

[**Database-specific optimization guide →**](#database-optimization-guide)

---

## Integration with Existing SQLc Code

sqld is designed to enhance existing SQLc projects without requiring rewrites.

### Step 1: Add Annotations to SQLc Queries

Update your existing `.sql` files to include sqld annotations:

**Before:**
```sql
-- name: GetUsers :many
SELECT id, name, email, status, created_at
FROM users  
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 20;
```

**After:**
```sql
-- name: GetUsers :many  
SELECT id, name, email, status, created_at
FROM users
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC /* sqld:orderby */ /* sqld:limit */;
```

### Step 2: Generate SQLc Code

Run your existing SQLc generation:

```bash
sqlc generate
```

SQLc ignores the `/* sqld:* */` comments, so your generated code remains unchanged.

### Step 3: Replace Manual Queries with sqld

**Before (manual query building):**
```go
func GetUsers(ctx context.Context, db *sql.DB, filters UserFilters) ([]User, error) {
    query := "SELECT id, name, email FROM users WHERE deleted_at IS NULL"
    args := []interface{}{}
    
    if filters.Name != "" {
        query += " AND name ILIKE ?"
        args = append(args, "%"+filters.Name+"%")
    }
    
    if filters.Status != "" {
        query += " AND status = ?"
        args = append(args, filters.Status)
    }
    
    query += " ORDER BY created_at DESC LIMIT ?"
    args = append(args, filters.Limit)
    
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil { return nil, err }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        err := rows.Scan(&user.ID, &user.Name, &user.Email)
        if err != nil { return nil, err }
        users = append(users, user)
    }
    
    return users, rows.Err()
}
```

**After (with sqld):**
```go
func GetUsers(ctx context.Context, db DBTX, r *http.Request) ([]User, error) {
    config := sqld.DefaultConfig().
        WithAllowedFields(map[string]bool{
            "name": true, "status": true, "created_at": true,
        })
    
    where, err := sqld.FromRequest(r, sqld.Postgres, config) 
    if err != nil { return nil, err }
    
    return sqld.QueryAll[User](
        ctx, db, db.GetUsers, sqld.Postgres,
        where, nil, nil, 20,
    )
}
```

### Step 4: Gradual Migration Strategy

You can migrate endpoints gradually without breaking existing functionality:

**1. Start with read-only endpoints**
```go
// Migrate search/list endpoints first
func SearchUsers(w http.ResponseWriter, r *http.Request) {
    // Use sqld for flexible filtering
    users, err := getUsersWithSqld(ctx, db, r)
    // ... handle response
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
    // Keep existing SQLc code for simple cases  
    user, err := queries.GetUser(ctx, userID)
    // ... handle response
}
```

**2. Add new features with sqld**
```go
// New endpoints get sqld from the start
func SearchUsersWithPagination(w http.ResponseWriter, r *http.Request) {
    // Built with sqld pagination, filtering, sorting
}
```

**3. Migrate complex queries last**
```go
// Complex joins and aggregations
// Migrate once you're comfortable with sqld patterns
```

### Maintaining SQLc Benefits

sqld preserves SQLc's key advantages:

- [+] **SQL-first approach** - Write real SQL, not ORM abstractions
- [+] **Compile-time safety** - Type checking for your queries
- [+] **Performance** - No reflection in query generation (only in result scanning)
- [+] **Tooling support** - IDE completion, query analysis tools

### Example: Complete Migration

**Original SQLc handler:**
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    users, err := h.queries.ListUsers(r.Context())
    if err != nil {
        http.Error(w, "Database error", 500)
        return
    }
    json.NewEncoder(w).Encode(users)
}
```

**Enhanced with sqld:**
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    config := h.getUsersConfig()  // Reusable configuration
    
    where, orderBy, err := sqld.FromRequestWithSort(r, sqld.Postgres, config)
    if err != nil {
        http.Error(w, "Invalid query parameters", 400)
        return  
    }
    
    users, err := sqld.QueryAll[db.User](
        r.Context(), h.db, db.ListUsers, sqld.Postgres,
        where, nil, orderBy, 50,
    )
    if err != nil {
        log.Printf("Query failed: %v", err)
        http.Error(w, "Database error", 500)
        return
    }
    
    json.NewEncoder(w).Encode(users)
}
```

Now your endpoint supports:
- `GET /users` - List all users
- `GET /users?name[contains]=john` - Filter by name
- `GET /users?status=active&sort=name:asc` - Filter and sort
- `GET /users?age[gte]=18&department[in]=eng,product` - Complex filtering

[**Complete migration guide →**](#migration-step-by-step)

---

## API Reference

### Core Query Functions

```go
// Execute query with automatic scanning - returns all results
func QueryAll[T any](
    ctx context.Context,
    db DBTX,
    sqlcQuery string,        // SQLc-generated query constant
    dialect Dialect,         // sqld.Postgres, sqld.MySQL, sqld.SQLite
    where *WhereBuilder,     // Dynamic WHERE conditions (optional)
    cursor *Cursor,          // Pagination cursor (optional)  
    orderBy *OrderByBuilder, // Dynamic ORDER BY (optional)
    limit int,               // Result limit (0 = no limit)
    originalParams ...interface{}, // Original SQLc parameters
) ([]T, error)

// Execute query with automatic scanning - returns single result
func QueryOne[T any](
    ctx context.Context,
    db DBTX,
    sqlcQuery string,
    dialect Dialect,
    where *WhereBuilder,
    originalParams ...interface{},
) (T, error)

// Execute query with pagination metadata
func QueryPaginated[T any](
    ctx context.Context,
    db DBTX, 
    sqlcQuery string,
    dialect Dialect,
    where *WhereBuilder,
    cursor *Cursor,
    orderBy *OrderByBuilder,
    limit int,
    getCursorFields func(T) (interface{}, interface{}),
    originalParams ...interface{},
) (*PaginatedResult[T], error)
```

### Configuration API

```go
// Create default configuration
func DefaultConfig() *Config

// Configuration methods (chainable)
func (c *Config) WithAllowedFields(fields map[string]bool) *Config
func (c *Config) WithFieldMappings(mappings map[string]string) *Config  
func (c *Config) WithDefaultOperator(op Operator) *Config
func (c *Config) WithDateLayout(layout string) *Config
func (c *Config) WithMaxFilters(max int) *Config
func (c *Config) WithMaxSortFields(max int) *Config
func (c *Config) WithDefaultSort(sort []SortField) *Config

// Configuration helper methods
func (c *Config) IsFieldAllowed(field string) bool
func (c *Config) MapField(field string) string
func (c *Config) ValidateAndBuild(fields []SortField) (*OrderByBuilder, error)
```

### WhereBuilder API

```go
// Create WHERE builder
func NewWhereBuilder(dialect Dialect) *WhereBuilder

// Basic conditions
func (w *WhereBuilder) Equal(field string, value interface{}) *WhereBuilder
func (w *WhereBuilder) NotEqual(field string, value interface{}) *WhereBuilder
func (w *WhereBuilder) GreaterThan(field string, value interface{}) *WhereBuilder
func (w *WhereBuilder) LessThan(field string, value interface{}) *WhereBuilder
func (w *WhereBuilder) Like(field, pattern string) *WhereBuilder
func (w *WhereBuilder) ILike(field, pattern string) *WhereBuilder
func (w *WhereBuilder) In(field string, values []interface{}) *WhereBuilder
func (w *WhereBuilder) Between(field string, min, max interface{}) *WhereBuilder
func (w *WhereBuilder) IsNull(field string) *WhereBuilder
func (w *WhereBuilder) IsNotNull(field string) *WhereBuilder

// Logical operators
func (w *WhereBuilder) And() *WhereBuilder
func (w *WhereBuilder) Or() *WhereBuilder  
func (w *WhereBuilder) Group(fn func(*WhereBuilder)) *WhereBuilder

// Raw SQL (use carefully)
func (w *WhereBuilder) Raw(condition string, params ...interface{}) *WhereBuilder

// Output
func (w *WhereBuilder) Build() (string, []interface{})
func (w *WhereBuilder) HasConditions() bool
```

### OrderByBuilder API

```go
// Create ORDER BY builder
func NewOrderByBuilder() *OrderByBuilder

// Add sorting fields
func (o *OrderByBuilder) Add(field string, direction SortDirection) *OrderByBuilder
func (o *OrderByBuilder) Asc(field string) *OrderByBuilder
func (o *OrderByBuilder) Desc(field string) *OrderByBuilder

// Output
func (o *OrderByBuilder) Build() string
func (o *OrderByBuilder) BuildWithPrefix() string
func (o *OrderByBuilder) HasFields() bool
func (o *OrderByBuilder) GetFields() []SortField
```

### HTTP Parsing Functions

```go
// Parse filters from HTTP request
func FromRequest(r *http.Request, dialect Dialect, config *Config) (*WhereBuilder, error)

// Parse filters from query string
func FromQueryString(queryString string, dialect Dialect, config *Config) (*WhereBuilder, error)

// Parse sorting from HTTP request  
func ParseSortFromRequest(r *http.Request, config *Config) (*OrderByBuilder, error)

// Parse both filters and sorting
func FromRequestWithSort(r *http.Request, dialect Dialect, config *Config) (*WhereBuilder, *OrderByBuilder, error)
```

### Types and Constants

```go
// Database dialects
const (
    Postgres Dialect = "postgres"
    MySQL    Dialect = "mysql" 
    SQLite   Dialect = "sqlite"
)

// Filter operators
const (
    OpEq       Operator = "="
    OpNe       Operator = "!="
    OpGt       Operator = ">"
    OpGte      Operator = ">="
    OpLt       Operator = "<"
    OpLte      Operator = "<="
    OpLike     Operator = "LIKE"
    OpILike    Operator = "ILIKE"
    OpContains Operator = "contains"
    OpIn       Operator = "in"
    OpNotIn    Operator = "notIn"
    OpBetween  Operator = "between"
    OpIsNull   Operator = "isNull"
    // ... see complete list in source
)

// Sort directions
const (
    SortAsc  SortDirection = "ASC"
    SortDesc SortDirection = "DESC"
)

// Core types
type Config struct { /* configuration fields */ }
type WhereBuilder struct { /* internal fields */ }  
type OrderByBuilder struct { /* internal fields */ }
type Cursor struct {
    CreatedAt interface{} `json:"created_at"`
    ID        int32       `json:"id"`
}
type PaginatedResult[T any] struct {
    Items      []T     `json:"items"`
    NextCursor *string `json:"next_cursor,omitempty"`
    HasMore    bool    `json:"has_more"`
    Limit      int     `json:"limit"`
}
```

---

## Best Practices

### 1. Configuration Management

**[+] DO: Create reusable configurations**
```go
// configs/database.go
func UserQueryConfig() *sqld.Config {
    return sqld.DefaultConfig().
        WithAllowedFields(map[string]bool{
            "name": true, "email": true, "status": true, "created_at": true,
        }).
        WithFieldMappings(map[string]string{
            "signup_date": "created_at",
            "username": "name",
        }).
        WithMaxFilters(10).
        WithMaxSortFields(3)
}
```

**[-] DON'T: Inline configurations everywhere**
```go
// Repeated configuration in every handler - hard to maintain
config := sqld.DefaultConfig().WithAllowedFields(...)
```

### 2. Security First

**[+] DO: Always use field whitelisting in production**
```go
config := sqld.DefaultConfig().
    WithAllowedFields(map[string]bool{
        "name": true,      // [+] Explicitly allowed
        "email": true,     // [+] Explicitly allowed
        // password field not listed = automatically blocked
    })
```

**[-] DON'T: Allow all fields (default behavior)**
```go
config := sqld.DefaultConfig()  // [-] Empty AllowedFields = allows all fields
```

### 3. Error Handling

**[+] DO: Provide context in error messages**
```go
users, err := sqld.QueryAll[db.User](ctx, database, query, ...)
if err != nil {
    var queryErr *sqld.QueryError
    if errors.As(err, &queryErr) {
        // Log detailed error for developers
        log.Printf("Query failed: %s, params: %+v, context: %s", 
            queryErr.Query, queryErr.Parameters, queryErr.Context)
        
        // Return safe error to users
        http.Error(w, "Invalid query parameters", http.StatusBadRequest)
        return
    }
    
    // Handle other error types
    http.Error(w, "Internal server error", http.StatusInternalServerError)
}
```

### 4. Performance Optimization

**[+] DO: Use appropriate limits and pagination**
```go
limit := 50  // Reasonable default
if requestedLimit > 100 {
    limit = 100  // Cap maximum results
}

users, err := sqld.QueryAll[db.User](
    ctx, db, query, sqld.Postgres,
    where, cursor, orderBy, limit,
)
```

**[+] DO: Index fields used for filtering and sorting**
```sql
-- Add database indexes for commonly filtered/sorted fields
CREATE INDEX idx_users_status_created_at ON users(status, created_at);
CREATE INDEX idx_users_name_gin ON users USING gin(name gin_trgm_ops);  -- PostgreSQL full-text search
```

### 5. Testing Strategies

**[+] DO: Test with real database scenarios**
```go
func TestUserFiltering(t *testing.T) {
    db := setupTestDB(t)  // Use real database (SQLite for tests)
    
    // Create test data
    createTestUsers(t, db)
    
    // Test filtering
    config := UserQueryConfig()
    where, err := sqld.FromQueryString("name[contains]=john&status=active", sqld.SQLite, config)
    require.NoError(t, err)
    
    users, err := sqld.QueryAll[User](ctx, db, SearchUsersQuery, sqld.SQLite, where, nil, nil, 50)
    require.NoError(t, err)
    
    // Verify results
    assert.Greater(t, len(users), 0)
    for _, user := range users {
        assert.Contains(t, strings.ToLower(user.Name), "john")
        assert.Equal(t, "active", user.Status)
    }
}
```

### 6. Gradual Adoption

**[+] DO: Migrate incrementally**
```go
// Phase 1: Add sqld to new endpoints
func NewAdvancedSearch(w http.ResponseWriter, r *http.Request) {
    // Use sqld from the start
    users, err := getUsersWithSqld(ctx, db, r)
    // ...
}

// Phase 2: Keep existing simple endpoints unchanged
func GetUserByID(w http.ResponseWriter, r *http.Request) {  
    // Keep using direct SQLc calls for simple cases
    user, err := queries.GetUser(ctx, userID)
    // ...
}

// Phase 3: Gradually replace when adding features
func ListUsersEnhanced(w http.ResponseWriter, r *http.Request) {
    // Replace when adding pagination, filtering, etc.
    users, err := getUsersWithSqld(ctx, db, r)
    // ...
}
```

### 7. Documentation and API Design

**[+] DO: Document your query parameters**
```go
// GET /users - List users with optional filtering and sorting
//
// Query Parameters:
//   name[contains]     - Filter by name containing text
//   status             - Filter by exact status (active, inactive, pending)  
//   age[gte]           - Filter by minimum age
//   sort               - Sort fields: name:asc, created_at:desc, etc.
//   limit              - Maximum results (default: 20, max: 100)
//   cursor             - Pagination cursor for next page
//
// Examples:
//   GET /users?name[contains]=john&status=active&sort=name:asc
//   GET /users?age[gte]=18&sort=created_at:desc&limit=50
func ListUsers(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### 8. Monitoring and Observability

**[+] DO: Add metrics and logging**
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        h.metrics.QueryDuration.Observe(duration.Seconds())
    }()
    
    // Parse query parameters
    where, orderBy, err := sqld.FromRequestWithSort(r, sqld.Postgres, h.config)
    if err != nil {
        h.metrics.QueryErrors.WithLabelValues("invalid_params").Inc()
        http.Error(w, "Invalid query parameters", 400)
        return
    }
    
    // Log query details (be careful with sensitive data)
    h.logger.Debug("executing user query",
        "filters", where.HasConditions(),
        "sorting", orderBy.HasFields(),
        "user_id", getCurrentUserID(r),
    )
    
    // Execute query
    users, err := sqld.QueryAll[db.User](ctx, h.db, db.ListUsers, sqld.Postgres, where, nil, orderBy, 50)
    if err != nil {
        h.metrics.QueryErrors.WithLabelValues("database_error").Inc()
        h.logger.Error("database query failed", "error", err)
        http.Error(w, "Internal server error", 500)
        return
    }
    
    h.metrics.QuerySuccess.Inc()
    h.metrics.ResultCount.Observe(float64(len(users)))
    
    json.NewEncoder(w).Encode(users)
}
```

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/getangry/sqld.git
cd sqld

# Install dependencies
go mod download

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

### Running Examples

```bash
# Start the example application
cd example
go run main.go

# The server starts on :8080 with sample endpoints
# Try: http://localhost:8080/users?name[contains]=john&sort=name:desc
```

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Related Projects

- **[SQLc](https://sqlc.dev)** - Generate type-safe Go code from SQL
- **[pgx](https://github.com/jackc/pgx)** - PostgreSQL driver and toolkit for Go
- **[Squirrel](https://github.com/Masterminds/squirrel)** - SQL query builder for Go
- **[GORM](https://gorm.io)** - Full-featured ORM for Go

---

**sqld** - Enhancing SQLc with dynamic query capabilities

Made with care by the sqld team