package sqld

import (
	"context"
	"strings"
)

// EnhancedQueries wraps any sqlc-generated Queries struct to add dynamic functionality
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

// DynamicQuery executes a dynamic query with the given conditions
func (eq *EnhancedQueries[T]) DynamicQuery(
	ctx context.Context,
	baseQuery string,
	whereConditions *WhereBuilder,
	scanFn func(rows Rows) error,
) error {
	qb := NewQueryBuilder(baseQuery, eq.dialect)
	if whereConditions != nil {
		qb.Where(whereConditions)
	}

	query, params := qb.Build()
	rows, err := eq.db.Query(ctx, query, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return scanFn(rows)
}

// DynamicQueryRow executes a dynamic query that returns a single row
func (eq *EnhancedQueries[T]) DynamicQueryRow(
	ctx context.Context,
	baseQuery string,
	whereConditions *WhereBuilder,
) Row {
	qb := NewQueryBuilder(baseQuery, eq.dialect)
	if whereConditions != nil {
		qb.Where(whereConditions)
	}

	query, params := qb.Build()
	return eq.db.QueryRow(ctx, query, params...)
}

// Common query patterns

// SearchQuery builds a search query with text search and filters
func (eq *EnhancedQueries[T]) SearchQuery(
	baseQuery string,
	searchColumns []string,
	searchText string,
	filters *WhereBuilder,
) *WhereBuilder {
	where := NewWhereBuilder(eq.dialect)

	// Add text search across multiple columns
	if searchText != "" && len(searchColumns) > 0 {
		where.Or(func(or ConditionBuilder) {
			for _, column := range searchColumns {
				or.ILike(column, SearchPattern(searchText, "contains"))
			}
		})
	}

	// Combine with additional filters
	if filters != nil && filters.HasConditions() {
		filterSQL, filterParams := filters.Build()
		where.Raw(filterSQL, filterParams...)
	}

	return where
}

// PaginationQuery adds LIMIT/OFFSET to a query
func (eq *EnhancedQueries[T]) PaginationQuery(
	baseQuery string,
	whereConditions *WhereBuilder,
	limit, offset int,
	orderBy string,
) (string, []interface{}) {
	qb := NewQueryBuilder(baseQuery, eq.dialect)
	if whereConditions != nil {
		qb.Where(whereConditions)
	}

	query, params := qb.Build()

	// Add ORDER BY
	if orderBy != "" {
		query += " ORDER BY " + orderBy
	}

	// Add pagination
	if limit > 0 {
		switch eq.dialect {
		case Postgres:
			query += " LIMIT $" + string(rune(len(params)+1))
			params = append(params, limit)
			if offset > 0 {
				query += " OFFSET $" + string(rune(len(params)+1))
				params = append(params, offset)
			}
		case MySQL, SQLite:
			query += " LIMIT ?"
			params = append(params, limit)
			if offset > 0 {
				query += " OFFSET ?"
				params = append(params, offset)
			}
		}
	}

	return query, params
}

// Common scanning helpers

// ScanToSlice scans query results into a slice using a provided scan function
func ScanToSlice[R any](rows Rows, scanFn func(rows Rows) (R, error)) ([]R, error) {
	var results []R

	for rows.Next() {
		item, err := scanFn(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// ScanToMap scans query results into a map using key and value extraction functions
func ScanToMap[K comparable, V any](
	rows Rows,
	keyFn func(rows Rows) (K, error),
	valueFn func(rows Rows) (V, error),
) (map[K]V, error) {
	results := make(map[K]V)

	for rows.Next() {
		key, err := keyFn(rows)
		if err != nil {
			return nil, err
		}

		value, err := valueFn(rows)
		if err != nil {
			return nil, err
		}

		results[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Query builders for common patterns

// BuildDateRangeQuery adds date range conditions
func BuildDateRangeQuery(
	where *WhereBuilder,
	column string,
	startDate, endDate interface{},
) *WhereBuilder {
	if startDate != nil && endDate != nil {
		where.Between(column, startDate, endDate)
	} else if startDate != nil {
		where.GreaterThan(column, startDate)
	} else if endDate != nil {
		where.LessThan(column, endDate)
	}
	return where
}

// BuildStatusFilter adds status filtering with optional exclusions
func BuildStatusFilter(
	where *WhereBuilder,
	column string,
	include []string,
	exclude []string,
) *WhereBuilder {
	if len(include) > 0 {
		includeValues := make([]interface{}, len(include))
		for i, v := range include {
			includeValues[i] = v
		}
		where.In(column, includeValues)
	}

	if len(exclude) > 0 {
		for _, status := range exclude {
			where.NotEqual(column, status)
		}
	}

	return where
}

// BuildFullTextSearch creates a full-text search condition
func BuildFullTextSearch(
	where *WhereBuilder,
	columns []string,
	searchText string,
	dialect Dialect,
) *WhereBuilder {
	if searchText == "" || len(columns) == 0 {
		return where
	}

	// Normalize search text
	searchPattern := SearchPattern(strings.TrimSpace(searchText), "contains")

	where.Or(func(or ConditionBuilder) {
		for _, column := range columns {
			if dialect == Postgres {
				// Use to_tsvector for better full-text search in Postgres
				or.Raw("to_tsvector('english', "+column+") @@ plainto_tsquery('english', ?)", searchText)
			} else {
				or.ILike(column, searchPattern)
			}
		}
	})

	return where
}

// Utility functions for working with existing sqlc code

// InjectWhereCondition injects WHERE conditions into an existing query
func InjectWhereCondition(
	originalQuery string,
	conditions *WhereBuilder,
	dialect Dialect,
) (string, []interface{}) {
	if conditions == nil || !conditions.HasConditions() {
		return originalQuery, nil
	}

	whereSQL, params := conditions.Build()

	// Find the insertion point
	upperQuery := strings.ToUpper(originalQuery)

	// Look for existing WHERE clause
	whereIndex := strings.Index(upperQuery, "WHERE")
	if whereIndex != -1 {
		// Insert after existing WHERE
		insertPoint := whereIndex + 5 // Length of "WHERE"
		// Find the end of the existing WHERE condition
		orderIndex := strings.Index(upperQuery[insertPoint:], "ORDER")
		groupIndex := strings.Index(upperQuery[insertPoint:], "GROUP")
		havingIndex := strings.Index(upperQuery[insertPoint:], "HAVING")
		limitIndex := strings.Index(upperQuery[insertPoint:], "LIMIT")

		// Find the earliest of these indices
		insertAfter := len(originalQuery)
		for _, idx := range []int{orderIndex, groupIndex, havingIndex, limitIndex} {
			if idx != -1 && idx+insertPoint < insertAfter {
				insertAfter = idx + insertPoint
			}
		}

		// Insert the condition
		beforePart := strings.TrimSpace(originalQuery[:insertAfter])
		afterPart := strings.TrimSpace(originalQuery[insertAfter:])

		if afterPart == "" {
			newQuery := beforePart + " AND " + whereSQL
			return newQuery, params
		}
		newQuery := beforePart + " AND " + whereSQL + " " + afterPart
		return newQuery, params
	}

	// No existing WHERE clause, add one
	// Find insertion point before ORDER BY, GROUP BY, etc.
	insertPoint := len(originalQuery)
	for _, keyword := range []string{"ORDER", "GROUP", "HAVING", "LIMIT"} {
		if idx := strings.Index(upperQuery, keyword); idx != -1 && idx < insertPoint {
			insertPoint = idx
		}
	}

	beforePart := strings.TrimSpace(originalQuery[:insertPoint])
	afterPart := strings.TrimSpace(originalQuery[insertPoint:])

	if afterPart == "" {
		// No keywords after
		newQuery := beforePart + " WHERE " + whereSQL
		return newQuery, params
	}
	newQuery := beforePart + " WHERE " + whereSQL + " " + afterPart
	return newQuery, params
}
