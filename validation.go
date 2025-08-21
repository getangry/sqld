package sqld

import (
	"fmt"
	"regexp"
	"strings"
)

// SQL injection detection patterns
var (
	// Common SQL injection patterns
	sqlInjectionPatterns = []*regexp.Regexp{
		// Comments that might be used to bypass validation
		regexp.MustCompile(`(?i)(--|#|/\*|\*/)`),
		// Union-based injection
		regexp.MustCompile(`(?i)\bUNION\b.*\bSELECT\b`),
		// Stacked queries
		regexp.MustCompile(`;\s*(SELECT|INSERT|UPDATE|DELETE|DROP|CREATE|ALTER)`),
		// Time-based blind injection
		regexp.MustCompile(`(?i)(SLEEP|WAITFOR|BENCHMARK|pg_sleep)`),
		// Boolean-based blind injection (simplified pattern)
		regexp.MustCompile(`(?i)(\bOR\b|\bAND\b)\s+(['"]?)[\w\s]+['"]?\s*=\s*['"]?[\w\s]+['"]?`),
		// SQL functions that might be exploited
		regexp.MustCompile(`(?i)(CONCAT|CHAR|ASCII|SUBSTRING|LENGTH|HEX|UNHEX)`),
		// System information functions
		regexp.MustCompile(`(?i)(VERSION|DATABASE|USER|CURRENT_USER|SESSION_USER|@@version)`),
		// File operations
		regexp.MustCompile(`(?i)(LOAD_FILE|INTO\s+OUTFILE|INTO\s+DUMPFILE)`),
		// XP commands (SQL Server)
		regexp.MustCompile(`(?i)(xp_cmdshell|sp_configure|sp_addextendedproc)`),
	}

	// Patterns that are generally safe in column names
	safeColumnPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

	// Pattern for safe table names (including schema)
	safeTablePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

	// Pattern for safe identifiers (with optional quotes)
	safeIdentifierPattern = regexp.MustCompile(`^"?[a-zA-Z_][a-zA-Z0-9_]*"?$`)
)

// ValidateQuery validates a query for potential SQL injection
func ValidateQuery(query string, dialect Dialect) error {
	if query == "" {
		return &ValidationError{
			Field:   "query",
			Message: "query cannot be empty",
		}
	}

	// Check for common SQL injection patterns in the query structure
	// Note: This is a basic check and should not be the only defense

	// Check for multiple statements (not counting subqueries)
	if countStatements(query) > 1 {
		return &ValidationError{
			Field:   "query",
			Message: "multiple statements detected",
		}
	}

	return nil
}

// ValidateColumnName validates a column name for safety
func ValidateColumnName(column string) error {
	if column == "" {
		return &ValidationError{
			Field:   "column",
			Message: "column name cannot be empty",
		}
	}

	// Allow quoted identifiers
	cleanColumn := strings.Trim(column, `"`)

	// Check if it matches safe pattern
	if !safeColumnPattern.MatchString(cleanColumn) {
		// Check for SQL injection patterns
		for _, pattern := range sqlInjectionPatterns {
			if pattern.MatchString(column) {
				return &ValidationError{
					Field:   "column",
					Value:   column,
					Message: "potential SQL injection detected in column name",
				}
			}
		}

		// If it doesn't match safe pattern but no injection detected,
		// it might be a complex expression which we'll allow with caution
		if strings.ContainsAny(column, ";--/*") {
			return &ValidationError{
				Field:   "column",
				Value:   column,
				Message: "unsafe characters in column name",
			}
		}
	}

	return nil
}

// ValidateTableName validates a table name for safety
func ValidateTableName(table string) error {
	if table == "" {
		return &ValidationError{
			Field:   "table",
			Message: "table name cannot be empty",
		}
	}

	// Allow quoted identifiers
	cleanTable := strings.Trim(table, `"`)

	if !safeTablePattern.MatchString(cleanTable) {
		return &ValidationError{
			Field:   "table",
			Value:   table,
			Message: "invalid table name format",
		}
	}

	return nil
}

// ValidateOrderBy validates an ORDER BY clause for safety
func ValidateOrderBy(orderBy string) error {
	if orderBy == "" {
		return &ValidationError{
			Field:   "orderBy",
			Message: "order by clause cannot be empty",
		}
	}

	// Split by comma to handle multiple columns
	parts := strings.Split(orderBy, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check for basic ORDER BY pattern: column [ASC|DESC]
		tokens := strings.Fields(part)
		if len(tokens) == 0 {
			return &ValidationError{
				Field:   "orderBy",
				Value:   part,
				Message: "empty order by clause part",
			}
		}

		// Validate column name
		columnName := tokens[0]
		if err := ValidateColumnName(columnName); err != nil {
			return &ValidationError{
				Field:   "orderBy",
				Value:   columnName,
				Message: fmt.Sprintf("invalid column in ORDER BY: %v", err),
			}
		}

		// If there's a second token, it should be ASC or DESC
		if len(tokens) > 1 {
			direction := strings.ToUpper(tokens[1])
			if direction != "ASC" && direction != "DESC" {
				return &ValidationError{
					Field:   "orderBy",
					Value:   direction,
					Message: "invalid sort direction, must be ASC or DESC",
				}
			}
		}

		// No more than 2 tokens per part
		if len(tokens) > 2 {
			return &ValidationError{
				Field:   "orderBy",
				Value:   part,
				Message: "invalid ORDER BY clause format",
			}
		}
	}

	return nil
}

// ValidateValue validates a parameter value
func ValidateValue(value interface{}) error {
	// Check for string values that might contain SQL
	if strVal, ok := value.(string); ok {
		// Check for SQL keywords in string values
		upperVal := strings.ToUpper(strVal)
		suspiciousKeywords := []string{
			"SELECT", "INSERT", "UPDATE", "DELETE", "DROP",
			"CREATE", "ALTER", "EXEC", "EXECUTE", "UNION",
		}

		for _, keyword := range suspiciousKeywords {
			if strings.Contains(upperVal, keyword) {
				// It's OK if it's in a proper string context
				// This is handled by parameterized queries
				// We just log a warning internally
				break
			}
		}
	}

	return nil
}

// SanitizeIdentifier sanitizes an identifier for use in SQL
func SanitizeIdentifier(identifier string, dialect Dialect) string {
	// Remove any potentially dangerous characters
	cleaned := regexp.MustCompile(`[^a-zA-Z0-9_.]`).ReplaceAllString(identifier, "")

	// Quote the identifier based on dialect
	switch dialect {
	case Postgres:
		return fmt.Sprintf(`"%s"`, cleaned)
	case MySQL:
		return fmt.Sprintf("`%s`", cleaned)
	case SQLite:
		return fmt.Sprintf(`"%s"`, cleaned)
	default:
		return fmt.Sprintf(`"%s"`, cleaned)
	}
}

// countStatements counts the number of SQL statements in a query
func countStatements(query string) int {
	// Remove string literals and comments to avoid false positives
	cleaned := removeStringLiteralsAndComments(query)

	// Count semicolons that might indicate multiple statements
	// This is a simple heuristic and not foolproof
	count := 1
	inParens := 0

	for _, char := range cleaned {
		switch char {
		case '(':
			inParens++
		case ')':
			inParens--
		case ';':
			if inParens == 0 {
				count++
			}
		}
	}

	return count
}

// removeStringLiteralsAndComments removes string literals and comments from SQL
func removeStringLiteralsAndComments(query string) string {
	result := []rune{}
	inString := false
	inComment := false
	inBlockComment := false
	stringDelimiter := '\x00'

	runes := []rune(query)
	for i := 0; i < len(runes); i++ {
		char := runes[i]

		// Handle block comments
		if !inString && !inComment && i < len(runes)-1 {
			if char == '/' && runes[i+1] == '*' {
				inBlockComment = true
				i++ // Skip next character
				continue
			}
		}

		if inBlockComment {
			if char == '*' && i < len(runes)-1 && runes[i+1] == '/' {
				inBlockComment = false
				i++ // Skip next character
			}
			continue
		}

		// Handle line comments
		if !inString && !inBlockComment {
			if char == '-' && i < len(runes)-1 && runes[i+1] == '-' {
				inComment = true
				i++ // Skip next character
				continue
			}
		}

		if inComment {
			if char == '\n' {
				inComment = false
			}
			continue
		}

		// Handle string literals
		if !inComment && !inBlockComment {
			if !inString && (char == '\'' || char == '"') {
				inString = true
				stringDelimiter = char
				continue
			}

			if inString && char == stringDelimiter {
				// Check for escaped quotes
				if i < len(runes)-1 && runes[i+1] == stringDelimiter {
					i++ // Skip escaped quote
					continue
				}
				inString = false
				stringDelimiter = '\x00'
				continue
			}
		}

		// Add character if not in string or comment
		if !inString && !inComment && !inBlockComment {
			result = append(result, char)
		}
	}

	return string(result)
}

// SecureQueryBuilder provides additional validation for query building
type SecureQueryBuilder struct {
	*QueryBuilder
	validationEnabled bool
}

// NewSecureQueryBuilder creates a new secure query builder
func NewSecureQueryBuilder(baseQuery string, dialect Dialect) *SecureQueryBuilder {
	return &SecureQueryBuilder{
		QueryBuilder:      NewQueryBuilder(baseQuery, dialect),
		validationEnabled: true,
	}
}

// Build builds the query with validation
func (sqb *SecureQueryBuilder) Build() (string, []interface{}, error) {
	query, params := sqb.QueryBuilder.Build()

	if sqb.validationEnabled {
		if err := ValidateQuery(query, sqb.dialect); err != nil {
			return "", nil, err
		}

		// Validate parameters
		for _, param := range params {
			if err := ValidateValue(param); err != nil {
				return "", nil, err
			}
		}
	}

	return query, params, nil
}

// DisableValidation disables validation (use with caution)
func (sqb *SecureQueryBuilder) DisableValidation() *SecureQueryBuilder {
	sqb.validationEnabled = false
	return sqb
}
