package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// DB drivers
	_ "github.com/lib/pq" // postgres driver
	//_ "modernc.org/sqlite" // pure-Go sqlite driver (registers the "sqlite" driver)
)

// DB defines the minimal driver-agnostic operations the app needs.
type DB interface {
	Ping(ctx context.Context) error
	Close() error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	DriverName() string
}

// SQLDB is a thin wrapper around *sql.DB implementing DB.
type SQLDB struct {
	db     *sql.DB
	driver string
}

// NewSQLDB opens a DB connection, configures pool settings and verifies connectivity.
func NewSQLDB(driver, dsn string, maxOpenConns, maxIdleConns int, connMaxLifetime time.Duration) (*SQLDB, error) {
	sqldb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	if maxOpenConns > 0 {
		sqldb.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		sqldb.SetMaxIdleConns(maxIdleConns)
	}
	if connMaxLifetime > 0 {
		sqldb.SetConnMaxLifetime(connMaxLifetime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqldb.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return &SQLDB{db: sqldb, driver: driver}, nil
}

// Implement DB interface
func (s *SQLDB) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLDB) Close() error {
	return s.db.Close()
}

func (s *SQLDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *SQLDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s *SQLDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *SQLDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

func (s *SQLDB) DriverName() string {
	return s.driver
}
