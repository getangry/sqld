# sqld - Dynamic Query Enhancement for SQLc

[![Go](https://github.com/getangry/sqld/actions/workflows/go.yml/badge.svg)](https://github.com/getangry/sqld/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/getangry/sqld.svg)](https://pkg.go.dev/github.com/getangry/sqld)
[![Go Report Card](https://goreportcard.com/badge/github.com/getangry/sqld)](https://goreportcard.com/report/github.com/getangry/sqld)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Enhance [SQLc](https://sqlc.dev)-generated code with dynamic query capabilities while preserving SQLc's SQL-first philosophy and compile-time safety.

## Quick Start

Add annotations to your SQLc queries:

```sql
-- name: SearchUsers :many
SELECT * FROM users 
WHERE status = 'active' /* sqld:where */
ORDER BY created_at DESC /* sqld:orderby */ /* sqld:limit */;
```

Execute with dynamic filters, sorting, and pagination:

```go
users, err := sqld.QueryAll[db.User](
    ctx, db, db.SearchUsers, sqld.Postgres,
    where,   // ?name[contains]=john&age[gte]=18
    cursor,  // Pagination cursor
    orderBy, // ?sort=name:desc,created_at:asc
    limit,   // Dynamic limit
)
```

## Installation

```bash
go get github.com/getangry/sqld
```

Requirements: Go 1.21+ and [SQLc](https://sqlc.dev)

## Features

- **Zero rewrites** - Works with existing SQLc code
- **HTTP-first** - Parse URL query params: `?name[contains]=john&age[gte]=18&sort=name:desc`
- **Type-safe** - Maintains compile-time safety with runtime flexibility
- **Security built-in** - Field whitelisting, parameter validation, SQL injection prevention
- **Multiple databases** - PostgreSQL, MySQL, SQLite support

## Basic Usage

### 1. Add annotations to SQLc queries

```sql
-- name: GetUsers :many
SELECT id, name, email, status, created_at
FROM users
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC /* sqld:orderby */ /* sqld:limit */;
```

### 2. Parse HTTP requests

```go
config := sqld.DefaultConfig().WithAllowedFields(map[string]bool{
    "name": true, "status": true, "created_at": true,
})

where, orderBy, err := sqld.FromRequestWithSort(r, sqld.Postgres, config)
```

### 3. Execute queries

```go
users, err := sqld.QueryAll[db.User](
    ctx, database, db.GetUsers, sqld.Postgres,
    where, nil, orderBy, 50,
)
```

## Supported Query Parameters

```http
# Basic filtering
GET /users?name=john&status=active

# Advanced operators
GET /users?name[contains]=john          # ILIKE '%john%'
GET /users?age[gte]=18                  # age >= 18
GET /users?status[in]=active,verified   # IN ('active', 'verified')
GET /users?created_at[between]=2024-01-01,2024-12-31

# Sorting
GET /users?sort=name:desc,created_at:asc
GET /users?sort=-name,+created_at       # Prefix notation

# Pagination
GET /users?limit=20&cursor=eyJpZCI6MTIzfQ==
```

## Configuration

```go
config := sqld.DefaultConfig().
    WithAllowedFields(map[string]bool{
        "name": true, "email": true, "status": true,
    }).
    WithFieldMappings(map[string]string{
        "user_name": "name",
        "signup_date": "created_at",
    }).
    WithMaxFilters(10).
    WithMaxSortFields(3)
```

## Available Annotations

- `/* sqld:where */` - Inject dynamic WHERE conditions
- `/* sqld:orderby */` - Inject dynamic ORDER BY clauses  
- `/* sqld:limit */` - Inject dynamic LIMIT
- `/* sqld:cursor */` - Inject cursor-based pagination conditions

## Core Functions

```go
// Query all results
func QueryAll[T any](ctx, db, sqlcQuery, dialect, where, cursor, orderBy, limit, params...) ([]T, error)

// Query single result  
func QueryOne[T any](ctx, db, sqlcQuery, dialect, where, params...) (T, error)

// Query with pagination metadata
func QueryPaginated[T any](...) (*PaginatedResult[T], error)
```

## Security Features

- **Field whitelisting** - Only allow specified fields
- **Parameter limits** - Prevent DoS with too many filters
- **SQL injection prevention** - All inputs are parameterized
- **Input validation** - Type checking and sanitization

## Database Support

| Database | Status | Dialect |
|----------|--------|---------|
| PostgreSQL | ✅ | `sqld.Postgres` |
| MySQL | ✅ | `sqld.MySQL` |
| SQLite | ✅ | `sqld.SQLite` |

## Example Integration

```go
func ListUsers(w http.ResponseWriter, r *http.Request) {
    config := getUsersConfig() // Reusable config
    
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
        http.Error(w, "Database error", 500)
        return
    }
    
    json.NewEncoder(w).Encode(users)
}
```

Now supports:
- `GET /users` - List all users
- `GET /users?name[contains]=john` - Filter by name
- `GET /users?status=active&sort=name:asc` - Filter and sort
- `GET /users?age[gte]=18&department[in]=eng,product` - Complex filtering

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Related Projects

- [SQLc](https://sqlc.dev) - Generate type-safe Go code from SQL
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver for Go