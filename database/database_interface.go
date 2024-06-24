package database

import (
	"context"
)

type DatabaseClientInterface interface { // TODO(mheberling): Change the name
	CreateDbClient(ctx context.Context) (DatabaseClientInterface, error)
	QueryDb(ctx context.Context, query string) (interface{}, error)
}
