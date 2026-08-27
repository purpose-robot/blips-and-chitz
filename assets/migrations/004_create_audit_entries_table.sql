-- +goose Up
CREATE TABLE audit_entries ();

-- +goose Down
DROP TABLE audit_entries;
