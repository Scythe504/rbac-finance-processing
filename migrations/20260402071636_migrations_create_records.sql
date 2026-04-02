-- +goose Up
-- +goose StatementBegin
CREATE TYPE txn_type_enum AS ENUM ('income', 'expense');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS records (
  id SERIAL PRIMARY KEY,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  amount NUMERIC(19, 4) NOT NULL,
  txn_type txn_type_enum NOT NULL,
  category TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_records_user_id ON records(user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_records_category ON records(category);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_records_txn_type ON records(txn_type);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_records_created_at ON records(created_at);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_records_created_at;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_records_txn_type;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_records_category;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_records_user_id;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS records;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TYPE IF EXISTS txn_type_enum;
-- +goose StatementEnd