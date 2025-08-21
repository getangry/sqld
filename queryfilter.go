package sqld

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Operator represents a filter operator for query string parsing
type Operator string

const (
	OpEq               Operator = "="
	OpNe               Operator = "!="
	OpGt               Operator = ">"
	OpGte              Operator = ">="
	OpLt               Operator = "<"
	OpLte              Operator = "<="
	OpLike             Operator = "LIKE"
	OpILike            Operator = "ILIKE"
	OpContains         Operator = "contains"
	OpIncludes         Operator = "includes"
	OpDoesNotContain   Operator = "doesNotContain"
	OpStartsWith       Operator = "startsWith"
	OpEndsWith         Operator = "endsWith"
	OpDoesNotStartWith Operator = "doesNotStartWith"
	OpDoesNotEndWith   Operator = "doesNotEndWith"
	OpBetween          Operator = "between"
	OpBefore           Operator = "before"
	OpAfter            Operator = "after"
	OpIn               Operator = "in"
	OpNotIn            Operator = "notIn"
	OpIsNull           Operator = "isNull"
	OpIsNotNull        Operator = "isNotNull"
)

// Filter represents a single filter condition from query parameters
type Filter struct {
	Field    string      `json:"field"`
	Operator Operator    `json:"operator"`
	Value    interface{} `json:"value"`
}

// QueryFilterConfig configures how query parameters are parsed
type QueryFilterConfig struct {
	// AllowedFields restricts which fields can be filtered
	AllowedFields map[string]bool
	// FieldMappings maps query parameter names to database column names
	FieldMappings map[string]string
	// DefaultOperator is used when no operator is specified
	DefaultOperator Operator
	// DateLayout for parsing date strings
	DateLayout string
	// MaxFilters limits the number of filters to prevent abuse
	MaxFilters int
}

// DefaultQueryFilterConfig returns a sensible default configuration
func DefaultQueryFilterConfig() *QueryFilterConfig {
	return &QueryFilterConfig{
		AllowedFields:   make(map[string]bool),
		FieldMappings:   make(map[string]string),
		DefaultOperator: OpEq,
		DateLayout:      "2006-01-02",
		MaxFilters:      50,
	}
}

// MapOperator converts string operators to Operator constants
func MapOperator(op string) Operator {
	switch strings.ToLower(op) {
	case "gt":
		return OpGt
	case "gte":
		return OpGte
	case "lt":
		return OpLt
	case "lte":
		return OpLte
	case "ne", "neq":
		return OpNe
	case "sw", "startswith":
		return OpStartsWith
	case "ew", "endswith":
		return OpEndsWith
	case "includes", "contains":
		return OpContains
	case "notcontains", "doesnotcontain":
		return OpDoesNotContain
	case "notstartswith", "doesnotstartswith":
		return OpDoesNotStartWith
	case "notendswith", "doesnotendwith":
		return OpDoesNotEndWith
	case "between":
		return OpBetween
	case "before":
		return OpBefore
	case "after":
		return OpAfter
	case "in":
		return OpIn
	case "notin", "notIn":
		return OpNotIn
	case "isnull", "null":
		return OpIsNull
	case "isnotnull", "notnull":
		return OpIsNotNull
	case "like":
		return OpLike
	case "ilike":
		return OpILike
	default:
		return OpEq
	}
}

// ParseQueryString parses URL query parameters into Filter objects
func ParseQueryString(queryString string, config *QueryFilterConfig) ([]Filter, error) {
	if config == nil {
		config = DefaultQueryFilterConfig()
	}

	values, err := url.ParseQuery(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query string: %w", err)
	}

	return ParseURLValues(values, config)
}

// ParseRequest parses filters from an HTTP request
func ParseRequest(r *http.Request, config *QueryFilterConfig) ([]Filter, error) {
	return ParseURLValues(r.URL.Query(), config)
}

// ParseURLValues parses url.Values into Filter objects
func ParseURLValues(values url.Values, config *QueryFilterConfig) ([]Filter, error) {
	if config == nil {
		config = DefaultQueryFilterConfig()
	}

	var filters []Filter

	for key, vals := range values {
		if len(filters) >= config.MaxFilters {
			return nil, fmt.Errorf("too many filters, maximum allowed: %d", config.MaxFilters)
		}

		// Skip empty values
		if len(vals) == 0 || vals[0] == "" {
			continue
		}

		// Parse the field and operator from the key
		field, operator := parseFieldOperator(key, config.DefaultOperator)

		// Map field name if configured
		if mapped, exists := config.FieldMappings[field]; exists {
			field = mapped
		}

		// Check if field is allowed
		if len(config.AllowedFields) > 0 && !config.AllowedFields[field] {
			continue // Skip disallowed fields
		}

		// Convert value based on operator
		value, err := convertValue(vals[0], operator, config.DateLayout)
		if err != nil {
			return nil, fmt.Errorf("invalid value for field %s: %w", field, err)
		}

		filters = append(filters, Filter{
			Field:    field,
			Operator: operator,
			Value:    value,
		})
	}

	return filters, nil
}

// isValidOperator checks if a string is a valid operator
func isValidOperator(op string) bool {
	validOps := []string{
		"gt", "gte", "lt", "lte", "ne", "neq", "eq",
		"sw", "startswith", "ew", "endswith",
		"contains", "includes", "notcontains", "doesnotcontain",
		"notstartswith", "doesnotstartswith", "notendswith", "doesnotendwith",
		"between", "before", "after", "in", "notin", "notIn",
		"isnull", "null", "isnotnull", "notnull", "like", "ilike",
	}

	opLower := strings.ToLower(op)
	for _, validOp := range validOps {
		if opLower == validOp {
			return true
		}
	}
	return false
}

// parseFieldOperator extracts field name and operator from query parameter key
func parseFieldOperator(key string, defaultOp Operator) (string, Operator) {
	// Support syntax like: name[eq], age[gt], email[contains]
	if strings.Contains(key, "[") && strings.HasSuffix(key, "]") {
		parts := strings.SplitN(key, "[", 2)
		field := parts[0]
		opStr := strings.TrimSuffix(parts[1], "]")
		return field, MapOperator(opStr)
	}

	// Support syntax like: name_eq, age_gt, email_contains
	// But only if the last part is a known operator
	if strings.Contains(key, "_") {
		parts := strings.Split(key, "_")
		if len(parts) >= 2 {
			opStr := parts[len(parts)-1]
			// Only treat as operator syntax if the last part is a valid operator
			if isValidOperator(opStr) {
				field := strings.Join(parts[:len(parts)-1], "_")
				return field, MapOperator(opStr)
			}
		}
	}

	// Default case: just the field name
	return key, defaultOp
}

// convertValue converts string values to appropriate types based on operator
func convertValue(value string, op Operator, dateLayout string) (interface{}, error) {
	switch op {
	case OpBetween:
		// Expect comma-separated values for between
		parts := strings.Split(value, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("between operator requires exactly 2 comma-separated values")
		}
		return []string{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])}, nil

	case OpIn, OpNotIn:
		// Expect comma-separated values
		parts := strings.Split(value, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil

	case OpBefore, OpAfter:
		// Try to parse as date
		if dateLayout != "" {
			if date, err := time.Parse(dateLayout, value); err == nil {
				return date, nil
			}
		}
		return value, nil

	case OpIsNull, OpIsNotNull:
		// These operators don't need a value
		return nil, nil

	case OpGt, OpGte, OpLt, OpLte:
		// Try to parse as number first
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal, nil
		}
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal, nil
		}
		// Fall back to string
		return value, nil

	default:
		return value, nil
	}
}

// ApplyFiltersToBuilder applies parsed filters to a WhereBuilder
func ApplyFiltersToBuilder(filters []Filter, builder *WhereBuilder) error {
	for _, filter := range filters {
		if err := applyFilter(filter, builder); err != nil {
			return fmt.Errorf("failed to apply filter for field %s: %w", filter.Field, err)
		}
	}
	return nil
}

// applyFilter applies a single filter to the WhereBuilder
func applyFilter(filter Filter, builder *WhereBuilder) error {
	field := filter.Field
	value := filter.Value

	switch filter.Operator {
	case OpEq:
		builder.Equal(field, value)

	case OpNe:
		builder.NotEqual(field, value)

	case OpGt:
		builder.GreaterThan(field, value)

	case OpGte:
		builder.Raw(field+" >= ?", value)

	case OpLt:
		builder.LessThan(field, value)

	case OpLte:
		builder.Raw(field+" <= ?", value)

	case OpLike:
		if str, ok := value.(string); ok {
			builder.Like(field, str)
		} else {
			return fmt.Errorf("LIKE operator requires string value")
		}

	case OpILike:
		if str, ok := value.(string); ok {
			builder.ILike(field, str)
		} else {
			return fmt.Errorf("ILIKE operator requires string value")
		}

	case OpContains, OpIncludes:
		if str, ok := value.(string); ok {
			builder.ILike(field, SearchPattern(str, "contains"))
		} else {
			return fmt.Errorf("contains operator requires string value")
		}

	case OpDoesNotContain:
		if str, ok := value.(string); ok {
			builder.Raw("NOT "+field+" ILIKE ?", SearchPattern(str, "contains"))
		} else {
			return fmt.Errorf("doesNotContain operator requires string value")
		}

	case OpStartsWith:
		if str, ok := value.(string); ok {
			builder.ILike(field, SearchPattern(str, "prefix"))
		} else {
			return fmt.Errorf("startsWith operator requires string value")
		}

	case OpEndsWith:
		if str, ok := value.(string); ok {
			builder.ILike(field, SearchPattern(str, "suffix"))
		} else {
			return fmt.Errorf("endsWith operator requires string value")
		}

	case OpDoesNotStartWith:
		if str, ok := value.(string); ok {
			builder.Raw("NOT "+field+" ILIKE ?", SearchPattern(str, "prefix"))
		} else {
			return fmt.Errorf("doesNotStartWith operator requires string value")
		}

	case OpDoesNotEndWith:
		if str, ok := value.(string); ok {
			builder.Raw("NOT "+field+" ILIKE ?", SearchPattern(str, "suffix"))
		} else {
			return fmt.Errorf("doesNotEndWith operator requires string value")
		}

	case OpBetween:
		if vals, ok := value.([]string); ok && len(vals) == 2 {
			builder.Between(field, vals[0], vals[1])
		} else {
			return fmt.Errorf("between operator requires array of 2 values")
		}

	case OpBefore:
		builder.LessThan(field, value)

	case OpAfter:
		builder.GreaterThan(field, value)

	case OpIn:
		if vals, ok := value.([]string); ok {
			interfaces := make([]interface{}, len(vals))
			for i, v := range vals {
				interfaces[i] = v
			}
			builder.In(field, interfaces)
		} else {
			return fmt.Errorf("in operator requires array value")
		}

	case OpNotIn:
		if vals, ok := value.([]string); ok {
			interfaces := make([]interface{}, len(vals))
			for i, v := range vals {
				interfaces[i] = v
			}
			builder.Raw("NOT "+field+" IN (?"+strings.Repeat(",?", len(vals)-1)+")", interfaces...)
		} else {
			return fmt.Errorf("notIn operator requires array value")
		}

	case OpIsNull:
		builder.IsNull(field)

	case OpIsNotNull:
		builder.IsNotNull(field)

	default:
		return fmt.Errorf("unsupported operator: %s", filter.Operator)
	}

	return nil
}

// BuildFromRequest is a convenience function that creates a WhereBuilder from HTTP request
func BuildFromRequest(r *http.Request, dialect Dialect, config *QueryFilterConfig) (*WhereBuilder, error) {
	filters, err := ParseRequest(r, config)
	if err != nil {
		return nil, err
	}

	builder := NewWhereBuilder(dialect)
	err = ApplyFiltersToBuilder(filters, builder)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

// BuildFromQueryString is a convenience function that creates a WhereBuilder from query string
func BuildFromQueryString(queryString string, dialect Dialect, config *QueryFilterConfig) (*WhereBuilder, error) {
	filters, err := ParseQueryString(queryString, config)
	if err != nil {
		return nil, err
	}

	builder := NewWhereBuilder(dialect)
	err = ApplyFiltersToBuilder(filters, builder)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

// FilterToJSON converts filters to JSON for debugging/logging
func FiltersToJSON(filters []Filter) (string, error) {
	data, err := json.MarshalIndent(filters, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
