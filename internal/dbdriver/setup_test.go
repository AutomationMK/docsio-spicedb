package dbdriver

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

var dsnString string
var dsn dsnConf

func TestMain(m *testing.M) {
	dsn = dsnConf{
		hostName:      "localhost",
		containerName: "test-postgres",
		user:          "testUser",
		password:      "testPassword",
		dbName:        "testdb",
		port:          "5440",
	}
	dsnString = createDSNString(dsn)

	// test with a postgres database container set up
	closeContainer, err := startPostgresContainer(dsn)
	if err != nil {
		log.Fatal(err)
	}

	// run tests
	exitCode := m.Run()
	// close test postgres contiainer
	closeContainer()

	os.Exit(exitCode)
}

type dsnConf struct {
	hostName      string
	containerName string
	user          string
	password      string
	dbName        string
	port          string
}

// startPostgresContainer creates a test postgres container using docker
func startPostgresContainer(dsn dsnConf) (func(), error) {
	cmd := exec.Command("docker", "run", "-d",
		"--name", dsn.containerName,
		"-e", fmt.Sprintf("POSTGRES_USER=%s", dsn.user),
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", dsn.password),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", dsn.dbName),
		"-p", fmt.Sprintf("%s:5432", dsn.port),
		"postgres:18",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return func() {}, fmt.Errorf("failed to start postgres container: %v, output: %s", err, out)
	}

	close := func() {
		// kill the postgres container
		cmd := exec.Command("docker", "kill", fmt.Sprintf("%s", dsn.containerName))
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("failed to kill postgres container: %v, output: %s", err, out)
		}
		// remove the postgres container
		cmd = exec.Command("docker", "rm", fmt.Sprintf("%s", dsn.containerName))
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("failed to remove postgres container: %v, output: %s", err, out)
		}
	}

	return close, nil
}

// createDSNString takes a dsnConf struct and returns a formated dsn database string
func createDSNString(dsn dsnConf) string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s", dsn.hostName, dsn.port, dsn.dbName, dsn.user, dsn.password)
}

// slice of mocked postgres errors
var mockTests = []struct {
	name string
	err  error
	want bool
}{
	// --- Class 23: Integrity Constraint Violation ---
	{"unique violation", &pgconn.PgError{Code: "23505"}, false},
	{"forign key violation", &pgconn.PgError{Code: "23503"}, false},
	{"not null violation", &pgconn.PgError{Code: "23502"}, false},
	{"check violation", &pgconn.PgError{Code: "23514"}, false},
	{"exclusion violation", &pgconn.PgError{Code: "23P01"}, false},
	// --- Class 40: Transaction Rollback (Retryable) ---
	{"serialization failure", &pgconn.PgError{Code: "40001"}, true},
	{"deadlock_detected", &pgconn.PgError{Code: "40P01"}, true},
	// --- Class 22: Data Exception ---
	{"string data right truncation", &pgconn.PgError{Code: "22001"}, false},
	{"invalid text representation", &pgconn.PgError{Code: "22P02"}, false},
	{"numeric value out of range", &pgconn.PgError{Code: "22003"}, false},
	{"invalid datetime format", &pgconn.PgError{Code: "22007"}, false},
	{"division by zero", &pgconn.PgError{Code: "22012"}, false},
	// --- Class 42: Syntax Error or Access Rule Violation ---
	{"syntax error", &pgconn.PgError{Code: "42601"}, false},
	{"undefined table", &pgconn.PgError{Code: "42P01"}, false},
	{"undefined column", &pgconn.PgError{Code: "42703"}, false},
	{"insufficient privilege", &pgconn.PgError{Code: "42501"}, false},
	// --- Class 53 / 57 / 08: Resource, Operator Intervention & Connection ---
	{"query canceled", &pgconn.PgError{Code: "57014"}, false},
	{"too many connections", &pgconn.PgError{Code: "53300"}, false},
	{"admin shutdown", &pgconn.PgError{Code: "57P01"}, false},
	{"connection failure", &pgconn.PgError{Code: "08006"}, false},
	{"non pgconn.PgError", errors.New("test error"), false},
}
