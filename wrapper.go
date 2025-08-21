package sqld

import (
	"context"
)

// EnhancedQueries wraps any sqlc-generated Queries struct to add dynamic functionality
// This is now primarily used for accessing the underlying database connection
// Most functionality has moved to reflection-based functions
type EnhancedQueries[T any] struct {
	queries T
	db      DBTX
	dialect Dialect
}

// NewEnhanced creates an enhanced wrapper around sqlc-generated queries
func NewEnhanced[T any](queries T, db DBTX, dialect Dialect) *EnhancedQueries[T] {
	return &EnhancedQueries[T]{
		queries: queries,
		db:      db,
		dialect: dialect,
	}
}

// Queries returns the underlying sqlc-generated queries for fallback
func (eq *EnhancedQueries[T]) Queries() T {
	return eq.queries
}

// DB returns the database interface
func (eq *EnhancedQueries[T]) DB() DBTX {
	return eq.db
}

// Dialect returns the database dialect
func (eq *EnhancedQueries[T]) Dialect() Dialect {
	return eq.dialect
}

// SearchWithReflection executes a search query with reflection-based scanning
// This is the preferred method for dynamic queries
func SearchWithReflection[R any](
	ctx context.Context,
	eq *EnhancedQueries[any],
	sqlcQuery string,
	where *WhereBuilder,
	cursor *Cursor,
	orderBy *OrderByBuilder,
	limit int,
	originalParams ...interface{},
) ([]R, error) {
	return QueryAndScanAllReflection[R](
		ctx,
		eq.db,
		sqlcQuery,
		eq.dialect,
		where,
		cursor,
		orderBy,
		limit,
		originalParams...,
	)
}

// LEGACY METHODS - Kept for backward compatibility but discouraged

// DynamicQuery executes a dynamic query with the given conditions
// DEPRECATED: Use QueryAndScanAllReflection instead
func (eq *EnhancedQueries[T]) DynamicQuery(
	ctx context.Context,
	baseQuery string,
	whereConditions *WhereBuilder,
	scanFn func(rows Rows) error,
) error {
	// Validate the base query
	if err := ValidateQuery(baseQuery, eq.dialect); err != nil {
		return WrapQueryError(err, baseQuery, nil, "dynamic query validation")
	}

	qb := NewQueryBuilder(baseQuery, eq.dialect)
	if whereConditions != nil {
		qb.Where(whereConditions)
	}

	query, params := qb.Build()
	rows, err := eq.db.Query(ctx, query, params...)
	if err != nil {
		return WrapQueryError(err, query, params, "dynamic query execution")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Log the close error but don't override the main error
		}
	}()

	if err := scanFn(rows); err != nil {
		return WrapQueryError(err, query, params, "dynamic query scanning")
	}

	return nil
}

// DynamicQueryRow executes a dynamic query that returns a single row
// DEPRECATED: Use QueryAndScanOneReflection instead
func (eq *EnhancedQueries[T]) DynamicQueryRow(
	ctx context.Context,
	baseQuery string,
	whereConditions *WhereBuilder,
) Row {
	// Validate the base query
	if err := ValidateQuery(baseQuery, eq.dialect); err != nil {
		return &ErrorRow{err: WrapQueryError(err, baseQuery, nil, "dynamic query row validation")}
	}

	qb := NewQueryBuilder(baseQuery, eq.dialect)
	if whereConditions != nil {
		qb.Where(whereConditions)
	}

	query, params := qb.Build()
	return eq.db.QueryRow(ctx, query, params...)
}

// ErrorRow represents a row that had an error during creation
type ErrorRow struct {
	err error
}

// Scan returns the error
func (r *ErrorRow) Scan(dest ...interface{}) error {
	return r.err
}