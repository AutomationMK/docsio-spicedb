-- +goose Up
Create TABLE permissions_outbox (
  id BIGSERIAL PRIMARY KEY,
  op TEXT NOT NULL CHECK (op IN ('TOUCH', 'DELETE')),
  resource TEXT NOT NULL,
  relation TEXT NOT NULL,
  subject TEXT NOT NULL,
  caveat_name TEXT,
  caveat_context JSONB,
  created_at TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ
);

CREATE INDEX permissions_outbox_unprocessed_idx
  ON permissions_outbox (id)
  WHERE processed_at is NULL;

-- +goose Down
DROP TABLE (IF EXISTS) permissions_outbox [CASCADE | RESTRICT];
