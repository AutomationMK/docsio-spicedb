package docstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DocStore interface {
	ShareDoc(ctx context.Context, docID, byUserID, granteeID, level string) error
}

func CreateDocStore(conn *pgxpool.Pool) DocStore {
	return createPostgresDocStore(conn)
}
