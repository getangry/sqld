package sqld

import (
	"errors"
	"fmt"
)

// Error types for structured error handling
var (
	// ErrNoConnection indicates a database connection error
	ErrNoConnection = errors.New("database connection not available")

	// ErrInvalidQuery indicates a malformed query
	ErrInvalidQuery = errors.New("invalid query")

	// ErrInvalidParameter indicates invalid parameters were provided
	ErrInvalidParameter = errors.New("invalid parameter")

	// ErrTransactionFailed indicates a transaction operation failed
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrSQLInjection indicates potential SQL injection detected
	ErrSQLInjection = errors.New("potential SQL injection detected")

	// ErrNoRows indicates no rows were returned
	ErrNoRows = errors.New("no rows in result set")

	// ErrTooManyRows indicates more rows than expected were returned
	ErrTooManyRows = errors.New("too many rows in result set")

	// ErrUnsupportedDialect indicates an unsupported database dialect
	ErrUnsupportedDialect = errors.New("unsupported database dialect")
)

// QueryError represents an error that occurred during query execution
type QueryError struct {
	Query   string
	Params  []interface{}
	Err     error
	Context string
}

// Error implements the error interface
func (e *QueryError) Error() string {
	return fmt.Sprintf("query error in %s: %v (query: %s)", e.Context, e.Err, e.Query)
}

// Unwrap returns the underlying error
func (e *QueryError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target
func (e *QueryError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}

// TransactionError represents an error during transaction operations
type TransactionError struct {
	Operation string
	Err       error
}

// Error implements the error interface
func (e *TransactionError) Error() string {
	return fmt.Sprintf("transaction error during %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *TransactionError) Unwrap() error {
	return e.Err
}

// WrapQueryError wraps an error with query context
func WrapQueryError(err error, query string, params []interface{}, context string) error {
	if err == nil {
		return nil
	}
	return &QueryError{
		Query:   query,
		Params:  params,
		Err:     err,
		Context: context,
	}
}

// WrapTransactionError wraps an error with transaction context
func WrapTransactionError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return &TransactionError{
		Operation: operation,
		Err:       err,
	}
}
