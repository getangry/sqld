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
