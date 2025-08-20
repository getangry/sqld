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
  - [Basic WHERE Conditions](#basic-where-conditions)
  - [Complex Conditions](#complex-conditions)
  - [Integration with SQLc](#integration-with-sqlc)
  - [Pagination](#pagination)
  - [Search Filters](#search-filters)
- [API Reference](#api-reference)
- [Database Support](#database-support)
- [Best Practices](#best-practices)
- [Contributing](#contributing)
- [License](#license)

## Features

- 🔒 **Type-safe** - Maintains type safety while building dynamic queries
- 🗄️ **Multi-database support** - Works with PostgreSQL, MySQL, and SQLite
- 🛡️ **SQL injection prevention** - All parameters are properly escaped and parameterized
- 🔧 **SQLc integration** - Seamlessly enhances existing sqlc-generated code
- 🎯 **Zero dependencies** - Only depends on standard library (test dependencies excluded)
- ⚡ **High performance** - Minimal overhead, no reflection or runtime parsing
- 🧩 **Composable** - Build complex queries by combining simple conditions

## Installation

```bash
go get github.com/getangry/sqld
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/getangry/sqld"
    // Your sqlc-generated package
    "yourproject/db"
)

func main() {
    // Create a WHERE builder
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // Add conditions
    where.Equal("status", "active")
    where.GreaterThan("age", 18)
    where.ILike("name", "%john%")
    
    // Build the SQL and get parameters
    sql, params := where.Build()
    fmt.Printf("SQL: %s\n", sql)
    // Output: SQL: status = $1 AND age > $2 AND name ILIKE $3
    fmt.Printf("Params: %v\n", params)
    // Output: Params: [active 18 %john%]
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

### Enhanced Queries

Wrap your sqlc-generated queries with enhanced capabilities:

```go
// Your existing sqlc setup
db := pgx.Connect(...)
queries := db.New(db)

// Enhance with dynamic capabilities
enhanced := sqld.NewEnhanced(queries, db, sqld.Postgres)

// Use dynamic queries alongside your generated ones
enhanced.Queries() // Access original sqlc queries
```

## Usage Examples

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

### Pagination

```go
func (s *UserService) GetUsersPage(ctx context.Context, page, pageSize int) ([]User, error) {
    baseQuery := `SELECT * FROM users`
    
    where := sqld.NewWhereBuilder(sqld.Postgres)
    where.Equal("status", "active")
    
    // Add pagination
    offset := (page - 1) * pageSize
    query, params := s.enhanced.PaginationQuery(
        baseQuery, 
        where, 
        pageSize, 
        offset, 
        "created_at DESC",
    )
    
    // Execute query
    rows, err := s.enhanced.DB().Query(ctx, query, params...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return sqld.ScanToSlice(rows, scanUser)
}
```

### Search Filters

```go
// Conditional filtering - only add conditions if values are provided
func BuildUserFilter(filter UserSearchRequest) *sqld.WhereBuilder {
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // ConditionalWhere only adds the condition if the value is not empty/nil
    sqld.ConditionalWhere(where, "name", filter.Name)
    sqld.ConditionalWhere(where, "email", filter.Email)
    sqld.ConditionalWhere(where, "country", filter.Country)
    
    // Date range filtering
    sqld.BuildDateRangeQuery(where, "created_at", filter.StartDate, filter.EndDate)
    
    // Status filtering with exclusions
    sqld.BuildStatusFilter(
        where, 
        "status",
        filter.IncludeStatuses,  // []string{"active", "pending"}
        filter.ExcludeStatuses,  // []string{"deleted", "banned"}
    )
    
    // Full-text search across multiple columns
    if filter.SearchText != "" {
        sqld.BuildFullTextSearch(
            where,
            []string{"name", "email", "bio"},
            filter.SearchText,
            sqld.Postgres,
        )
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
| `BuildDateRangeQuery(builder, column, start, end)` | Adds date range conditions |
| `BuildStatusFilter(builder, column, include, exclude)` | Adds status filtering |
| `BuildFullTextSearch(builder, columns, text, dialect)` | Adds full-text search |

### EnhancedQueries Methods

| Method | Description |
|--------|-------------|
| `DynamicQuery(ctx, baseQuery, where, scanFn)` | Executes a dynamic query with conditions |
| `DynamicQueryRow(ctx, baseQuery, where)` | Executes a query returning single row |
| `SearchQuery(baseQuery, columns, text, filters)` | Builds a search query |
| `PaginationQuery(baseQuery, where, limit, offset, orderBy)` | Adds pagination to query |

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

### 1. Validate Input

Always validate user input before building queries:

```go
func BuildFilter(userInput UserInput) (*sqld.WhereBuilder, error) {
    // Validate input first
    if err := userInput.Validate(); err != nil {
        return nil, err
    }
    
    where := sqld.NewWhereBuilder(sqld.Postgres)
    
    // Sanitize text input
    if userInput.Search != "" {
        sanitized := strings.TrimSpace(userInput.Search)
        where.ILike("name", sqld.SearchPattern(sanitized, "contains"))
    }
    
    return where, nil
}
```

### 2. Use ConditionalWhere for Optional Filters

```go
// Good - automatically skips empty values
sqld.ConditionalWhere(where, "name", filter.Name)
sqld.ConditionalWhere(where, "email", filter.Email)

// Less optimal - manual checking
if filter.Name != "" {
    where.Equal("name", filter.Name)
}
if filter.Email != "" {
    where.Equal("email", filter.Email)
}
```

### 3. Limit IN Clause Size

```go
// Prevent extremely large IN clauses
if len(ids) > 1000 {
    return errors.New("too many IDs in filter")
}

values := make([]interface{}, len(ids))
for i, id := range ids {
    values[i] = id
}
where.In("id", values)
```

### 4. Use Transactions for Complex Operations

```go
tx, err := db.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

enhanced := sqld.NewEnhanced(queries.WithTx(tx), tx, sqld.Postgres)
// Perform operations...

return tx.Commit(ctx)
```

### 5. Index Columns Used in WHERE Clauses

Ensure your database has appropriate indexes for columns frequently used in dynamic queries:

```sql
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_email ON users(email);
```

## Testing

Run the test suite:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

Run specific tests:

```bash
go test -run TestWhereBuilder
```

## Performance Considerations

- **No Reflection**: sqld uses direct type assertions and explicit interfaces
- **Minimal Allocations**: Builders reuse internal slices where possible
- **Prepared Statements**: All queries use parameterized placeholders
- **Zero Dependencies**: No external libraries means minimal overhead

## Troubleshooting

### Common Issues

**Issue**: Parameters are numbered incorrectly in PostgreSQL
```go
// Solution: Ensure you're using the correct dialect
where := sqld.NewWhereBuilder(sqld.Postgres) // Not MySQL or SQLite
```

**Issue**: ILIKE not working in MySQL
```go
// sqld automatically converts ILIKE to LOWER() LIKE LOWER() for MySQL
// This is handled transparently
```

**Issue**: Combining conditions from multiple builders
```go
// Use CombineConditions for proper parameter adjustment
combined := sqld.CombineConditions(sqld.Postgres, builder1, builder2)
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