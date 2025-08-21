package sqld

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryError(t *testing.T) {
	originalErr := errors.New("connection lost")
	query := "SELECT * FROM users WHERE id = $1"
	params := []interface{}{123}
	context := "test query"

	qErr := &QueryError{
		Query:   query,
		Params:  params,
		Err:     originalErr,
		Context: context,
	}

	// Test Error method
	expectedMsg := "query error in test query: connection lost (query: SELECT * FROM users WHERE id = $1)"
	assert.Equal(t, expectedMsg, qErr.Error())

	// Test Unwrap method
	assert.Equal(t, originalErr, qErr.Unwrap())

	// Test Is method
	assert.True(t, qErr.Is(originalErr))
	assert.False(t, qErr.Is(errors.New("different error")))
}

func TestValidationError(t *testing.T) {
	vErr := &ValidationError{
		Field:   "username",
		Value:   "invalid@name",
		Message: "contains invalid characters",
	}

	expectedMsg := "validation error for field username: contains invalid characters"
	assert.Equal(t, expectedMsg, vErr.Error())
}

func TestTransactionError(t *testing.T) {
	originalErr := errors.New("deadlock detected")
	operation := "commit"

	tErr := &TransactionError{
		Operation: operation,
		Err:       originalErr,
	}

	// Test Error method
	expectedMsg := "transaction error during commit: deadlock detected"
	assert.Equal(t, expectedMsg, tErr.Error())

	// Test Unwrap method
	assert.Equal(t, originalErr, tErr.Unwrap())
}

func TestWrapQueryError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := WrapQueryError(nil, "SELECT 1", nil, "test")
		assert.Nil(t, result)
	})

	t.Run("wrap error", func(t *testing.T) {
		originalErr := errors.New("database error")
		query := "SELECT * FROM users"
		params := []interface{}{"test"}
		context := "user query"

		result := WrapQueryError(originalErr, query, params, context)

		assert.NotNil(t, result)

		var qErr *QueryError
		assert.True(t, errors.As(result, &qErr))
		assert.Equal(t, query, qErr.Query)
		assert.Equal(t, params, qErr.Params)
		assert.Equal(t, originalErr, qErr.Err)
		assert.Equal(t, context, qErr.Context)
	})
}

func TestWrapTransactionError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := WrapTransactionError(nil, "begin")
		assert.Nil(t, result)
	})

	t.Run("wrap error", func(t *testing.T) {
		originalErr := errors.New("transaction error")
		operation := "rollback"

		result := WrapTransactionError(originalErr, operation)

		assert.NotNil(t, result)

		var tErr *TransactionError
		assert.True(t, errors.As(result, &tErr))
		assert.Equal(t, operation, tErr.Operation)
		assert.Equal(t, originalErr, tErr.Err)
	})
}

func TestErrorConstants(t *testing.T) {
	// Test that all error constants are properly defined
	assert.NotNil(t, ErrNoConnection)
	assert.NotNil(t, ErrInvalidQuery)
	assert.NotNil(t, ErrInvalidParameter)
	assert.NotNil(t, ErrTransactionFailed)
	assert.NotNil(t, ErrSQLInjection)
	assert.NotNil(t, ErrNoRows)
	assert.NotNil(t, ErrTooManyRows)
	assert.NotNil(t, ErrUnsupportedDialect)

	// Test that error messages are meaningful
	assert.Contains(t, ErrNoConnection.Error(), "connection")
	assert.Contains(t, ErrInvalidQuery.Error(), "query")
	assert.Contains(t, ErrInvalidParameter.Error(), "parameter")
	assert.Contains(t, ErrTransactionFailed.Error(), "transaction")
	assert.Contains(t, ErrSQLInjection.Error(), "injection")
	assert.Contains(t, ErrNoRows.Error(), "rows")
	assert.Contains(t, ErrTooManyRows.Error(), "rows")
	assert.Contains(t, ErrUnsupportedDialect.Error(), "dialect")
}
