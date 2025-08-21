package sqld

import (
	"fmt"
)

// Config is the unified configuration for both filtering and sorting
type Config struct {
	// === FILTERING CONFIGURATION ===
	
	// AllowedFields restricts which fields can be filtered or sorted
	AllowedFields map[string]bool
	
	// FieldMappings maps query parameter names to database column names
	FieldMappings map[string]string
	
	// DefaultOperator is used when no filter operator is specified
	DefaultOperator Operator
	
	// DateLayout for parsing date strings in filters
	DateLayout string
	
	// MaxFilters limits the number of filters to prevent abuse
	MaxFilters int
	
	// === SORTING CONFIGURATION ===
	
	// MaxSortFields limits the number of sort fields to prevent abuse
	MaxSortFields int
	
	// DefaultSort defines the default sorting when no sort is specified
	DefaultSort []SortField
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		AllowedFields:   make(map[string]bool),
		FieldMappings:   make(map[string]string),
		DefaultOperator: OpEq,
		DateLayout:      "2006-01-02",
		MaxFilters:      50,
		MaxSortFields:   5,
		DefaultSort:     []SortField{},
	}
}

// WithAllowedFields sets the allowed fields for both filtering and sorting
func (c *Config) WithAllowedFields(fields map[string]bool) *Config {
	c.AllowedFields = fields
	return c
}

// WithFieldMappings sets the field mappings for both filtering and sorting
func (c *Config) WithFieldMappings(mappings map[string]string) *Config {
	c.FieldMappings = mappings
	return c
}

// WithDefaultOperator sets the default filter operator
func (c *Config) WithDefaultOperator(op Operator) *Config {
	c.DefaultOperator = op
	return c
}

// WithMaxFilters sets the maximum number of filters
func (c *Config) WithMaxFilters(max int) *Config {
	c.MaxFilters = max
	return c
}

// WithMaxSortFields sets the maximum number of sort fields
func (c *Config) WithMaxSortFields(max int) *Config {
	c.MaxSortFields = max
	return c
}

// WithDefaultSort sets the default sorting
func (c *Config) WithDefaultSort(sort []SortField) *Config {
	c.DefaultSort = sort
	return c
}

// WithDateLayout sets the date parsing layout
func (c *Config) WithDateLayout(layout string) *Config {
	c.DateLayout = layout
	return c
}

// HELPER METHODS

// IsFieldAllowed checks if a field is allowed for filtering/sorting
func (c *Config) IsFieldAllowed(field string) bool {
	if len(c.AllowedFields) == 0 {
		// If no allowed fields specified, allow all (not recommended for production)
		return true
	}
	return c.AllowedFields[field]
}

// MapField maps a query parameter field name to the actual database column
func (c *Config) MapField(field string) string {
	if mapped, exists := c.FieldMappings[field]; exists {
		return mapped
	}
	return field
}

// ValidateAndBuild validates sort fields against the config and builds the ORDER BY clause
func (c *Config) ValidateAndBuild(fields []SortField) (*OrderByBuilder, error) {
	if len(fields) > c.MaxSortFields {
		return nil, fmt.Errorf("too many sort fields: %d (max %d)", len(fields), c.MaxSortFields)
	}

	builder := NewOrderByBuilder()

	if len(fields) == 0 {
		for _, defaultField := range c.DefaultSort {
			if c.IsFieldAllowed(defaultField.Field) {
				mappedField := c.MapField(defaultField.Field)
				builder.Add(mappedField, defaultField.Direction)
			}
		}
		return builder, nil
	}

	for _, field := range fields {
		if !c.IsFieldAllowed(field.Field) {
			return nil, fmt.Errorf("field '%s' is not allowed for sorting", field.Field)
		}

		mappedField := c.MapField(field.Field)
		builder.Add(mappedField, field.Direction)
	}

	return builder, nil
}

