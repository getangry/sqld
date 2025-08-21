package sqld

import (
	"context"
	"database/sql"
	"fmt"
)

// TxOptions represents transaction options
type TxOptions struct {
	IsolationLevel sql.IsolationLevel
	ReadOnly       bool
}

// Tx represents a database transaction
type Tx interface {
	DBTX
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TxManager manages database transactions
type TxManager interface {
	BeginTx(ctx context.Context, opts *TxOptions) (Tx, error)
	WithTransaction(ctx context.Context, opts *TxOptions, fn func(ctx context.Context, tx Tx) error) error
}

// StandardTx wraps a standard database/sql transaction
type StandardTx struct {
	tx      *sql.Tx
	dialect Dialect
}

// NewStandardTx creates a new standard transaction wrapper
func NewStandardTx(tx *sql.Tx, dialect Dialect) *StandardTx {
	return &StandardTx{
		tx:      tx,
		dialect: dialect,
	}
}

// Query executes a query within the transaction
func (t *StandardTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	// Validate query for SQL injection
	if err := ValidateQuery(query, t.dialect); err != nil {
		return nil, WrapQueryError(err, query, args, "transaction query")
	}

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, WrapQueryError(err, query, args, "transaction query")
	}
	return &StandardRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row within the transaction
func (t *StandardTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	// Validate query for SQL injection
	if err := ValidateQuery(query, t.dialect); err != nil {
		return &ErrorRow{err: WrapQueryError(err, query, args, "transaction query row")}
	}

	row := t.tx.QueryRowContext(ctx, query, args...)
	return &StandardRow{row: row}
}

// Exec executes a query that doesn't return rows within the transaction
func (t *StandardTx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Validate query for SQL injection
	if err := ValidateQuery(query, t.dialect); err != nil {
		return nil, WrapQueryError(err, query, args, "transaction exec")
	}

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, WrapQueryError(err, query, args, "transaction exec")
	}
	return result, nil
}

// Commit commits the transaction
func (t *StandardTx) Commit(ctx context.Context) error {
	if err := t.tx.Commit(); err != nil {
		return WrapTransactionError(err, "commit")
	}
	return nil
}

// Rollback rolls back the transaction
func (t *StandardTx) Rollback(ctx context.Context) error {
	if err := t.tx.Rollback(); err != nil {
		return WrapTransactionError(err, "rollback")
	}
	return nil
}

// StandardDB wraps a standard database/sql DB to provide transaction support
type StandardDB struct {
	db      *sql.DB
	dialect Dialect
}

// NewStandardDB creates a new standard database wrapper
func NewStandardDB(db *sql.DB, dialect Dialect) *StandardDB {
	return &StandardDB{
		db:      db,
		dialect: dialect,
	}
}

// Query executes a query
func (d *StandardDB) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	// Validate query for SQL injection
	if err := ValidateQuery(query, d.dialect); err != nil {
		return nil, WrapQueryError(err, query, args, "query")
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, WrapQueryError(err, query, args, "query")
	}
	return &StandardRows{rows: rows}, nil
}

// QueryRow executes a query that returns a single row
func (d *StandardDB) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	// Validate query for SQL injection
	if err := ValidateQuery(query, d.dialect); err != nil {
		return &ErrorRow{err: WrapQueryError(err, query, args, "query row")}
	}

	row := d.db.QueryRowContext(ctx, query, args...)
	return &StandardRow{row: row}
}

// BeginTx starts a new transaction
func (d *StandardDB) BeginTx(ctx context.Context, opts *TxOptions) (Tx, error) {
	var txOpts *sql.TxOptions
	if opts != nil {
		txOpts = &sql.TxOptions{
			Isolation: opts.IsolationLevel,
			ReadOnly:  opts.ReadOnly,
		}
	}

	tx, err := d.db.BeginTx(ctx, txOpts)
	if err != nil {
		return nil, WrapTransactionError(err, "begin")
	}

	return NewStandardTx(tx, d.dialect), nil
}

// WithTransaction executes a function within a transaction
func (d *StandardDB) WithTransaction(ctx context.Context, opts *TxOptions, fn func(ctx context.Context, tx Tx) error) error {
	tx, err := d.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	// Ensure transaction is handled properly
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r) // Re-panic after rollback
		}
	}()

	// Execute the function
	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %v", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return WrapTransactionError(err, "commit")
	}

	return nil
}

// StandardRows wraps database/sql Rows
type StandardRows struct {
	rows *sql.Rows
}

// Close closes the rows
func (r *StandardRows) Close() error {
	return r.rows.Close()
}

// Next advances to the next row
func (r *StandardRows) Next() bool {
	return r.rows.Next()
}

// Scan scans the current row
func (r *StandardRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

// Err returns any error that occurred during iteration
func (r *StandardRows) Err() error {
	return r.rows.Err()
}

// StandardRow wraps database/sql Row
type StandardRow struct {
	row *sql.Row
}

// Scan scans the row
func (r *StandardRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// ErrorRow represents a row that had an error during creation
type ErrorRow struct {
	err error
}

// Scan returns the error
func (r *ErrorRow) Scan(dest ...interface{}) error {
	return r.err
}

// TransactionalQueries wraps queries to support transactions
type TransactionalQueries[T any] struct {
	*EnhancedQueries[T]
	txManager TxManager
}

// NewTransactionalQueries creates a new transactional queries wrapper
func NewTransactionalQueries[T any](queries T, db DBTX, dialect Dialect, txManager TxManager) *TransactionalQueries[T] {
	return &TransactionalQueries[T]{
		EnhancedQueries: NewEnhanced(queries, db, dialect),
		txManager:       txManager,
	}
}

// WithTx executes queries within a transaction
func (tq *TransactionalQueries[T]) WithTx(ctx context.Context, opts *TxOptions, fn func(ctx context.Context, queries *EnhancedQueries[T]) error) error {
	return tq.txManager.WithTransaction(ctx, opts, func(ctx context.Context, tx Tx) error {
		// Create a new EnhancedQueries instance with the transaction
		txQueries := NewEnhanced(tq.queries, tx, tq.dialect)
		return fn(ctx, txQueries)
	})
}

// RunInTransaction is a helper to run multiple operations in a transaction
func RunInTransaction(ctx context.Context, txManager TxManager, opts *TxOptions, operations ...func(ctx context.Context, tx Tx) error) error {
	return txManager.WithTransaction(ctx, opts, func(ctx context.Context, tx Tx) error {
		for _, op := range operations {
			if err := op(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
}
