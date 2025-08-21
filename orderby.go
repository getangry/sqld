package sqld

import (
	"fmt"
	"strings"
)

// SortDirection represents the direction of sorting
type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

// SortField represents a single field to sort by
type SortField struct {
	Field     string        `json:"field"`
	Direction SortDirection `json:"direction"`
}

// OrderByBuilder builds ORDER BY clauses dynamically
type OrderByBuilder struct {
	fields []SortField
}

// NewOrderByBuilder creates a new OrderByBuilder
func NewOrderByBuilder() *OrderByBuilder {
	return &OrderByBuilder{
		fields: make([]SortField, 0),
	}
}

// Add adds a sort field with the specified direction
func (ob *OrderByBuilder) Add(field string, direction SortDirection) *OrderByBuilder {
	ob.fields = append(ob.fields, SortField{
		Field:     field,
		Direction: direction,
	})
	return ob
}

// Asc adds a field to sort by in ascending order
func (ob *OrderByBuilder) Asc(field string) *OrderByBuilder {
	return ob.Add(field, SortAsc)
}

// Desc adds a field to sort by in descending order
func (ob *OrderByBuilder) Desc(field string) *OrderByBuilder {
	return ob.Add(field, SortDesc)
}

// Clear removes all sort fields
func (ob *OrderByBuilder) Clear() *OrderByBuilder {
	ob.fields = make([]SortField, 0)
	return ob
}

// HasFields returns true if any sort fields are defined
func (ob *OrderByBuilder) HasFields() bool {
	return len(ob.fields) > 0
}

// GetFields returns a copy of the sort fields
func (ob *OrderByBuilder) GetFields() []SortField {
	result := make([]SortField, len(ob.fields))
	copy(result, ob.fields)
	return result
}

// Build generates the ORDER BY SQL clause
func (ob *OrderByBuilder) Build() string {
	if len(ob.fields) == 0 {
		return ""
	}

	var clauses []string
	for _, field := range ob.fields {
		clause := fmt.Sprintf("%s %s", field.Field, field.Direction)
		clauses = append(clauses, clause)
	}

	return strings.Join(clauses, ", ")
}

// BuildWithPrefix generates the ORDER BY SQL clause with "ORDER BY" prefix
func (ob *OrderByBuilder) BuildWithPrefix() string {
	clause := ob.Build()
	if clause == "" {
		return ""
	}
	return "ORDER BY " + clause
}

// OrderByConfig configures sorting behavior and security
type OrderByConfig struct {
	// AllowedFields is a whitelist of fields that can be sorted by
	AllowedFields map[string]bool

	// FieldMappings maps query parameter field names to actual database column names
	FieldMappings map[string]string

	// DefaultSort specifies the default sort when no sort is provided
	DefaultSort []SortField

	// MaxSortFields limits the number of sort fields to prevent abuse
	MaxSortFields int
}

// DefaultOrderByConfig returns a safe default configuration
func DefaultOrderByConfig() *OrderByConfig {
	return &OrderByConfig{
		AllowedFields: make(map[string]bool),
		FieldMappings: make(map[string]string),
		DefaultSort:   []SortField{},
		MaxSortFields: 5, // Reasonable default
	}
}

// IsFieldAllowed checks if a field is allowed for sorting
func (config *OrderByConfig) IsFieldAllowed(field string) bool {
	if len(config.AllowedFields) == 0 {
		// If no allowed fields specified, allow all (not recommended for production)
		return true
	}
	return config.AllowedFields[field]
}

// MapField maps a query parameter field name to the actual database column
func (config *OrderByConfig) MapField(field string) string {
	if mapped, exists := config.FieldMappings[field]; exists {
		return mapped
	}
	return field
}

// ValidateAndBuild validates sort fields against the config and builds the ORDER BY clause
func (config *OrderByConfig) ValidateAndBuild(fields []SortField) (*OrderByBuilder, error) {
	if len(fields) > config.MaxSortFields {
		return nil, fmt.Errorf("too many sort fields: %d (max %d)", len(fields), config.MaxSortFields)
	}

	builder := NewOrderByBuilder()

	// If no fields provided, use default sort
	if len(fields) == 0 {
		for _, defaultField := range config.DefaultSort {
			if config.IsFieldAllowed(defaultField.Field) {
				mappedField := config.MapField(defaultField.Field)
				builder.Add(mappedField, defaultField.Direction)
			}
		}
		return builder, nil
	}

	// Validate and add each field
	for _, field := range fields {
		if !config.IsFieldAllowed(field.Field) {
			return nil, fmt.Errorf("field '%s' is not allowed for sorting", field.Field)
		}

		mappedField := config.MapField(field.Field)
		builder.Add(mappedField, field.Direction)
	}

	return builder, nil
}

// ParseSortDirection converts a string to SortDirection
func ParseSortDirection(dir string) SortDirection {
	switch strings.ToUpper(strings.TrimSpace(dir)) {
	case "DESC", "DESCENDING", "-", "D":
		return SortDesc
	default:
		return SortAsc
	}
}

// SortFieldFromString parses a sort field from string formats like:
// - "name" (ascending)
// - "name:desc"
// - "name:asc"
// - "-name" (descending)
// - "+name" (ascending)
func SortFieldFromString(s string) SortField {
	s = strings.TrimSpace(s)

	// Handle prefix notation: -name, +name
	if strings.HasPrefix(s, "-") {
		return SortField{
			Field:     s[1:],
			Direction: SortDesc,
		}
	}
	if strings.HasPrefix(s, "+") {
		return SortField{
			Field:     s[1:],
			Direction: SortAsc,
		}
	}

	// Handle colon notation: name:desc, name:asc
	parts := strings.Split(s, ":")
	field := parts[0]
	direction := SortAsc

	if len(parts) > 1 {
		direction = ParseSortDirection(parts[1])
	}

	return SortField{
		Field:     field,
		Direction: direction,
	}
}

// ParseSortFields parses multiple sort fields from common formats:
// - Comma-separated: "name:desc,email:asc"
// - Array format: []string{"name:desc", "email:asc"}
func ParseSortFields(input interface{}) []SortField {
	var fields []SortField

	switch v := input.(type) {
	case string:
		if v == "" {
			return fields
		}
		// Split by comma and parse each field
		parts := strings.Split(v, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				fields = append(fields, SortFieldFromString(part))
			}
		}
	case []string:
		for _, field := range v {
			field = strings.TrimSpace(field)
			if field != "" {
				fields = append(fields, SortFieldFromString(field))
			}
		}
	}

	return fields
}
