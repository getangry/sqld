package sqld

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockDB struct {
	mock.Mock
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	mockArgs := append([]interface{}{ctx, query}, args...)
	ret := m.Called(mockArgs...)
	return ret.Get(0).(Rows), ret.Error(1)
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	mockArgs := append([]interface{}{ctx, query}, args...)
	ret := m.Called(mockArgs...)
	return ret.Get(0).(Row)
}

type MockRows struct {
	mock.Mock
}

func (m *MockRows) Close() error {
	ret := m.Called()
	return ret.Error(0)
}

func (m *MockRows) Next() bool {
	ret := m.Called()
	return ret.Bool(0)
}

func (m *MockRows) Scan(dest ...interface{}) error {
	ret := m.Called(dest...)
	return ret.Error(0)
}

func (m *MockRows) Err() error {
	ret := m.Called()
	return ret.Error(0)
}

type MockRow struct {
	mock.Mock
}

func (m *MockRow) Scan(dest ...interface{}) error {
	ret := m.Called(dest...)
	return ret.Error(0)
}

// ErrorRow represents a row that had an error during creation (for testing)
type ErrorRow struct {
	err error
}

// Scan returns the error
func (r *ErrorRow) Scan(dest ...interface{}) error {
	return r.err
}

// TestQueries tests the wrapper functionality
func TestQueries(t *testing.T) {
	t.Run("New creates wrapper correctly", func(t *testing.T) {
		mockDB := &MockDB{}

		q := New(mockDB, Postgres)

		assert.NotNil(t, q)
		assert.Equal(t, mockDB, q.DB())
		assert.Equal(t, Postgres, q.Dialect())
	})
}
