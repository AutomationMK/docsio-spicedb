package dbdriver

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestNewDatabase tests the NewDatabase function
func TestNewDatabase(t *testing.T) {
	wrongDsn := dsnConf{
		hostName:      "wronghost",
		containerName: "test-postgres",
		user:          "testUser",
		password:      "testPassword",
		dbName:        "testdb",
		port:          "5432",
	}
	wrongDsnString := createDSNString(wrongDsn)

	// test with empty dsn string and see if an error occurs
	if _, err := NewDatabase("", 1); err == nil {
		t.Errorf("did not get error when using empty dsn string")
	}

	// test without a database connection available first and see if an error occurs
	if _, err := NewDatabase(wrongDsnString, 1); err == nil {
		t.Errorf("did not receive error when missing database connection")
	}

	db, err := NewDatabase(dsnString, 7)
	defer db.Close()
	if err != nil {
		t.Errorf("got an error when using a valid database connection: %v", err)
	}
	if db == nil {
		t.Errorf("empty database pool when using a valid database connection")
	}

	// test if db.config matches dsn config
	dbConfig := db.Config()
	if dbConfig.ConnConfig.Host != dsn.hostName {
		t.Errorf("expected %s ConnConfig.Host but return has %s", dsn.hostName, dbConfig.ConnConfig.Host)
	}
	if dbConfig.ConnConfig.User != dsn.user {
		t.Errorf("expected %s ConnConfig.User but return has %s", dsn.user, dbConfig.ConnConfig.User)
	}
	if dbConfig.ConnConfig.Password != dsn.password {
		t.Errorf("expected %s ConnConfig.Password but return has %s", dsn.password, dbConfig.ConnConfig.Password)
	}
	if dbConfig.ConnConfig.Database != dsn.dbName {
		t.Errorf("expected %s ConnConfig.Database but return has %s", dsn.dbName, dbConfig.ConnConfig.Database)
	}
	dsnPortInt, err := strconv.Atoi(dsn.port)
	if err != nil {
		t.Fatalf("dsn port not converting to int: %v", err)
	}
	if dbConfig.ConnConfig.Port != uint16(dsnPortInt) {
		t.Errorf("expected %d ConnConfig.Port but return has %d", dsnPortInt, dbConfig.ConnConfig.Port)
	}
}

// TestConnectSQL tests the ConnectSQL function
func TestConnectSQL(t *testing.T) {
	wrongDsn := dsnConf{
		hostName:      "wronghost",
		containerName: "test-postgres",
		user:          "testUser",
		password:      "testPassword",
		dbName:        "testdb",
		port:          "5432",
	}
	wrongDsnString := createDSNString(wrongDsn)

	// test with empty dsn string and see if an error occurs
	if _, err := ConnectSQL("", 1); err == nil {
		t.Errorf("did not get error when using empty dsn string")
	}

	// test without a database connection available first and see if an error occurs
	if _, err := ConnectSQL(wrongDsnString, 1); err == nil {
		t.Errorf("did not receive error when missing database connection")
	}

	db, err := ConnectSQL(dsnString, 7)
	defer db.SQL.Close()
	if err != nil {
		t.Errorf("got an error when using a valid database connection: %v", err)
	}
	if db == nil {
		t.Errorf("empty database pool when using a valid database connection")
	}
	if dbConn == nil {
		t.Errorf("empty global database pool variable when using a valid database connection")
	}

	// test if db.config matches dsn config
	configReturn := db.SQL.Config()
	if configReturn.ConnConfig.Host != dsn.hostName {
		t.Errorf("expected %s ConnConfig.Host but return has %s", dsn.hostName, configReturn.ConnConfig.Host)
	}
	if configReturn.ConnConfig.User != dsn.user {
		t.Errorf("expected %s ConnConfig.User but return has %s", dsn.user, configReturn.ConnConfig.User)
	}
	if configReturn.ConnConfig.Password != dsn.password {
		t.Errorf("expected %s ConnConfig.Password but return has %s", dsn.password, configReturn.ConnConfig.Password)
	}
	if configReturn.ConnConfig.Database != dsn.dbName {
		t.Errorf("expected %s ConnConfig.Database but return has %s", dsn.dbName, configReturn.ConnConfig.Database)
	}
	dsnPortInt, err := strconv.Atoi(dsn.port)
	if err != nil {
		t.Fatalf("dsn port not converting to int: %v", err)
	}
	if configReturn.ConnConfig.Port != uint16(dsnPortInt) {
		t.Errorf("expected %d ConnConfig.Port but return has %d", dsnPortInt, configReturn.ConnConfig.Port)
	}

	// compare database pool configurations to see if they match
	configGlobal := db.SQL.Config()
	if configReturn.ConnConfig.Host != configGlobal.ConnConfig.Host {
		t.Errorf("expected %s ConnConfig.Host but got return of %s", configGlobal.ConnConfig.Host, configReturn.ConnConfig.Host)
	}
	if configReturn.ConnConfig.Port != configGlobal.ConnConfig.Port {
		t.Errorf("expected %d ConnConfig.Port but got return of %d", configGlobal.ConnConfig.Port, configReturn.ConnConfig.Port)
	}
	if configReturn.ConnConfig.User != configGlobal.ConnConfig.User {
		t.Errorf("expected %s ConnConfig.User but got return of %s", configGlobal.ConnConfig.User, configReturn.ConnConfig.User)
	}
	if configReturn.ConnConfig.Password != configGlobal.ConnConfig.Password {
		t.Errorf("expected %s ConnConfig.Password but got return of %s", configGlobal.ConnConfig.Password, configReturn.ConnConfig.Password)
	}
	if configReturn.ConnConfig.Database != configGlobal.ConnConfig.Database {
		t.Errorf("expected %s ConnConfig.Database but got return of %s", configGlobal.ConnConfig.Database, configReturn.ConnConfig.Database)
	}
	if int64(configReturn.ConnConfig.ConnectTimeout) != int64(configGlobal.ConnConfig.ConnectTimeout) {
		t.Errorf("expected %d ConnConfig.ConnectTimeout but got return of %d", int64(configGlobal.ConnConfig.ConnectTimeout), int64(configReturn.ConnConfig.ConnectTimeout))
	}
	if configReturn.MaxConns != configGlobal.MaxConns {
		t.Errorf("expected %d MaxConns but got return of %d", configGlobal.MaxConns, configReturn.MaxConns)
	}
	if configReturn.MinConns != configGlobal.MinConns {
		t.Errorf("expected %d MinConns but got return of %d", configGlobal.MinConns, configReturn.MinConns)
	}
	if configReturn.MinIdleConns != configGlobal.MinIdleConns {
		t.Errorf("expected %d MinIdleConns but got return of %d", configGlobal.MinIdleConns, configReturn.MinIdleConns)
	}
	if int64(configReturn.MaxConnLifetime) != int64(configGlobal.MaxConnLifetime) {
		t.Errorf("expected %d MaxConnLifetime but got return of %d", int64(configGlobal.MaxConnLifetime), int64(configReturn.MaxConnLifetime))
	}
	if int64(configReturn.MaxConnLifetimeJitter) != int64(configGlobal.MaxConnLifetimeJitter) {
		t.Errorf("expected %d MaxConnLifetimeJitter but got return of %d", int64(configGlobal.MaxConnLifetimeJitter), int64(configReturn.MaxConnLifetimeJitter))
	}
	if int64(configReturn.MaxConnIdleTime) != int64(configGlobal.MaxConnIdleTime) {
		t.Errorf("expected %d MaxConnIdleTime but got return of %d", int64(configGlobal.MaxConnIdleTime), int64(configReturn.MaxConnIdleTime))
	}
	if int64(configReturn.PingTimeout) != int64(configGlobal.PingTimeout) {
		t.Errorf("expected %d PingTimeout but got return of %d", int64(configGlobal.PingTimeout), int64(configReturn.PingTimeout))
	}
	if int64(configReturn.HealthCheckPeriod) != int64(configGlobal.HealthCheckPeriod) {
		t.Errorf("expected %d HealthCheckPeriod but got return of %d", int64(configGlobal.HealthCheckPeriod), int64(configReturn.HealthCheckPeriod))
	}
}

// TestisRetryable tests the isRetryable function
func TestIsRetryable(t *testing.T) {
	// test every possible errors and confirm the correct response
	for _, test := range mockTests {
		t.Run(test.name, func(t *testing.T) {
			if got := isRetryable(test.err); got != test.want {
				t.Errorf("isRetryable() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestTxWithRetry tests the TxWithRetry function
func TestTxWithRetry(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsnString)
	if err != nil {
		t.Fatalf("unable to connect to docker database")
	}
	defer db.Close()

	fastCfg := DBRetryConfig{
		BaseDelay:      5 * time.Millisecond,
		MaxDelay:       20 * time.Millisecond,
		MaxElapsedTime: 2 * time.Second,
	}

	t.Run("commits on success", func(t *testing.T) {
		err := TxWithRetry(ctx, db, fastCfg, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "CREATE TEMP TABLE t1(x int)")
			return err
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	for _, test := range mockTests {
		t.Run(fmt.Sprintf("checks for correct retries on test %s", test.name), func(t *testing.T) {
			calls := 0
			err := TxWithRetry(ctx, db, fastCfg, func(tx pgx.Tx) error {
				calls++
				if calls == 1 {
					// force a test failure
					return test.err
				}
				return nil
			})
			// test for retryable errors
			if err != nil && test.want {
				t.Fatalf("unexpected err: %v for test %s", err, test.name)
			}
			if calls != 2 && test.want {
				t.Fatalf("expected 2 calls, got %d", calls)
			}
			// test for non retryable errors
			if err == nil && !test.want {
				t.Fatalf("no error but expected err: %v for test %s", test.err, test.name)
			}
			if err != test.err && !test.want {
				t.Fatalf("expected err: %v but got %v for test %s", test.err, err, test.name)
			}
		})
	}
}
