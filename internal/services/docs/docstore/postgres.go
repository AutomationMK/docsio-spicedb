package docstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type docStorePostgres struct {
	DB *pgxpool.Pool
}

func createPostgresDocStore(conn *pgxpool.Pool) docStorePostgres {
	return docStorePostgres{
		DB: conn,
	}
}

func (s *docStorePostgres) ShareDoc(ctx context.Context, docID,
	byUserID, granteeID, level string) error {
	return nil
}
