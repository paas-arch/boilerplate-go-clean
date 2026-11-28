-- +goose Up
-- +goose StatementBegin
CREATE TABLE notes (
    id         bigserial PRIMARY KEY,
    text       text      NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW()
);

-- Список заметок идёт от новых к старым.
CREATE INDEX idx_notes_created_at ON notes (created_at DESC);
-- +goose StatementEnd
