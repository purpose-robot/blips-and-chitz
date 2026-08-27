-- +goose Up
CREATE TABLE permissions (
    id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL
);

CREATE TABLE users_permissions (
    user_id       uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    permission_id uuid NOT NULL REFERENCES permissions ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

INSERT INTO permissions (code)
VALUES
    ('health:read');

-- +goose Down
DROP TABLE users_permissions;
DROP TABLE permissions;
