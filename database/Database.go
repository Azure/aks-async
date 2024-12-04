package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	log "log/slog"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/microsoft/go-mssqldb/azuread"
)

// TODO(mheberling): Make interfaces like service bus to use other types of db.
// Create a database connection using the database name, server, and port. Must be logged in to azure cli.
func NewDbClient(ctx context.Context, server string, port int, database string) (*sql.DB, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating a db client.")

	// Build connection string
	connString := fmt.Sprintf("server=%s;port%d;database=%s;fedauth=ActiveDirectoryDefault;", server, port, database) // Working because we're logged into azure.

	db, err := sql.Open(azuread.DriverName, connString)
	if err != nil {
		logger.Error("Error creating connection pool: " + err.Error())
		return nil, err
	}

	// Pinging to check that we do have access.
	err = db.PingContext(ctx)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	} else {
		logger.Info("Connected!")
	}

	return db, nil
}

// Create a dabatase connection using a connection string.
func NewDbClientWithConnectionString(ctx context.Context, connectionstring string) (*sql.DB, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating a db client.")

	db, err := sql.Open(azuread.DriverName, connectionstring)
	if err != nil {
		logger.Error("Error creating connection pool: " + err.Error())
		return nil, err
	}

	// Pinging to check that we do have access.
	err = db.PingContext(ctx)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	} else {
		logger.Info("Connected!")
	}

	return db, nil
}

// TODO(mheberling): Change this to return something more digestible than sql.Rows?
// Query the database, appropriate for "SELECT" methods.
func QueryDb(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Querying db.")
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Info("Error executing query: " + query + ". With error: " + err.Error())
		return nil, err
	}

	return rows, nil
}

// Execute a query for "INSERT", "UPDATE", or "DELETE" methods which affect rows.
func ExecDb(ctx context.Context, db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Executing query to db.")
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Info("Error executing query: " + query + ". With error: " + err.Error())
		return nil, err
	}

	if rows, err := result.RowsAffected(); rows == 0 {
		log.Error("No rows were affected!")
		return nil, errors.New("No rows were affected!")
	} else if err != nil {
		log.Error("Error checking the number of affected rows.")
		return nil, err
	}

	return result, nil
}
