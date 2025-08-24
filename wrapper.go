// Package sqld provides dynamic query enhancement for SQLc-generated code.
// The wrapper.go file contains the Executor pattern for clean, type-safe query execution.
package sqld

import (
	"context"
)

// Queries wraps a database connection with dialect information for simplified sqld usage.
// This eliminates the need to pass the database connection and dialect to every query.
//
// Usage:
//
//	q := sqld.New(database, sqld.Postgres)
//	exec := sqld.NewExecutor[db.User](q)
//	users, err := exec.QueryAll(ctx, db.SearchUsers, where, cursor, orderBy, limit)
type Queries struct {
	db      DBTX
	dialect Dialect
}

// New creates a new Queries wrapper with database and dialect.
// The returned Queries instance can be used to create typed executors
// or passed to the helper functions for query execution.
//
// Example:
//
//	adapter := pgxadapter.NewPgxAdapter(conn)
//	q := sqld.New(adapter, sqld.Postgres)
func New(db DBTX, dialect Dialect) *Queries {
	return &Queries{
		db:      db,
		dialect: dialect,
	}
}

// DB returns the database interface
func (q *Queries) DB() DBTX {
	return q.db
}

// Dialect returns the database dialect
func (q *Queries) Dialect() Dialect {
	return q.dialect
}

// Executor provides a fluent interface for executing queries with a specific type.
// By binding the type at creation time, it eliminates the need to specify the type
// parameter on every query call and provides a cleaner API.
//
// The Executor pattern allows you to:
//   - Create once, use everywhere
//   - Avoid passing database and dialect repeatedly
//   - Get compile-time type safety
//   - Write cleaner, more maintainable code
//
// Example:
//
//	// Setup once in your handler/service
//	q := sqld.New(database, sqld.Postgres)
//	userExec := sqld.NewExecutor[db.User](q)
//
//	// Use throughout your code - clean and simple
//	users, err := userExec.QueryAll(ctx, db.SearchUsers, where, nil, orderBy, 50)
//	user, err := userExec.QueryOne(ctx, db.GetUser, whereClause)
type Executor[T any] struct {
	queries *Queries
}

// NewExecutor creates a typed executor for a specific result type.
// This should typically be created once during initialization and reused.
//
// Example:
//
//	userExec := sqld.NewExecutor[db.User](queries)
//	postExec := sqld.NewExecutor[db.Post](queries)
func NewExecutor[T any](q *Queries) *Executor[T] {
	return &Executor[T]{queries: q}
}

// QueryAll executes a query and scans all results
func (e *Executor[T]) QueryAll(ctx context.Context, sqlcQuery string, where *WhereBuilder, cursor *Cursor, orderBy *OrderByBuilder, limit int, originalParams ...interface{}) ([]T, error) {
	return QueryAll[T](ctx, e.queries.db, sqlcQuery, e.queries.dialect, where, cursor, orderBy, limit, originalParams...)
}

// QueryOne executes a query and scans a single result
func (e *Executor[T]) QueryOne(ctx context.Context, sqlcQuery string, where *WhereBuilder, originalParams ...interface{}) (T, error) {
	return QueryOne[T](ctx, e.queries.db, sqlcQuery, e.queries.dialect, where, originalParams...)
}

// QueryPaginated executes a paginated query
func (e *Executor[T]) QueryPaginated(ctx context.Context, sqlcQuery string, where *WhereBuilder, cursor *Cursor, orderBy *OrderByBuilder, limit int, getCursorFields func(T) (interface{}, interface{}), originalParams ...interface{}) (*PaginatedResult[T], error) {
	return QueryPaginated[T](ctx, e.queries.db, sqlcQuery, e.queries.dialect, where, cursor, orderBy, limit, getCursorFields, originalParams...)
}

// Legacy helper functions for backward compatibility

// QueryAllWith executes a query and scans all results using the Queries wrapper
func QueryAllWith[T any](ctx context.Context, q *Queries, sqlcQuery string, where *WhereBuilder, cursor *Cursor, orderBy *OrderByBuilder, limit int, originalParams ...interface{}) ([]T, error) {
	return QueryAll[T](ctx, q.db, sqlcQuery, q.dialect, where, cursor, orderBy, limit, originalParams...)
}

// QueryOneWith executes a query and scans a single result using the Queries wrapper
func QueryOneWith[T any](ctx context.Context, q *Queries, sqlcQuery string, where *WhereBuilder, originalParams ...interface{}) (T, error) {
	return QueryOne[T](ctx, q.db, sqlcQuery, q.dialect, where, originalParams...)
}

// QueryPaginatedWith executes a paginated query using the Queries wrapper
func QueryPaginatedWith[T any](ctx context.Context, q *Queries, sqlcQuery string, where *WhereBuilder, cursor *Cursor, orderBy *OrderByBuilder, limit int, getCursorFields func(T) (interface{}, interface{}), originalParams ...interface{}) (*PaginatedResult[T], error) {
	return QueryPaginated[T](ctx, q.db, sqlcQuery, q.dialect, where, cursor, orderBy, limit, getCursorFields, originalParams...)
}
