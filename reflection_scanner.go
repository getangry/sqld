package sqld

import (
	"context"
	"database/sql/driver"
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

// QueryAndScanAllReflection executes a query and scans all results automatically using reflection
func QueryAndScanAllReflection[T any](
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

// QueryAndScanOneReflection executes a query and scans a single result automatically using reflection
func QueryAndScanOneReflection[T any](
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

// QueryAndScanPaginatedReflection executes a paginated query with automatic scanning
func QueryAndScanPaginatedReflection[T any](
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
	items, err := QueryAndScanAllReflection[T](
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

// StructScanner is a convenience type for creating scanners with struct field mapping
type StructScanner[T any] struct {
	fieldOrder []string // Order of fields in SELECT statement
	scanner    *ReflectionScanner[T]
}

// NewStructScanner creates a scanner with explicit field ordering
// This is useful when the SELECT field order doesn't match struct field order
func NewStructScanner[T any](fieldOrder []string) *StructScanner[T] {
	return &StructScanner[T]{
		fieldOrder: fieldOrder,
		scanner:    NewReflectionScanner[T](),
	}
}

// ScanRow scans with explicit field ordering
func (ss *StructScanner[T]) ScanRow(rows Rows) (T, error) {
	var result T
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()

	// Create scan destinations based on field order
	scanDests := make([]interface{}, len(ss.fieldOrder))

	for i, fieldName := range ss.fieldOrder {
		_, found := resultType.FieldByName(fieldName)
		if !found {
			// Field not found in struct, use dummy destination
			var dummy interface{}
			scanDests[i] = &dummy
			continue
		}

		// Get the actual field value
		fieldValue := resultValue.FieldByName(fieldName)
		if fieldValue.CanSet() {
			scanDests[i] = fieldValue.Addr().Interface()
		} else {
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

// Helper function to determine if a type implements sql.Scanner
func implementsScanner(t reflect.Type) bool {
	scannerType := reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	return t.Implements(scannerType) || reflect.PtrTo(t).Implements(scannerType)
}

// SmartReflectionScanner is an enhanced scanner that handles complex types better
type SmartReflectionScanner[T any] struct {
	structType   reflect.Type
	fieldMapping map[string]int // Maps struct field names to positions
}

// NewSmartReflectionScanner creates an enhanced reflection scanner
func NewSmartReflectionScanner[T any]() *SmartReflectionScanner[T] {
	var zero T
	structType := reflect.TypeOf(zero)

	// Build field mapping
	fieldMapping := make(map[string]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		// Use db tag if available, otherwise use field name
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			fieldMapping[dbTag] = i
		} else {
			fieldMapping[field.Name] = i
		}
	}

	return &SmartReflectionScanner[T]{
		structType:   structType,
		fieldMapping: fieldMapping,
	}
}

// ScanRow scans with better type handling
func (srs *SmartReflectionScanner[T]) ScanRow(rows Rows) (T, error) {
	var result T
	resultValue := reflect.ValueOf(&result).Elem()

	// Get the number of fields to scan
	numFields := srs.structType.NumField()
	scanDests := make([]interface{}, numFields)

	// Create scan destinations for each field
	for i := 0; i < numFields; i++ {
		field := resultValue.Field(i)
		if field.CanSet() {
			// Handle special types like pgtype
			if field.Kind() == reflect.Interface {
				// For interface{} fields, create a generic destination
				scanDests[i] = field.Addr().Interface()
			} else {
				scanDests[i] = field.Addr().Interface()
			}
		} else {
			// Skip unexported fields
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
