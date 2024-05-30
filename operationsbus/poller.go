package operationsbus

import (
	"context"

	"github.com/Azure/aks-middleware/ctxlogger"
	db "go.goms.io/aks/rp/mygreeterv3/server/internal/toolkit/database"
)

type Poller struct {
	operationId      string
	query            string // This is the query to get the status of the operation from the db
	status           string //TODO(mheberling): This should be an array of possible terminal results.
	connectionstring string
}

func NewPoller(ctx context.Context, operationId string, query string, status string, connectionstring string) (*Poller, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Creating a new poller.")
	return &Poller{
		operationId:      operationId,
		query:            query,
		status:           status,
		connectionstring: connectionstring, //TODO(mheberling): Remove this
	}, nil
}

func (p *Poller) Poll(ctx context.Context) (bool, error) {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Polling the operation status.")

	//TODO(mheberling): Change to add the specific table and db
	dbClient, err := db.NewDbClient(ctx, "heberling.database.windows.net", 1433, "hcp_servicehub")
	if err != nil {
		return false, err
	}

	rows, err := db.QueryDb(ctx, dbClient, p.query)
	if err != nil {
		return false, err
	}

	var operationStatus string
	for rows.Next() {
		err := rows.Scan(&operationStatus)
		if err != nil {
			logger.Info("Error scanning row: " + err.Error())
			return false, err
		}
	}

	if operationStatus != p.status {
		return false, nil
	}

	return true, nil
}

func (p *Poller) CheckUntilDone(ctx context.Context) error {
	logger := ctxlogger.GetLogger(ctx)
	logger.Info("Polling the operation status until it's done.")
	for {
		finished, err := p.Poll(ctx)
		if err != nil {
			logger.Error("Error polling the operation status: " + err.Error())
			return err
		}

		if finished {
			logger.Info("Operation finished!")
			return nil
		}
	}
}
