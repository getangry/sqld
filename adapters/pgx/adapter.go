package pgx

import (
	"context"

	"github.com/getangry/sqld"
	"github.com/jackc/pgx/v5"
)

// PgxAdapter wraps pgx.Conn to implement the sqld DBTX interface
type PgxAdapter struct {
	conn *pgx.Conn
}

// NewPgxAdapter creates a new adapter for pgx.Conn
func NewPgxAdapter(conn *pgx.Conn) *PgxAdapter {
	return &PgxAdapter{conn: conn}
}

// Query implements the DBTX interface
func (p *PgxAdapter) Query(ctx context.Context, sql string, args ...interface{}) (sqld.Rows, error) {
	rows, err := p.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRowsAdapter{rows: rows}, nil
}

// QueryRow implements the DBTX interface
func (p *PgxAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) sqld.Row {
	row := p.conn.QueryRow(ctx, sql, args...)
	return &PgxRowAdapter{row: row}
}

// PgxRowsAdapter wraps pgx.Rows to implement the sqld Rows interface
type PgxRowsAdapter struct {
	rows pgx.Rows
}

// Close implements the Rows interface
func (p *PgxRowsAdapter) Close() error {
	p.rows.Close()
	return nil
}

// Next implements the Rows interface
func (p *PgxRowsAdapter) Next() bool {
	return p.rows.Next()
}

// Scan implements the Rows interface
func (p *PgxRowsAdapter) Scan(dest ...interface{}) error {
	return p.rows.Scan(dest...)
}

// Err implements the Rows interface
func (p *PgxRowsAdapter) Err() error {
	return p.rows.Err()
}

// PgxRowAdapter wraps pgx.Row to implement the sqld Row interface
type PgxRowAdapter struct {
	row pgx.Row
}

// Scan implements the Row interface
func (p *PgxRowAdapter) Scan(dest ...interface{}) error {
	return p.row.Scan(dest...)
}
