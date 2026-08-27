-- +goose Up
CREATE TABLE device_stack_members (
    device_id     uuid    NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    mac_address   macaddr NOT NULL,
    slot          integer NOT NULL,
    role          text    NOT NULL,
    model         text    NOT NULL,
    serial_number text    NOT NULL,

    PRIMARY KEY (device_id, slot)
);

ALTER TABLE device_stack_members
    ADD CONSTRAINT device_stack_members_role_check
    CHECK (role IN ('member', 'primary', 'standby'));

CREATE INDEX device_stack_members_mac_address_idx ON device_stack_members (mac_address);
CREATE INDEX device_stack_members_serial_number_idx ON device_stack_members (serial_number);

-- +goose Down
DROP TABLE device_stack_members;
