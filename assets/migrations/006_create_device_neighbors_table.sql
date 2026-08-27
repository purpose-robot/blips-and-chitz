-- +goose Up
CREATE TABLE device_neighbors (
    id           uuid    PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id    uuid    NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    local_port   text    NOT NULL,
    remote_port  text    NOT NULL,
    capabilities text[]  NOT NULL DEFAULT '{}',
    mac          macaddr,
    model        text    NOT NULL DEFAULT '',
    mgmt_ip      inet,
    hostname     text    NOT NULL DEFAULT '',

    UNIQUE (device_id, local_port, remote_port)
);

ALTER TABLE device_neighbors
    ADD CONSTRAINT device_neighbors_capabilities_check
    CHECK (capabilities <@ ARRAY['switch', 'router', 'access_point']);

-- +goose Down
DROP TABLE device_neighbors;
