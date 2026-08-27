-- +goose Up
CREATE TABLE devices (
    id                        uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    version                   bigint      NOT NULL DEFAULT 1,
    created_at                timestamptz NOT NULL DEFAULT now(),
    tags                      text[]      NOT NULL DEFAULT '{}',
    mgmt_ip                   inet        NOT NULL UNIQUE,
    platform                  text        NOT NULL,
    observed_mac              macaddr,
    observed_hostname         text,
    observed_software_version text,
    sync_error                text        NOT NULL DEFAULT '',
    last_sync_at              timestamptz,
    last_seen_at              timestamptz
);

CREATE INDEX devices_tags_idx ON devices USING gin (tags);

ALTER TABLE devices
    ADD CONSTRAINT devices_platform_check
    CHECK (platform IN ('aos_cx', 'ios_xe'));

ALTER TABLE devices
    ADD CONSTRAINT devices_observed_all_or_none_check
    CHECK (num_nonnulls(observed_mac, observed_hostname, observed_software_version) IN (0, 3));

-- +goose Down
DROP TABLE devices;
