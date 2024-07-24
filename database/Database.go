package database

import (
	"context"
	"database/sql"
	"fmt"
	log "log/slog"

	"github.com/Azure/aks-middleware/ctxlogger"
	"github.com/microsoft/go-mssqldb/azuread"
)

// TODO(mheberling): Make interfaces like service bus to use other types of db.
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
func QueryDb(ctx context.Context, db *sql.DB, query string) (*sql.Rows, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Querying db.")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Info("Error executing query: " + query + ". With error: " + err.Error())
		return nil, err
	}

	return rows, nil
}
