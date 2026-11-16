-- +goose Up
Create TABLE doc_shares (
  id BIGSERIAL PRIMARY KEY,
  doc_id TEXT NOT NULL,
  grantee_id TEXT NOT NULL,
  level TEXT NOT NULL,
  granted_by TEXT NOT NULL,
  granted_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX doc_shares_unique_idx
  ON doc_shares (doc_id, grantee_id);

-- +goose Down
DROP TABLE (IF EXISTS) doc_shares [CASCADE | RESTRICT];
