package sqld

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSchema(t *testing.T) {
	config := DefaultConfig().
		WithAllowedFields(map[string]bool{
			"id":          true,
			"name":        true,
			"email":       true,
			"age":         true,
			"status":      true,
			"created_at":  true,
			"is_verified": true,
		}).
		WithFieldMappings(map[string]string{
			"user_name": "name",
			"signup":    "created_at",
		}).
		WithMaxFilters(10).
		WithMaxSortFields(3)

	schema := GenerateSchema(config)

	// Test basic schema structure
	assert.NotNil(t, schema)
	assert.Equal(t, 10, schema.MaxFilters)
	assert.Equal(t, 3, schema.MaxSortFields)
	assert.False(t, schema.SupportsCursor) // Default value
	assert.Len(t, schema.Examples, 3)      // Should have example queries

	// Test that we have the right number of fields
	assert.Len(t, schema.Fields, 7) // All allowed fields

	// Find specific fields and test their properties
	fieldMap := make(map[string]FieldSchema)
	for _, field := range schema.Fields {
		fieldMap[field.Name] = field
	}

	// Test ID field (integer type with number operators)
	idField, exists := fieldMap["id"]
	require.True(t, exists)
	assert.Equal(t, "integer", idField.Type)
	assert.True(t, idField.Filterable)
	assert.True(t, idField.Sortable)
	assert.Contains(t, idField.Operators, "eq")
	assert.Contains(t, idField.Operators, "gt")
	assert.Contains(t, idField.Operators, "in")
	assert.NotContains(t, idField.Operators, "like") // Integer shouldn't have string operators
	assert.Equal(t, "Unique identifier", idField.Description)
	assert.Equal(t, 123, idField.Example)

	// Test name field (string type with text operators)
	nameField, exists := fieldMap["name"]
	require.True(t, exists)
	assert.Equal(t, "string", nameField.Type)
	assert.True(t, nameField.Filterable)
	assert.True(t, nameField.Sortable)
	assert.Contains(t, nameField.Operators, "eq")
	assert.Contains(t, nameField.Operators, "contains")
	assert.Contains(t, nameField.Operators, "like")
	assert.Equal(t, "Name field", nameField.Description)
	assert.Equal(t, "John Doe", nameField.Example)

	// Test email field
	emailField, exists := fieldMap["email"]
	require.True(t, exists)
	assert.Equal(t, "string", emailField.Type)
	assert.Equal(t, "Email address", emailField.Description)
	assert.Equal(t, "user@example.com", emailField.Example)

	// Test age field (number type)
	ageField, exists := fieldMap["age"]
	require.True(t, exists)
	assert.Equal(t, "number", ageField.Type)
	assert.Contains(t, ageField.Operators, "gte")
	assert.Contains(t, ageField.Operators, "between")

	// Test boolean field (is_verified should be detected as boolean)
	boolField, exists := fieldMap["is_verified"]
	require.True(t, exists)
	assert.Equal(t, "boolean", boolField.Type)
	assert.Contains(t, boolField.Operators, "eq")
	assert.Contains(t, boolField.Operators, "ne")
	assert.NotContains(t, boolField.Operators, "gt") // Boolean shouldn't have comparison operators

	// Test datetime field
	datetimeField, exists := fieldMap["created_at"]
	require.True(t, exists)
	assert.Equal(t, "datetime", datetimeField.Type)
	assert.Contains(t, datetimeField.Operators, "gt")
	assert.Contains(t, datetimeField.Operators, "between")
	assert.Equal(t, "Creation timestamp", datetimeField.Description)
	assert.Equal(t, "2024-01-01T00:00:00Z", datetimeField.Example)
}

func TestSchemaMiddleware(t *testing.T) {
	config := DefaultConfig().
		WithAllowedFields(map[string]bool{
			"name":   true,
			"status": true,
		}).
		WithMaxFilters(5)

	middleware := SchemaMiddleware(config)

	// Create a dummy handler that should NOT be called when schema is requested
	handlerCalled := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("normal response"))
	})

	wrappedHandler := middleware(dummyHandler)

	t.Run("returns schema when Accept header contains schema content type", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/users", nil)
		req.Header.Set("Accept", "application/vnd.surf+schema")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		// Should not call the original handler
		assert.False(t, handlerCalled)

		// Should return schema response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, SchemaContentType+"+json", w.Header().Get("Content-Type"))
		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))

		// Parse and validate the schema response
		var schema QuerySchema
		err := json.NewDecoder(w.Body).Decode(&schema)
		require.NoError(t, err)
		assert.Equal(t, 5, schema.MaxFilters)
		assert.Len(t, schema.Fields, 2) // name and status
	})

	t.Run("returns schema when Accept header contains schema content type with json suffix", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/users", nil)
		req.Header.Set("Accept", "application/vnd.surf+schema+json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)

		var schema QuerySchema
		err := json.NewDecoder(w.Body).Decode(&schema)
		require.NoError(t, err)
		assert.Equal(t, 5, schema.MaxFilters)
	})

	t.Run("calls normal handler when Accept header does not contain schema content type", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/users", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		// Should call the original handler
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
		body, _ := io.ReadAll(w.Body)
		assert.Equal(t, "normal response", string(body))
	})

	t.Run("calls normal handler when no Accept header is present", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		// Should call the original handler
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSchemaHandler(t *testing.T) {
	config := DefaultConfig().
		WithAllowedFields(map[string]bool{
			"id":   true,
			"name": true,
		}).
		WithMaxFilters(3).
		WithMaxSortFields(2)

	handler := SchemaHandler(config)

	req := httptest.NewRequest("GET", "/schema", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))

	var schema QuerySchema
	err := json.NewDecoder(w.Body).Decode(&schema)
	require.NoError(t, err)
	assert.Equal(t, 3, schema.MaxFilters)
	assert.Equal(t, 2, schema.MaxSortFields)
	assert.Len(t, schema.Fields, 2)
}

func TestWithSchema(t *testing.T) {
	config := DefaultConfig().
		WithAllowedFields(map[string]bool{
			"name": true,
		})

	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("original"))
	})

	wrappedHandler := WithSchema(config, originalHandler)

	t.Run("returns schema when schema content type requested", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept", SchemaContentType)
		w := httptest.NewRecorder()

		wrappedHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var schema QuerySchema
		err := json.NewDecoder(w.Body).Decode(&schema)
		require.NoError(t, err)
		assert.Len(t, schema.Fields, 1)
	})

	t.Run("calls original handler when normal content type requested", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body, _ := io.ReadAll(w.Body)
		assert.Equal(t, "original", string(body))
	})
}

func TestFieldTypeDetection(t *testing.T) {
	tests := []struct {
		fieldName    string
		expectedType string
		description  string
	}{
		{"id", "integer", "ID field should be integer"},
		{"user_id", "integer", "Fields ending with _id should be integer"},
		{"created_at", "datetime", "Fields ending with _at should be datetime"},
		{"updated_at", "datetime", "Fields ending with _at should be datetime"},
		{"date_of_birth", "datetime", "Fields containing 'date' should be datetime"},
		{"timestamp", "datetime", "Fields containing 'time' should be datetime"},
		{"is_active", "boolean", "Fields starting with 'is_' should be boolean"},
		{"has_permission", "boolean", "Fields starting with 'has_' should be boolean"},
		{"verified", "boolean", "Common boolean fields should be detected"},
		{"active", "boolean", "Common boolean fields should be detected"},
		{"age", "number", "Fields containing 'age' should be number"},
		{"count", "number", "Fields containing 'count' should be number"},
		{"amount", "number", "Fields containing 'amount' should be number"},
		{"price", "number", "Fields containing 'price' should be number"},
		{"name", "string", "Default should be string"},
		{"description", "string", "Default should be string"},
		{"random_field", "string", "Unknown fields should default to string"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			config := DefaultConfig().WithAllowedFields(map[string]bool{tt.fieldName: true})
			schema := GenerateSchema(config)

			require.Len(t, schema.Fields, 1)
			assert.Equal(t, tt.expectedType, schema.Fields[0].Type, "Field %s should be type %s", tt.fieldName, tt.expectedType)
		})
	}
}

func TestSchemaContentTypeConstant(t *testing.T) {
	assert.Equal(t, "application/vnd.surf+schema", SchemaContentType)
}

func TestSchemaExamples(t *testing.T) {
	t.Run("with name and age fields only", func(t *testing.T) {
		config := DefaultConfig().WithAllowedFields(map[string]bool{"name": true, "age": true})
		schema := GenerateSchema(config)

		// Should only generate age example since name+status example requires status field
		require.Len(t, schema.Examples, 1)

		// Should be the age range example
		example := schema.Examples[0]
		assert.Contains(t, example.Query, "age[gte]=18")
		assert.Contains(t, example.Query, "age[lt]=65")
		assert.Contains(t, example.Description, "aged")
	})

	t.Run("with all common fields", func(t *testing.T) {
		config := DefaultConfig().WithAllowedFields(map[string]bool{
			"name":       true,
			"age":        true,
			"status":     true,
			"created_at": true,
		})
		schema := GenerateSchema(config)

		// Should generate all 3 examples when all required fields are present
		require.Len(t, schema.Examples, 3)

		examples := schema.Examples

		// First example should be about name contains + status
		assert.Contains(t, examples[0].Query, "name[contains]=john")
		assert.Contains(t, examples[0].Query, "status=active")
		assert.Contains(t, examples[0].Description, "john")

		// Second example should be about age range + sorting
		assert.Contains(t, examples[1].Query, "age[gte]=18")
		assert.Contains(t, examples[1].Query, "age[lt]=65")
		assert.Contains(t, examples[1].Query, "sort=-created_at")
		assert.Contains(t, examples[1].Description, "aged")

		// Third example should be about status and sorting
		assert.Contains(t, examples[2].Query, "status[in]=active,verified")
		assert.Contains(t, examples[2].Query, "sort=name:asc,created_at:desc")
		assert.Contains(t, examples[2].Description, "sort")
	})

	t.Run("with no common fields generates fallback", func(t *testing.T) {
		config := DefaultConfig().WithAllowedFields(map[string]bool{"custom_field": true})
		schema := GenerateSchema(config)

		// Should generate fallback example
		require.Len(t, schema.Examples, 1)

		example := schema.Examples[0]
		assert.Contains(t, example.Query, "custom_field[eq]=value")
		assert.Contains(t, example.Description, "Filter by custom_field")
	})
}

func TestMiddlewareErrorHandling(t *testing.T) {
	// Test middleware behavior when schema generation might fail
	// (though GenerateSchema is pretty robust)
	config := DefaultConfig() // Empty config should still work
	middleware := SchemaMiddleware(config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", SchemaContentType)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var schema QuerySchema
	err := json.NewDecoder(w.Body).Decode(&schema)
	require.NoError(t, err)
	assert.NotNil(t, schema.Fields) // Should at least have an empty slice
}
