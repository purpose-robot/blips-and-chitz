-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text        NOT NULL,
    email         citext      NOT NULL UNIQUE,
    password_hash bytea       NOT NULL,
    activated     boolean     NOT NULL DEFAULT false,
    version       bigint      NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

-- +goose Down
DROP TABLE users;
