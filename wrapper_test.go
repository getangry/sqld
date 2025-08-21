package sqld

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEnhancedQueries_DynamicQuery(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		mockDB := &MockDB{}
		mockRows := &MockRows{}

		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "SELECT * FROM users"

		// Setup mocks
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(mockRows, nil)
		mockRows.On("Close").Return(nil)

		// Mock scan function
		scanFn := func(rows Rows) error {
			return nil
		}

		err := eq.DynamicQuery(ctx, baseQuery, nil, scanFn)
		assert.NoError(t, err)

		mockDB.AssertExpectations(t)
		mockRows.AssertExpectations(t)
	})

	t.Run("query validation error", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "" // Invalid empty query

		scanFn := func(rows Rows) error {
			return nil
		}

		err := eq.DynamicQuery(ctx, baseQuery, nil, scanFn)
		assert.Error(t, err)

		var qErr *QueryError
		assert.True(t, errors.As(err, &qErr))
		assert.Contains(t, qErr.Context, "validation")
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "SELECT * FROM users"

		dbError := errors.New("database connection lost")
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return((*MockRows)(nil), dbError)

		scanFn := func(rows Rows) error {
			return nil
		}

		err := eq.DynamicQuery(ctx, baseQuery, nil, scanFn)
		assert.Error(t, err)

		var qErr *QueryError
		assert.True(t, errors.As(err, &qErr))
		assert.Contains(t, qErr.Context, "execution")
		assert.Equal(t, dbError, qErr.Unwrap())

		mockDB.AssertExpectations(t)
	})

	t.Run("scan error", func(t *testing.T) {
		mockDB := &MockDB{}
		mockRows := &MockRows{}

		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "SELECT * FROM users"

		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(mockRows, nil)
		mockRows.On("Close").Return(nil)

		scanError := errors.New("scan error")
		scanFn := func(rows Rows) error {
			return scanError
		}

		err := eq.DynamicQuery(ctx, baseQuery, nil, scanFn)
		assert.Error(t, err)

		var qErr *QueryError
		assert.True(t, errors.As(err, &qErr))
		assert.Contains(t, qErr.Context, "scanning")
		assert.Equal(t, scanError, qErr.Unwrap())

		mockDB.AssertExpectations(t)
		mockRows.AssertExpectations(t)
	})

	t.Run("with where conditions", func(t *testing.T) {
		mockDB := &MockDB{}
		mockRows := &MockRows{}

		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "SELECT * FROM users"

		where := NewWhereBuilder(Postgres)
		where.Equal("status", "active")

		expectedQuery := "SELECT * FROM users WHERE status = $1"

		// Use mock.Anything for the params since they're passed as variadic args
		mockDB.On("Query", ctx, expectedQuery, "active").Return(mockRows, nil)
		mockRows.On("Close").Return(nil)

		scanFn := func(rows Rows) error {
			return nil
		}

		err := eq.DynamicQuery(ctx, baseQuery, where, scanFn)
		assert.NoError(t, err)

		mockDB.AssertExpectations(t)
		mockRows.AssertExpectations(t)
	})
}

func TestEnhancedQueries_DynamicQueryRow(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		mockDB := &MockDB{}
		mockRow := &MockRow{}

		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "SELECT * FROM users WHERE id = $1"

		mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

		row := eq.DynamicQueryRow(ctx, baseQuery, nil)
		assert.NotNil(t, row)
		assert.Equal(t, mockRow, row)

		mockDB.AssertExpectations(t)
	})

	t.Run("query validation error", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)

		ctx := context.Background()
		baseQuery := "" // Invalid empty query

		row := eq.DynamicQueryRow(ctx, baseQuery, nil)

		errorRow, ok := row.(*ErrorRow)
		assert.True(t, ok)
		assert.Error(t, errorRow.err)

		var qErr *QueryError
		assert.True(t, errors.As(errorRow.err, &qErr))
		assert.Contains(t, qErr.Context, "validation")
	})
}


