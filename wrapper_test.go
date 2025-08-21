package sqld

import (
	"context"
	"errors"
	"strings"
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

func TestEnhancedQueries_PaginationQuery(t *testing.T) {
	t.Run("successful pagination", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)
		
		ctx := context.Background()
		baseQuery := "SELECT * FROM users"
		orderBy := "name ASC"
		limit := 10
		offset := 20
		
		query, params, err := eq.PaginationQuery(ctx, baseQuery, nil, limit, offset, orderBy)
		
		assert.NoError(t, err)
		assert.Contains(t, query, "ORDER BY name ASC")
		assert.Contains(t, query, "LIMIT $1")
		assert.Contains(t, query, "OFFSET $2")
		assert.Equal(t, []interface{}{10, 20}, params)
	})
	
	t.Run("query validation error", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)
		
		ctx := context.Background()
		baseQuery := "" // Invalid empty query
		
		_, _, err := eq.PaginationQuery(ctx, baseQuery, nil, 10, 0, "")
		assert.Error(t, err)
		
		var qErr *QueryError
		assert.True(t, errors.As(err, &qErr))
		assert.Contains(t, qErr.Context, "validation")
	})
	
	t.Run("invalid order by", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)
		
		ctx := context.Background()
		baseQuery := "SELECT * FROM users"
		orderBy := "name INVALID_DIRECTION"
		
		_, _, err := eq.PaginationQuery(ctx, baseQuery, nil, 10, 0, orderBy)
		assert.Error(t, err)
		
		var qErr *QueryError
		assert.True(t, errors.As(err, &qErr))
	})
	
	t.Run("negative limit", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)
		
		ctx := context.Background()
		baseQuery := "SELECT * FROM users"
		
		_, _, err := eq.PaginationQuery(ctx, baseQuery, nil, -1, 0, "")
		assert.Error(t, err)
		
		var vErr *ValidationError
		assert.True(t, errors.As(err, &vErr))
		assert.Equal(t, "limit", vErr.Field)
	})
	
	t.Run("negative offset", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)
		
		ctx := context.Background()
		baseQuery := "SELECT * FROM users"
		
		_, _, err := eq.PaginationQuery(ctx, baseQuery, nil, 10, -1, "")
		assert.Error(t, err)
		
		var vErr *ValidationError
		assert.True(t, errors.As(err, &vErr))
		assert.Equal(t, "offset", vErr.Field)
	})
	
	t.Run("MySQL dialect", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, MySQL)
		
		ctx := context.Background()
		baseQuery := "SELECT * FROM users"
		
		query, params, err := eq.PaginationQuery(ctx, baseQuery, nil, 10, 20, "")
		
		assert.NoError(t, err)
		assert.Contains(t, query, "LIMIT ?")
		assert.Contains(t, query, "OFFSET ?")
		assert.Equal(t, []interface{}{10, 20}, params)
	})
	
	t.Run("with where conditions", func(t *testing.T) {
		mockDB := &MockDB{}
		queries := struct{}{}
		eq := NewEnhanced(queries, mockDB, Postgres)
		
		ctx := context.Background()
		baseQuery := "SELECT * FROM users"
		
		where := NewWhereBuilder(Postgres)
		where.Equal("status", "active")
		
		query, params, err := eq.PaginationQuery(ctx, baseQuery, where, 10, 0, "name")
		
		assert.NoError(t, err)
		assert.Contains(t, query, "WHERE status = $1")
		assert.Contains(t, query, "ORDER BY name")
		assert.Contains(t, query, "LIMIT $2")
		assert.Equal(t, []interface{}{"active", 10}, params)
	})
}

func TestEnhancedQueries_SearchQuery(t *testing.T) {
	mockDB := &MockDB{}
	queries := struct{}{}
	eq := NewEnhanced(queries, mockDB, Postgres)
	
	t.Run("search with text and filters", func(t *testing.T) {
		searchColumns := []string{"name", "email"}
		searchText := "john"
		
		filters := NewWhereBuilder(Postgres)
		filters.Equal("status", "active")
		
		where := eq.SearchQuery("SELECT * FROM users", searchColumns, searchText, filters)
		
		sql, params := where.Build()
		
		assert.Contains(t, sql, "ILIKE")
		assert.Contains(t, sql, "status = ")
		assert.Len(t, params, 3) // 2 for search + 1 for filter
		
		// Check that the search pattern is in parameters
		found := false
		for _, p := range params {
			if str, ok := p.(string); ok && strings.Contains(str, "john") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find 'john' in parameters")
	})
	
	t.Run("search without text", func(t *testing.T) {
		searchColumns := []string{"name", "email"}
		searchText := ""
		
		filters := NewWhereBuilder(Postgres)
		filters.Equal("status", "active")
		
		where := eq.SearchQuery("SELECT * FROM users", searchColumns, searchText, filters)
		
		sql, params := where.Build()
		
		assert.NotContains(t, sql, "ILIKE")
		assert.Contains(t, sql, "status = ")
		assert.Len(t, params, 1) // Only filter parameter
	})
	
	t.Run("search without filters", func(t *testing.T) {
		searchColumns := []string{"name", "email"}
		searchText := "john"
		
		where := eq.SearchQuery("SELECT * FROM users", searchColumns, searchText, nil)
		
		sql, params := where.Build()
		
		assert.Contains(t, sql, "ILIKE")
		assert.Len(t, params, 2) // 2 search parameters
		
		// Check that the search pattern is in parameters
		found := false
		for _, p := range params {
			if str, ok := p.(string); ok && strings.Contains(str, "john") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find 'john' in parameters")
	})
}

func TestScanToSlice(t *testing.T) {
	t.Run("successful scan", func(t *testing.T) {
		mockRows := &MockRows{}
		
		// Setup mock to return 2 rows
		mockRows.On("Next").Return(true).Once()
		mockRows.On("Next").Return(true).Once()
		mockRows.On("Next").Return(false).Once()
		mockRows.On("Err").Return(nil)
		
		scanCount := 0
		scanFn := func(rows Rows) (string, error) {
			scanCount++
			return "item", nil
		}
		
		results, err := ScanToSlice(mockRows, scanFn)
		
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, []string{"item", "item"}, results)
		assert.Equal(t, 2, scanCount)
		
		mockRows.AssertExpectations(t)
	})
	
	t.Run("scan function error", func(t *testing.T) {
		mockRows := &MockRows{}
		
		mockRows.On("Next").Return(true).Once()
		
		scanError := errors.New("scan error")
		scanFn := func(rows Rows) (string, error) {
			return "", scanError
		}
		
		results, err := ScanToSlice(mockRows, scanFn)
		
		assert.Error(t, err)
		assert.Equal(t, scanError, err)
		assert.Nil(t, results)
		
		mockRows.AssertExpectations(t)
	})
	
	t.Run("rows error", func(t *testing.T) {
		mockRows := &MockRows{}
		
		mockRows.On("Next").Return(false).Once()
		rowsError := errors.New("rows error")
		mockRows.On("Err").Return(rowsError)
		
		scanFn := func(rows Rows) (string, error) {
			return "item", nil
		}
		
		results, err := ScanToSlice(mockRows, scanFn)
		
		assert.Error(t, err)
		assert.Equal(t, rowsError, err)
		assert.Nil(t, results)
		
		mockRows.AssertExpectations(t)
	})
}

func TestScanToMap(t *testing.T) {
	t.Run("successful scan", func(t *testing.T) {
		mockRows := &MockRows{}
		
		// Setup mock to return 2 rows
		mockRows.On("Next").Return(true).Once()
		mockRows.On("Next").Return(true).Once()
		mockRows.On("Next").Return(false).Once()
		mockRows.On("Err").Return(nil)
		
		keyCount := 0
		valueCount := 0
		
		keyFn := func(rows Rows) (int, error) {
			keyCount++
			return keyCount, nil
		}
		
		valueFn := func(rows Rows) (string, error) {
			valueCount++
			return "value", nil
		}
		
		results, err := ScanToMap(mockRows, keyFn, valueFn)
		
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, map[int]string{1: "value", 2: "value"}, results)
		
		mockRows.AssertExpectations(t)
	})
	
	t.Run("key function error", func(t *testing.T) {
		mockRows := &MockRows{}
		
		mockRows.On("Next").Return(true).Once()
		
		keyError := errors.New("key error")
		keyFn := func(rows Rows) (int, error) {
			return 0, keyError
		}
		
		valueFn := func(rows Rows) (string, error) {
			return "value", nil
		}
		
		results, err := ScanToMap(mockRows, keyFn, valueFn)
		
		assert.Error(t, err)
		assert.Equal(t, keyError, err)
		assert.Nil(t, results)
		
		mockRows.AssertExpectations(t)
	})
}

