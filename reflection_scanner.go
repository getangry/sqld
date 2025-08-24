package sqld

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
)

// ReflectionScanner uses reflection to automatically scan database rows into structs
// This eliminates the need to write manual scan functions
type ReflectionScanner[T any] struct {
	structType reflect.Type
}

// NewReflectionScanner creates a new reflection-based scanner for type T
func NewReflectionScanner[T any]() *ReflectionScanner[T] {
	var zero T
	return &ReflectionScanner[T]{
		structType: reflect.TypeOf(zero),
	}
}

// ScanRow scans a database row into a struct using reflection
func (rs *ReflectionScanner[T]) ScanRow(rows Rows) (T, error) {
	var result T
	resultValue := reflect.ValueOf(&result).Elem()

	// Get the number of fields to scan
	numFields := rs.structType.NumField()
	scanDests := make([]interface{}, numFields)

	// Create scan destinations for each field
	for i := 0; i < numFields; i++ {
		field := resultValue.Field(i)
		if field.CanSet() {
			scanDests[i] = field.Addr().Interface()
		} else {
			// Skip unexported fields by providing a dummy destination
			var dummy interface{}
			scanDests[i] = &dummy
		}
	}

	// Scan the row
	if err := rows.Scan(scanDests...); err != nil {
		return result, err
	}

	return result, nil
}

// ScanAll executes a query and scans all results using reflection
func (rs *ReflectionScanner[T]) ScanAll(ctx context.Context, db DBTX, query string, params ...interface{}) ([]T, error) {
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, WrapQueryError(err, query, params, "executing query")
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		item, err := rs.ScanRow(rows)
		if err != nil {
			return nil, WrapQueryError(err, query, params, "scanning row")
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, WrapQueryError(err, query, params, "iterating rows")
	}

	return results, nil
}

// ScanOne executes a query and scans a single result using reflection
func (rs *ReflectionScanner[T]) ScanOne(ctx context.Context, db DBTX, query string, params ...interface{}) (T, error) {
	var zero T
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return zero, WrapQueryError(err, query, params, "executing query")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zero, WrapQueryError(err, query, params, "no rows found")
		}
		return zero, ErrNoRows
	}

	result, err := rs.ScanRow(rows)
	if err != nil {
		return zero, WrapQueryError(err, query, params, "scanning row")
	}

	return result, nil
}

// Generic helper functions that use reflection

// QueryAll executes a query and scans all results automatically using reflection
func QueryAll[T any](
	ctx context.Context,
	db DBTX,
	sqlcQuery string,
	dialect Dialect,
	where *WhereBuilder,
	cursor *Cursor,
	orderBy *OrderByBuilder,
	limit int,
	originalParams ...interface{},
) ([]T, error) {
	// Build the query with annotations
	query, params, err := SearchQuery(sqlcQuery, dialect, where, cursor, orderBy, limit, originalParams...)
	if err != nil {
		return nil, err
	}

	// Use reflection scanner
	scanner := NewReflectionScanner[T]()
	return scanner.ScanAll(ctx, db, query, params...)
}

// QueryOne executes a query and scans a single result automatically using reflection
func QueryOne[T any](
	ctx context.Context,
	db DBTX,
	sqlcQuery string,
	dialect Dialect,
	where *WhereBuilder,
	originalParams ...interface{},
) (T, error) {
	// Build the query with annotations
	query, params, err := SearchQuery(sqlcQuery, dialect, where, nil, nil, 0, originalParams...)
	if err != nil {
		var zero T
		return zero, err
	}

	// Use reflection scanner
	scanner := NewReflectionScanner[T]()
	return scanner.ScanOne(ctx, db, query, params...)
}

// QueryPaginated executes a paginated query with automatic scanning
func QueryPaginated[T any](
	ctx context.Context,
	db DBTX,
	sqlcQuery string,
	dialect Dialect,
	where *WhereBuilder,
	cursor *Cursor,
	orderBy *OrderByBuilder,
	limit int,
	getCursorFields func(T) (interface{}, interface{}), // Returns (timestamp, id) for cursor
	originalParams ...interface{},
) (*PaginatedResult[T], error) {
	// Query for limit+1 to check for more results
	items, err := QueryAll[T](
		ctx, db, sqlcQuery, dialect, where, cursor, orderBy, limit+1, originalParams...,
	)
	if err != nil {
		return nil, err
	}

	result := &PaginatedResult[T]{
		Limit: limit,
	}

	// Check if there are more results
	if len(items) > limit {
		result.HasMore = true
		result.Items = items[:limit]

		// Generate next cursor from last item
		if getCursorFields != nil {
			lastItem := items[limit-1]
			timestamp, id := getCursorFields(lastItem)
			cursorStr := EncodeCursor(timestamp, id)
			result.NextCursor = &cursorStr
		}
	} else {
		result.Items = items
		result.HasMore = false
	}

	return result, nil
}

// PaginatedResult wraps results with pagination metadata
type PaginatedResult[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
	Limit      int     `json:"limit"`
}

// CursorData represents the data stored in a pagination cursor
type CursorData struct {
	Timestamp interface{} `json:"timestamp"`
	ID        interface{} `json:"id"`
}

// EncodeCursor creates a cursor string from timestamp and ID
func EncodeCursor(timestamp interface{}, id interface{}) string {
	cursor := CursorData{
		Timestamp: timestamp,
		ID:        id,
	}
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor parses a cursor string back into components
func DecodeCursor(encoded string) (*Cursor, error) {
	if encoded == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var cursorData CursorData
	if err := json.Unmarshal(data, &cursorData); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	cursor := &Cursor{
		CreatedAt: cursorData.Timestamp,
	}

	if id, ok := cursorData.ID.(float64); ok {
		cursor.ID = int32(id)
	} else if id, ok := cursorData.ID.(int32); ok {
		cursor.ID = id
	}

	return cursor, nil
}
