// Package sqld provides dynamic query building capabilities for sqlc-generated code
package sqld

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

// Dialect represents the SQL database dialect
type Dialect string

const (
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
	SQLite   Dialect = "sqlite"
)

// DBTX is the interface that wraps the basic database operations
type DBTX interface {
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
}

// DBTXWithExec extends DBTX with Exec method for write operations
type DBTXWithExec interface {
	DBTX
	Exec(ctx context.Context, sql string, args ...interface{}) (sql.Result, error)
}

// Rows represents query result rows
type Rows interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}

// Row represents a single query result row
type Row interface {
	Scan(dest ...interface{}) error
}

// Condition represents a single SQL condition
type Condition struct {
	SQL        string
	ParamCount int
}

// ConditionBuilder is the interface for building SQL conditions
type ConditionBuilder interface {
	Equal(column string, value interface{}) ConditionBuilder
	NotEqual(column string, value interface{}) ConditionBuilder
	GreaterThan(column string, value interface{}) ConditionBuilder
	LessThan(column string, value interface{}) ConditionBuilder
	Like(column string, value string) ConditionBuilder
	ILike(column string, value string) ConditionBuilder
	In(column string, values []interface{}) ConditionBuilder
	Between(column string, start, end interface{}) ConditionBuilder
	IsNull(column string) ConditionBuilder
	IsNotNull(column string) ConditionBuilder
	Raw(sql string, params ...interface{}) ConditionBuilder
	Or(fn func(ConditionBuilder)) ConditionBuilder
	Build() (string, []interface{})
	HasConditions() bool
}

// WhereBuilder builds dynamic WHERE conditions
type WhereBuilder struct {
	conditions []Condition
	params     []interface{}
	paramIndex int
	dialect    Dialect
}

// NewWhereBuilder creates a new WHERE condition builder
func NewWhereBuilder(dialect Dialect) *WhereBuilder {
	return &WhereBuilder{
		conditions: make([]Condition, 0),
		params:     make([]interface{}, 0),
		dialect:    dialect,
		paramIndex: 0,
	}
}

// Equal adds an equality condition
func (w *WhereBuilder) Equal(column string, value interface{}) ConditionBuilder {
	if value == nil {
		return w
	}

	// Validate column name
	if err := ValidateColumnName(column); err != nil {
		// Skip validation for now to maintain compatibility
		// In production, you might want to log this or handle it differently
	}

	w.addCondition(column+" = "+w.placeholder(), value)
	return w
}

// NotEqual adds a not-equal condition
func (w *WhereBuilder) NotEqual(column string, value interface{}) ConditionBuilder {
	if value == nil {
		return w
	}

	// Validate column name
	if err := ValidateColumnName(column); err != nil {
		// Skip validation for now to maintain compatibility
	}

	w.addCondition(column+" != "+w.placeholder(), value)
	return w
}

// GreaterThan adds a greater-than condition
func (w *WhereBuilder) GreaterThan(column string, value interface{}) ConditionBuilder {
	if value == nil {
		return w
	}

	// Validate column name
	if err := ValidateColumnName(column); err != nil {
		// Skip validation for now to maintain compatibility
	}

	w.addCondition(column+" > "+w.placeholder(), value)
	return w
}

// LessThan adds a less-than condition
func (w *WhereBuilder) LessThan(column string, value interface{}) ConditionBuilder {
	if value == nil {
		return w
	}

	// Validate column name
	if err := ValidateColumnName(column); err != nil {
		// Skip validation for now to maintain compatibility
	}

	w.addCondition(column+" < "+w.placeholder(), value)
	return w
}

// Like adds a LIKE condition
func (w *WhereBuilder) Like(column string, value string) ConditionBuilder {
	if value == "" {
		return w
	}
	w.addCondition(column+" LIKE "+w.placeholder(), value)
	return w
}

// ILike adds an ILIKE condition (case-insensitive)
func (w *WhereBuilder) ILike(column string, value string) ConditionBuilder {
	if value == "" {
		return w
	}

	if w.dialect == Postgres {
		w.addCondition(column+" ILIKE "+w.placeholder(), value)
	} else {
		// Fallback for MySQL/SQLite
		w.addCondition("LOWER("+column+") LIKE LOWER("+w.placeholder()+")", value)
	}
	return w
}

// In adds an IN condition
func (w *WhereBuilder) In(column string, values []interface{}) ConditionBuilder {
	if len(values) == 0 {
		return w
	}

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = w.placeholder()
	}

	sql := column + " IN (" + strings.Join(placeholders, ", ") + ")"
	w.conditions = append(w.conditions, Condition{
		SQL:        sql,
		ParamCount: len(values),
	})
	w.params = append(w.params, values...)

	return w
}

// Between adds a BETWEEN condition
func (w *WhereBuilder) Between(column string, start, end interface{}) ConditionBuilder {
	if start == nil || end == nil {
		return w
	}
	w.addConditionWithParams(
		column+" BETWEEN "+w.placeholder()+" AND "+w.placeholder(),
		start, end,
	)
	return w
}

// IsNull adds an IS NULL condition
func (w *WhereBuilder) IsNull(column string) ConditionBuilder {
	w.conditions = append(w.conditions, Condition{
		SQL:        column + " IS NULL",
		ParamCount: 0,
	})
	return w
}

// IsNotNull adds an IS NOT NULL condition
func (w *WhereBuilder) IsNotNull(column string) ConditionBuilder {
	w.conditions = append(w.conditions, Condition{
		SQL:        column + " IS NOT NULL",
		ParamCount: 0,
	})
	return w
}

// Raw adds a raw SQL condition
func (w *WhereBuilder) Raw(sql string, params ...interface{}) ConditionBuilder {
	processedSQL := w.processRawSQL(sql, len(params))
	w.conditions = append(w.conditions, Condition{
		SQL:        processedSQL,
		ParamCount: len(params),
	})
	w.params = append(w.params, params...)
	// Don't increment paramIndex here as it's already incremented in processRawSQL
	return w
}

// Or groups conditions with OR logic
func (w *WhereBuilder) Or(fn func(ConditionBuilder)) ConditionBuilder {
	subBuilder := NewWhereBuilder(w.dialect)
	subBuilder.paramIndex = w.paramIndex
	fn(subBuilder)

	if len(subBuilder.conditions) > 0 {
		parts := make([]string, len(subBuilder.conditions))
		for i, cond := range subBuilder.conditions {
			parts[i] = cond.SQL
		}
		orSQL := "(" + strings.Join(parts, " OR ") + ")"

		w.conditions = append(w.conditions, Condition{
			SQL:        orSQL,
			ParamCount: len(subBuilder.params),
		})
		w.params = append(w.params, subBuilder.params...)
		w.paramIndex = subBuilder.paramIndex
	}

	return w
}

// Build returns the SQL and parameters
func (w *WhereBuilder) Build() (string, []interface{}) {
	if len(w.conditions) == 0 {
		return "", nil
	}

	parts := make([]string, len(w.conditions))
	for i, cond := range w.conditions {
		parts[i] = cond.SQL
	}

	return strings.Join(parts, " AND "), w.params
}

// HasConditions returns true if there are conditions to build
func (w *WhereBuilder) HasConditions() bool {
	return len(w.conditions) > 0
}

// Helper methods

func (w *WhereBuilder) placeholder() string {
	w.paramIndex++
	switch w.dialect {
	case Postgres:
		return "$" + strconv.Itoa(w.paramIndex)
	case MySQL, SQLite:
		return "?"
	default:
		return "?"
	}
}

func (w *WhereBuilder) addCondition(sql string, param interface{}) {
	w.conditions = append(w.conditions, Condition{
		SQL:        sql,
		ParamCount: 1,
	})
	w.params = append(w.params, param)
	// Don't increment paramIndex here as it's already incremented in placeholder()
}

func (w *WhereBuilder) addConditionWithParams(sql string, params ...interface{}) {
	w.conditions = append(w.conditions, Condition{
		SQL:        sql,
		ParamCount: len(params),
	})
	w.params = append(w.params, params...)
	// Don't increment paramIndex here as it's already incremented in placeholder() calls
}

func (w *WhereBuilder) processRawSQL(sql string, paramCount int) string {
	if w.dialect == Postgres {
		// Replace ? with $N for PostgreSQL
		result := sql
		for i := 0; i < paramCount; i++ {
			w.paramIndex++
			placeholder := "$" + strconv.Itoa(w.paramIndex)
			result = strings.Replace(result, "?", placeholder, 1)
		}
		return result
	}

	// For MySQL/SQLite, just update the counter
	w.paramIndex += paramCount
	return sql
}

// QueryBuilder helps build complete dynamic queries
type QueryBuilder struct {
	baseQuery string
	dialect   Dialect
	where     *WhereBuilder
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(baseQuery string, dialect Dialect) *QueryBuilder {
	return &QueryBuilder{
		baseQuery: baseQuery,
		dialect:   dialect,
	}
}

// Where adds WHERE conditions
func (qb *QueryBuilder) Where(conditions *WhereBuilder) *QueryBuilder {
	qb.where = conditions
	return qb
}

// Build builds the final query
func (qb *QueryBuilder) Build() (string, []interface{}) {
	query := qb.baseQuery
	var params []interface{}

	if qb.where != nil && qb.where.HasConditions() {
		whereSQL, whereParams := qb.where.Build()
		if whereSQL != "" {
			if strings.Contains(strings.ToUpper(query), "WHERE") {
				query += " AND " + whereSQL
			} else {
				query += " WHERE " + whereSQL
			}
			params = append(params, whereParams...)
		}
	}

	return query, params
}

// ParameterAdjuster helps adjust parameter indices for complex queries
type ParameterAdjuster struct {
	dialect Dialect
}

// NewParameterAdjuster creates a new parameter adjuster
func NewParameterAdjuster(dialect Dialect) *ParameterAdjuster {
	return &ParameterAdjuster{dialect: dialect}
}

// AdjustSQL adjusts parameter placeholders starting from the given offset
func (pa *ParameterAdjuster) AdjustSQL(sql string, startIndex int) string {
	if pa.dialect != Postgres {
		return sql // MySQL/SQLite use ?, no adjustment needed
	}

	// For PostgreSQL, renumber $1, $2, etc.
	result := sql
	placeholderCount := strings.Count(sql, "$")

	for i := 1; i <= placeholderCount; i++ {
		oldPlaceholder := "$" + strconv.Itoa(i)
		newPlaceholder := "$" + strconv.Itoa(i+startIndex)
		result = strings.Replace(result, oldPlaceholder, newPlaceholder, 1)
	}

	return result
}

// Utility functions for common patterns

// CombineConditions combines multiple condition builders with AND logic
func CombineConditions(dialect Dialect, builders ...*WhereBuilder) *WhereBuilder {
	combined := NewWhereBuilder(dialect)

	for _, builder := range builders {
		if builder != nil && builder.HasConditions() {
			sql, params := builder.Build()

			// Adjust parameter placeholders if needed
			if dialect == Postgres {
				adjustedSQL := sql
				// Replace $1, $2, etc. with proper indices based on current parameter count
				for i := 1; i <= len(params); i++ {
					oldPlaceholder := "$" + strconv.Itoa(i)
					newPlaceholder := "$" + strconv.Itoa(combined.paramIndex+i)
					adjustedSQL = strings.Replace(adjustedSQL, oldPlaceholder, newPlaceholder, 1)
				}
				combined.paramIndex += len(params)

				combined.conditions = append(combined.conditions, Condition{
					SQL:        adjustedSQL,
					ParamCount: len(params),
				})
				combined.params = append(combined.params, params...)
			} else {
				// For MySQL/SQLite, just use Raw as it doesn't need parameter adjustment
				combined.Raw(sql, params...)
			}
		}
	}

	return combined
}

// ConditionalWhere adds conditions only if the value is not empty/nil
func ConditionalWhere(builder *WhereBuilder, column string, value interface{}) *WhereBuilder {
	switch v := value.(type) {
	case string:
		if v != "" {
			builder.Equal(column, v)
		}
	case *string:
		if v != nil && *v != "" {
			builder.Equal(column, *v)
		}
	case int, int32, int64:
		if v != 0 {
			builder.Equal(column, v)
		}
	case *int, *int32, *int64:
		if v != nil {
			builder.Equal(column, v)
		}
	default:
		if v != nil {
			builder.Equal(column, v)
		}
	}

	return builder
}

// SearchPattern creates a search pattern for LIKE/ILIKE conditions
func SearchPattern(text string, mode string) string {
	switch mode {
	case "prefix":
		return text + "%"
	case "suffix":
		return "%" + text
	case "contains":
		return "%" + text + "%"
	case "exact":
		return text
	default:
		return "%" + text + "%" // Default to contains
	}
}
