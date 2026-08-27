-- +goose Up
CREATE TABLE tokens (
    hash       bytea       PRIMARY KEY,
    scope      text        NOT NULL,
    user_id    uuid        NOT NULL REFERENCES users ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE tokens;
