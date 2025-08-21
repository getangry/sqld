package sqld

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// AnnotatedQuery represents a SQLc query with sqld annotations
type AnnotatedQuery struct {
	OriginalSQL    string
	FilterEnabled  bool
	CursorEnabled  bool
	DefaultLimit   int
	RequiredParams []string // For queries like SearchUsersByStatus that need specific params
}

// AnnotationProcessor processes sqld annotations in SQLc queries
type AnnotationProcessor struct {
	dialect Dialect
}

// NewAnnotationProcessor creates a new annotation processor
func NewAnnotationProcessor(dialect Dialect) *AnnotationProcessor {
	return &AnnotationProcessor{dialect: dialect}
}

// ProcessQuery processes a SQLc query with sqld annotations
func (ap *AnnotationProcessor) ProcessQuery(
	originalSQL string,
	where *WhereBuilder,
	cursor *Cursor,
	orderBy *OrderByBuilder,
	limit int,
	originalParams ...interface{},
) (string, []interface{}, error) {
	sql := originalSQL
	params := make([]interface{}, len(originalParams))
	copy(params, originalParams)

	// Track parameter index for new parameters
	paramIndex := len(params)

	// Build all WHERE conditions first
	var whereConditions []string

	// Add cursor condition if present
	if cursor != nil && strings.Contains(sql, "/* sqld:cursor */") {
		cursorCondition := fmt.Sprintf("(created_at < $%d OR (created_at = $%d AND id < $%d))",
			paramIndex+1, paramIndex+1, paramIndex+2)
		whereConditions = append(whereConditions, cursorCondition)
		params = append(params, cursor.CreatedAt, cursor.ID)
		paramIndex += 2
	}

	// Add dynamic where conditions if present
	if where != nil && where.HasConditions() {
		whereSQL, whereParams := where.Build()
		// Adjust parameter placeholders
		whereSQL = ap.adjustParameterPlaceholders(whereSQL, paramIndex)
		whereConditions = append(whereConditions, whereSQL)
		params = append(params, whereParams...)
		paramIndex += len(whereParams)
	}

	// Replace where annotation with all conditions
	if len(whereConditions) > 0 && strings.Contains(sql, "/* sqld:where */") {
		allConditions := " AND " + strings.Join(whereConditions, " AND ")
		sql = strings.Replace(sql, "/* sqld:where */", allConditions, 1)
	} else {
		// Remove where annotation if no conditions
		sql = strings.Replace(sql, "/* sqld:where */", "", 1)
	}

	// Remove cursor annotation (it's now handled in WHERE clause)
	sql = strings.Replace(sql, "/* sqld:cursor */", "", 1)

	// Process orderby annotation
	if strings.Contains(sql, "/* sqld:orderby */") {
		if orderBy != nil && orderBy.HasFields() {
			// Replace the default ORDER BY with dynamic ordering
			// Find the ORDER BY clause and replace everything before the annotation
			re := regexp.MustCompile(`ORDER BY ([^/]*)(/\* sqld:orderby \*/)`)
			if re.MatchString(sql) {
				// Replace the default ORDER BY fields with dynamic ones
				orderBySQL := orderBy.Build()
				sql = re.ReplaceAllString(sql, "ORDER BY "+orderBySQL+" ")
			}
		} else {
			// No dynamic ordering provided, just remove the annotation
			sql = strings.Replace(sql, "/* sqld:orderby */", "", 1)
		}
	}

	// Process limit annotation
	if limit > 0 && strings.Contains(sql, "/* sqld:limit */") {
		var limitSQL string
		switch ap.dialect {
		case Postgres:
			limitSQL = fmt.Sprintf(" LIMIT $%d", paramIndex+1)
		case MySQL, SQLite:
			limitSQL = " LIMIT ?"
		}
		sql = strings.Replace(sql, "/* sqld:limit */", limitSQL, 1)
		params = append(params, limit)
	} else {
		// Remove limit annotation if no limit
		sql = strings.Replace(sql, "/* sqld:limit */", "", 1)
	}

	return sql, params, nil
}

// adjustParameterPlaceholders adjusts $1, $2, etc. placeholders by an offset
func (ap *AnnotationProcessor) adjustParameterPlaceholders(sql string, offset int) string {
	// Use regex to find and replace parameter placeholders
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(sql, func(match string) string {
		// Extract the number
		numStr := match[1:] // Remove the $
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return match // Return original if can't parse
		}
		return fmt.Sprintf("$%d", num+offset)
	})
}

// Cursor represents a pagination cursor for annotation processing
type Cursor struct {
	CreatedAt interface{} `json:"created_at"`
	ID        int32       `json:"id"`
}

// Example helper functions for common patterns

// SearchQuery builds a search query from SQLc query with annotations
func SearchQuery(
	originalSQL string,
	dialect Dialect,
	where *WhereBuilder,
	cursor *Cursor,
	orderBy *OrderByBuilder,
	limit int,
	originalParams ...interface{},
) (string, []interface{}, error) {
	// DEBUG: Log input parameters
	fmt.Printf("DEBUG SearchQuery: OrderBy hasFields=%v\n", orderBy != nil && orderBy.HasFields())
	if orderBy != nil && orderBy.HasFields() {
		fmt.Printf("DEBUG SearchQuery: OrderBy SQL='%s'\n", orderBy.Build())
	}
	
	processor := NewAnnotationProcessor(dialect)
	finalSQL, params, err := processor.ProcessQuery(originalSQL, where, cursor, orderBy, limit, originalParams...)
	
	// DEBUG: Log final result
	fmt.Printf("DEBUG SearchQuery: Final SQL='%s'\n", finalSQL)
	
	return finalSQL, params, err
}
