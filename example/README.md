# SQLc Integration Example

This example demonstrates how to integrate `sqld` with SQLc-generated code for powerful dynamic query capabilities while maintaining type safety, context support, comprehensive error handling, and built-in security features.

## Project Structure

```
example/
├── sqlc/                  # SQLc configuration and SQL files
│   ├── schema.sql        # Database schema
│   ├── queries.sql       # SQLc queries
│   └── sqlc.yaml        # SQLc configuration
├── generated/            # Generated SQLc code
│   └── db/              # Generated Go types and queries
│       ├── models.go    # Generated structs (User, Post)
│       ├── querier.go   # Generated interface
│       └── queries.sql.go # Generated query methods
├── integration.go       # Full REST API example
├── simple_usage.go     # Basic usage example
└── README.md           # This file
```

## Quick Start with Docker

The fastest way to test the examples is using Docker Compose:

```bash
# Start PostgreSQL and the application
make up

# Or manually:
docker-compose up -d

# Test the API
curl "http://localhost:8080/users?name[contains]=john&age[gt]=25"

# Run comprehensive tests
make test

# View logs
make logs

# Connect to database
make psql

# Clean up
make down
```

See [DOCKER.md](./DOCKER.md) for detailed Docker instructions.

## Manual Setup

### 1. Install Dependencies

```bash
# Install SQLc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install required Go modules
go mod init your-project
go get github.com/getangry/sqld
go get github.com/jackc/pgx/v5
go get github.com/gin-gonic/gin  # For REST API example
```

### 2. Generate SQLc Code

```bash
cd sqlc/
sqlc generate
```

This creates the `generated/db/` directory with type-safe Go code.

### 3. Database Setup

Create a PostgreSQL database and run the schema:

```bash
psql -U your-user -d your-db -f sqlc/schema.sql
```

## Usage Patterns

### Pattern 1: Basic Integration with Context and Error Handling

```go
import (
    "context"
    "errors"
    "log"
    "github.com/getangry/sqld"
)

// Your existing sqlc setup
ctx := context.Background()
conn, _ := pgx.Connect(ctx, "postgres://...")
queries := db.New(conn)

// Enhance with dynamic capabilities and validation
enhanced := sqld.NewEnhanced(queries, conn, sqld.Postgres)

// Use original SQLc methods
user, err := enhanced.Queries().GetUser(ctx, 1)
if err != nil {
    log.Printf("Failed to get user: %v", err)
}

// Use dynamic queries with comprehensive error handling
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Equal("status", "active").GreaterThan("age", 18)

baseQuery := "SELECT id, name, email FROM users"
err = enhanced.DynamicQuery(ctx, baseQuery, where, func(rows sqld.Rows) error {
    for rows.Next() {
        var id int
        var name, email string
        if err := rows.Scan(&id, &name, &email); err != nil {
            return sqld.WrapQueryError(err, baseQuery, nil, "scanning user")
        }
        // Process user
    }
    return nil
})

if err != nil {
    // Handle structured errors
    var qErr *sqld.QueryError
    var vErr *sqld.ValidationError
    
    if errors.As(err, &qErr) {
        log.Printf("Query failed in %s: %v", qErr.Context, qErr.Unwrap())
    } else if errors.As(err, &vErr) {
        log.Printf("Validation failed for %s: %s", vErr.Field, vErr.Message)
    }
}
```

### Pattern 2: HTTP Query Parameter Filtering

```go
// Configure allowed fields and mappings
config := &sqld.QueryFilterConfig{
    AllowedFields: map[string]bool{
        "name": true,
        "email": true, 
        "status": true,
        "age": true,
    },
    FieldMappings: map[string]string{
        "user_name": "name",     // Map user_name param to name column
        "user_age": "age",       // Map user_age param to age column
    },
    DefaultOperator: sqld.OpEq,
    MaxFilters: 10,
}

// Parse from HTTP request
where, err := sqld.FromRequest(r, sqld.Postgres, config)

// Execute query
baseQuery := "SELECT * FROM users"
enhanced.DynamicQuery(ctx, baseQuery, where, scanFunc)
```

### Pattern 3: Transaction Support

```go
import "database/sql"

// Create transaction-enabled wrapper
stdDB := sqld.NewStandardDB(db, sqld.Postgres)
txQueries := sqld.NewTransactionalQueries(queries, enhanced.DB(), sqld.Postgres, stdDB)

// Execute operations within a transaction
err = txQueries.WithTx(ctx, &sqld.TxOptions{
    IsolationLevel: sql.LevelReadCommitted,
}, func(ctx context.Context, queries *sqld.EnhancedQueries[*db.Queries]) error {
    // Update user status
    err := queries.Queries().UpdateUserStatus(ctx, db.UpdateUserStatusParams{
        ID:     userID,
        Status: "verified",
    })
    if err != nil {
        return sqld.WrapQueryError(err, "", nil, "updating user status")
    }
    
    // Add to verified users with dynamic query
    where := sqld.NewWhereBuilder(sqld.Postgres)
    where.Equal("id", userID)
    
    insertQuery := "INSERT INTO verified_users (user_id) VALUES ($1)"
    err = queries.DynamicQuery(ctx, insertQuery, nil, func(rows sqld.Rows) error {
        // Process results if needed
        return nil
    })
    if err != nil {
        return err // Automatic rollback on error
    }
    
    // Transaction commits automatically if no error
    return nil
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### Pattern 4: Mixing Static and Dynamic Queries

```go
// Use SQLc for specific queries
user, err := enhanced.Queries().GetUserByEmail(ctx, "john@example.com")

// Use dynamic for flexible search with security validation
where := sqld.NewWhereBuilder(sqld.Postgres)
where.Like("name", "%john%").In("status", []interface{}{"active", "pending"})

baseQuery := "SELECT * FROM users" 
enhanced.DynamicQuery(ctx, baseQuery, where, scanFunc)
```

## Supported Query Operators

### URL Query Examples

| Operator | URL Example | SQL Result |
|----------|-------------|------------|
| Equal | `?name=john` | `name = 'john'` |
| Not Equal | `?status[ne]=inactive` | `status != 'inactive'` |
| Greater Than | `?age[gt]=18` | `age > 18` |
| Greater/Equal | `?age[gte]=21` | `age >= 21` |
| Less Than | `?age[lt]=65` | `age < 65` |
| Less/Equal | `?age[lte]=64` | `age <= 64` |
| Like/Contains | `?name[contains]=john` | `name ILIKE '%john%'` |
| Starts With | `?name[startswith]=j` | `name ILIKE 'j%'` |
| Ends With | `?name[endswith]=son` | `name ILIKE '%son'` |
| Between | `?age[between]=18,65` | `age BETWEEN 18 AND 65` |
| In | `?status[in]=active,verified` | `status IN ('active', 'verified')` |
| Not In | `?status[notin]=banned,deleted` | `status NOT IN ('banned', 'deleted')` |
| Is Null | `?deleted_at[isnull]=true` | `deleted_at IS NULL` |
| Not Null | `?confirmed_at[isnotnull]=true` | `confirmed_at IS NOT NULL` |

### Syntax Variations

Both bracket and underscore syntax are supported:

```
# Bracket syntax
?name[eq]=john&age[gt]=18

# Underscore syntax  
?name_eq=john&age_gt=18

# Mixed syntax
?name=john&age[gt]=18
```

## Working with SQLc Types

### Handling Nullable Fields

SQLc generates `pgtype` wrappers for nullable database fields:

```go
// In your scan function
var user db.User
err := rows.Scan(
    &user.ID,        // int32
    &user.Name,      // string (NOT NULL)
    &user.Email,     // string (NOT NULL) 
    &user.Age,       // pgtype.Int4 (nullable)
    &user.Status,    // pgtype.Text (nullable)
    &user.Country,   // pgtype.Text (nullable)
    &user.Verified,  // pgtype.Bool (nullable)
    &user.CreatedAt, // pgtype.Timestamp
    &user.UpdatedAt, // pgtype.Timestamp
    &user.DeletedAt, // pgtype.Timestamp (nullable)
)

// Access nullable values
if user.Age.Valid {
    fmt.Printf("User age: %d", user.Age.Int32)
}

if user.Status.Valid {
    fmt.Printf("User status: %s", user.Status.String)
}
```

### JSON Serialization

The generated structs include JSON tags:

```go
type User struct {
    ID        int32            `db:"id" json:"id"`
    Name      string           `db:"name" json:"name"`
    Email     string           `db:"email" json:"email"`
    Age       pgtype.Int4      `db:"age" json:"age"`
    Status    pgtype.Text      `db:"status" json:"status"`
    // ...
}
```

You can serialize directly to JSON:

```go
users := []db.User{...}
jsonData, err := json.Marshal(users)
```

## Security Considerations

1. **Always use AllowedFields** to prevent unauthorized field access:
   ```go
   config := &sqld.QueryFilterConfig{
       AllowedFields: map[string]bool{
           "name": true,
           "email": true,
           // Don't include sensitive fields like "password_hash"
       },
   }
   ```

2. **Use MaxFilters** to prevent abuse:
   ```go
   config.MaxFilters = 10  // Limit number of filters per request
   ```

3. **Field mapping** for API design and security:
   ```go
   config.FieldMappings = map[string]string{
       "user_name": "name",        // API uses user_name, DB uses name
       "signup_date": "created_at", // API uses signup_date, DB uses created_at
   }
   ```

4. **Comprehensive validation** is built-in:
   - SQL injection detection with pattern analysis
   - Column and table name validation
   - ORDER BY clause validation
   - Query structure analysis (prevents multiple statements)
   - Automatic parameter sanitization

5. **Monitor security events**:
   ```go
   func handleQueryError(r *http.Request, err error) {
       var vErr *sqld.ValidationError
       if errors.As(err, &vErr) {
           if strings.Contains(vErr.Message, "injection") {
               log.Printf("SECURITY: Potential injection attempt from %s: %s", 
                   r.RemoteAddr, vErr.Message)
           }
       }
   }
   ```

6. **Use SecureQueryBuilder** for additional validation:
   ```go
   sqb := sqld.NewSecureQueryBuilder(baseQuery, sqld.Postgres)
   query, params, err := sqb.Build() // Comprehensive security validation
   ```

## Performance Tips

1. **Add database indexes** for filtered columns
2. **Use LIMIT/OFFSET** for pagination
3. **Consider caching** for expensive queries
4. **Monitor query performance** with EXPLAIN

## Complete Examples

- `simple_usage.go` - Basic integration pattern
- `integration.go` - Full REST API with all patterns
- Run either example and test with the provided URLs

## Common Patterns

### Search API Endpoint
```go
func SearchUsers(w http.ResponseWriter, r *http.Request) {
    config := &sqld.QueryFilterConfig{...}
    where, err := sqld.FromRequest(r, sqld.Postgres, config)
    // Add business logic filters
    where.IsNull("deleted_at")
    
    baseQuery := "SELECT * FROM users"
    enhanced.DynamicQuery(r.Context(), baseQuery, where, scanFunc)
}
```

### Flexible Updates
```go
func UpdateUser(w http.ResponseWriter, r *http.Request) {
    // Parse JSON body for fields to update
    var updates map[string]interface{}
    json.NewDecoder(r.Body).Decode(&updates)
    
    // Build dynamic UPDATE using sqld
    // ... build SET clauses dynamically
}
```

This integration gives you the best of both worlds: type-safe SQLc queries for standard operations and flexible dynamic queries for complex filtering scenarios.