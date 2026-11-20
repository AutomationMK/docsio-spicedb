package dbdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB holds the database connection pool
type DB struct {
	SQL *pgxpool.Pool
}

var dbConn = &DB{}

// ConnectSQL sets the global database pool variable
func ConnectSQL(dsn string, pings int) (*DB, error) {
	d, err := NewDatabase(dsn, pings)
	if err != nil {
		return nil, err
	}

	dbConn.SQL = d

	return dbConn, nil
}

// NewDatabase creates a new database for the application
func NewDatabase(dsn string, pings int) (*pgxpool.Pool, error) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// test if database is connected and return error if not
	for i := range pings {
		if err := db.Ping(ctx); err == nil {
			return db, nil
		}
		time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for DB connection")
}

// DBRetryConfig holds database transaction retry configuration
type DBRetryConfig struct {
	BaseDelay      time.Duration // initial backoff
	MaxDelay       time.Duration // cap on backoff
	MaxElapsedTime time.Duration // max elapsed time for retry
}

// set default database transaction retry configuration
var DefaultDBRetryCongig = DBRetryConfig{
	BaseDelay:      10 * time.Millisecond,
	MaxDelay:       500 * time.Millisecond,
	MaxElapsedTime: 5 * time.Second,
}

// retryableCodes are Postgres SQLSTATEs that are safe to retry on a fresh transaction
// 40001 = serialization_failure
// 40P01 = deadlock_detected
var retryableCodes = map[string]struct{}{
	"40001": {},
	"40P01": {},
}

// isRetryable gives a true value if the pgconn.PgError is in retryableCodes map
func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	_, ok := retryableCodes[pgErr.Code]
	return ok
}

// TxWithRetry runs fn inside a transaction, retrying the whole transaction
// on retryable errors
func TxWithRetry(ctx context.Context, db *pgxpool.Pool, cfg DBRetryConfig, fn func(pgx.Tx) error) error {
	// exponential backoff with
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = cfg.BaseDelay
	b.MaxInterval = cfg.MaxDelay
	b.MaxElapsedTime = cfg.MaxElapsedTime // overall ceiling

	operation := func() error {
		err := pgx.BeginFunc(ctx, db, fn)
		if err == nil {
			return nil
		}

		// non-retryable: tell the backoff to stop and return the error
		if !isRetryable(err) {
			return backoff.Permanent(err)
		}

		return err
	}

	// Retry accepts the context via backoff.WithContext, ensuring cancelation
	// interupts the sleep between attempts
	return backoff.Retry(operation, backoff.WithContext(b, ctx))
}
