package sqld

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SchemaContentType is the content type for schema discovery requests
const SchemaContentType = "application/vnd.surf+schema"

// FieldSchema describes a field that can be filtered or sorted
type FieldSchema struct {
	// Name is the field name as used in API requests
	Name string `json:"name"`
	
	// DBColumn is the actual database column name (if different from Name)
	DBColumn string `json:"db_column,omitempty"`
	
	// Type indicates the field's data type
	Type string `json:"type"`
	
	// Filterable indicates if this field can be used in filters
	Filterable bool `json:"filterable"`
	
	// Sortable indicates if this field can be used for sorting
	Sortable bool `json:"sortable"`
	
	// Operators lists the allowed filter operators for this field
	Operators []string `json:"operators,omitempty"`
	
	// Description provides human-readable documentation for the field
	Description string `json:"description,omitempty"`
	
	// Example shows an example value for documentation
	Example any `json:"example,omitempty"`
}

// QuerySchema describes the complete query capabilities for an endpoint
type QuerySchema struct {
	// Fields lists all available fields with their capabilities
	Fields []FieldSchema `json:"fields"`
	
	// MaxFilters indicates the maximum number of filters allowed
	MaxFilters int `json:"max_filters,omitempty"`
	
	// MaxSortFields indicates the maximum number of sort fields allowed
	MaxSortFields int `json:"max_sort_fields,omitempty"`
	
	// DefaultSort shows the default sort order if none specified
	DefaultSort []SortField `json:"default_sort,omitempty"`
	
	// SupportsCursor indicates if cursor-based pagination is supported
	SupportsCursor bool `json:"supports_cursor"`
	
	// Examples provides example query strings for documentation
	Examples []QueryExample `json:"examples,omitempty"`
}

// QueryExample provides an example query with description
type QueryExample struct {
	Query       string `json:"query"`
	Description string `json:"description"`
}

// GenerateSchema creates a QuerySchema from a Config
func GenerateSchema(config *Config) *QuerySchema {
	schema := &QuerySchema{
		Fields:         make([]FieldSchema, 0),
		MaxFilters:     config.MaxFilters,
		MaxSortFields:  config.MaxSortFields,
		DefaultSort:    config.DefaultSort,
		SupportsCursor: false, // Can be set based on query annotations
	}
	
	// Determine common operators based on field types
	textOperators := []string{"eq", "ne", "like", "ilike", "contains", "startswith", "endswith", "in", "notin", "isnull", "isnotnull"}
	numberOperators := []string{"eq", "ne", "gt", "gte", "lt", "lte", "between", "in", "notin", "isnull", "isnotnull"}
	boolOperators := []string{"eq", "ne", "isnull", "isnotnull"}
	dateOperators := []string{"eq", "ne", "gt", "gte", "lt", "lte", "between", "isnull", "isnotnull"}
	
	// Build fields from allowed fields
	for field, allowed := range config.AllowedFields {
		if !allowed {
			continue
		}
		
		// Get the database column name (this field is from AllowedFields, so it's the DB name)
		dbColumn := field
		
		// Determine field type and operators based on naming conventions
		// This is a heuristic; real implementation might need type information
		var fieldType string
		var operators []string
		
		switch {
		case strings.HasSuffix(field, "_id") || field == "id":
			fieldType = "integer"
			operators = numberOperators
		case strings.HasSuffix(field, "_at") || strings.Contains(field, "date") || strings.Contains(field, "time"):
			fieldType = "datetime"
			operators = dateOperators
		case strings.HasPrefix(field, "is_") || strings.HasPrefix(field, "has_") || field == "verified" || field == "active":
			fieldType = "boolean"
			operators = boolOperators
		case strings.Contains(field, "age") || strings.Contains(field, "count") || strings.Contains(field, "amount") || strings.Contains(field, "price"):
			fieldType = "number"
			operators = numberOperators
		default:
			fieldType = "string"
			operators = textOperators
		}
		
		// Check if field is sortable (all allowed fields are sortable by default)
		sortable := true
		
		fieldSchema := FieldSchema{
			Name:       field,
			DBColumn:   dbColumn,
			Type:       fieldType,
			Filterable: true,
			Sortable:   sortable,
			Operators:  operators,
		}
		
		// Add descriptions for common fields
		switch field {
		case "id":
			fieldSchema.Description = "Unique identifier"
			fieldSchema.Example = 123
		case "name":
			fieldSchema.Description = "Name field"
			fieldSchema.Example = "John Doe"
		case "email":
			fieldSchema.Description = "Email address"
			fieldSchema.Example = "user@example.com"
		case "status":
			fieldSchema.Description = "Current status"
			fieldSchema.Example = "active"
		case "created_at":
			fieldSchema.Description = "Creation timestamp"
			fieldSchema.Example = "2024-01-01T00:00:00Z"
		case "updated_at":
			fieldSchema.Description = "Last update timestamp"
			fieldSchema.Example = "2024-01-01T00:00:00Z"
		}
		
		schema.Fields = append(schema.Fields, fieldSchema)
	}
	
	// Add dynamic example queries based on available fields
	examples := []QueryExample{}
	
	// Generate examples only using fields that are actually allowed
	hasName := config.AllowedFields["name"]
	hasStatus := config.AllowedFields["status"] 
	hasAge := config.AllowedFields["age"]
	hasCreatedAt := config.AllowedFields["created_at"]
	
	if hasName && hasStatus {
		examples = append(examples, QueryExample{
			Query:       "?name[contains]=john&status=active",
			Description: "Find active users with 'john' in their name",
		})
	}
	
	if hasAge && hasCreatedAt {
		examples = append(examples, QueryExample{
			Query:       "?age[gte]=18&age[lt]=65&sort=-created_at",
			Description: "Find users aged 18-64, sorted by newest first",
		})
	} else if hasAge {
		examples = append(examples, QueryExample{
			Query:       "?age[gte]=18&age[lt]=65",
			Description: "Find users aged 18-64",
		})
	}
	
	if hasStatus && hasName && hasCreatedAt {
		examples = append(examples, QueryExample{
			Query:       "?status[in]=active,verified&sort=name:asc,created_at:desc",
			Description: "Find active or verified users, sorted by name then creation date",
		})
	} else if hasStatus {
		examples = append(examples, QueryExample{
			Query:       "?status[in]=active,verified",
			Description: "Find active or verified users",
		})
	}
	
	// Fallback: if no common fields, create a generic example with any available field
	if len(examples) == 0 && len(schema.Fields) > 0 {
		firstField := schema.Fields[0]
		examples = append(examples, QueryExample{
			Query:       fmt.Sprintf("?%s[eq]=value", firstField.Name),
			Description: fmt.Sprintf("Filter by %s field", firstField.Name),
		})
	}
	
	schema.Examples = examples
	
	return schema
}

// SchemaMiddleware creates a middleware that returns schema for discovery requests
func SchemaMiddleware(config *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client wants schema
			acceptHeader := r.Header.Get("Accept")
			if strings.Contains(acceptHeader, SchemaContentType) {
				// Generate and return schema
				schema := GenerateSchema(config)
				
				// Set response headers
				w.Header().Set("Content-Type", SchemaContentType+"+json")
				w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
				
				// Write schema response
				if err := json.NewEncoder(w).Encode(schema); err != nil {
					http.Error(w, "Failed to encode schema", http.StatusInternalServerError)
					return
				}
				return
			}
			
			// Process normal request
			next.ServeHTTP(w, r)
		})
	}
}

// SchemaHandler creates a standalone handler that returns schema information
func SchemaHandler(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := GenerateSchema(config)
		
		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		
		// Write schema response
		if err := json.NewEncoder(w).Encode(schema); err != nil {
			http.Error(w, "Failed to encode schema", http.StatusInternalServerError)
			return
		}
	}
}

// WithSchema wraps a handler function to support schema discovery
func WithSchema(config *Config, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if client wants schema
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, SchemaContentType) {
			SchemaHandler(config)(w, r)
			return
		}
		
		// Process normal request
		handler(w, r)
	}
}