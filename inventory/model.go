package inventory

import (
	"fmt"
	"net"
	"net/netip"
	"time"
	"uuid"
)

type Device struct {
	ID         uuid.UUID  `json:"id"`
	Version    int64      `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	Tags       []string   `json:"tags,omitzero"`
	MgmtIP     netip.Addr `json:"mgmt_ip"`
	Platform   Platform   `json:"platform"`
	Observed   *Observed  `json:"observed,omitzero"`
	SyncError  string     `json:"sync_error,omitzero"`
	LastSyncAt time.Time  `json:"last_sync_at,omitzero"`
	LastSeenAt time.Time  `json:"last_seen_at,omitzero"`
}

type SyncStatus string

const (
	StatusOK      SyncStatus = "ok"
	StatusStale   SyncStatus = "stale"
	StatusFailed  SyncStatus = "failed"
	StatusPending SyncStatus = "pending"
)

func (d Device) Status(maxAge time.Duration) SyncStatus {
	switch {
	case d.SyncError != "":
		return StatusFailed

	case d.LastSyncAt.IsZero():
		return StatusPending

	case time.Since(d.LastSyncAt) > maxAge:
		return StatusStale

	default:
		return StatusOK
	}
}

type Platform string

const (
	PlatformAOSCX Platform = "aos_cx"
	PlatformIOSXE Platform = "ios_xe"
)

type Observed struct {
	MAC             MAC           `json:"mac"`
	Hostname        string        `json:"hostname"`
	Neighbors       []Neighbor    `json:"neighbors,omitzero"`
	StackMembers    []StackMember `json:"stack_members,omitzero"`
	SoftwareVersion string        `json:"software_version"`
}

type MAC [6]byte

func (m MAC) String() string {
	return net.HardwareAddr(m[:]).String()
}

func (m MAC) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *MAC) UnmarshalText(t []byte) error {
	mac, err := ParseMAC(string(t))
	if err != nil {
		return err
	}

	*m = mac
	return nil
}

func ParseMAC(m string) (MAC, error) {
	mac, err := net.ParseMAC(m)
	if err != nil {
		return MAC{}, fmt.Errorf("parse mac address: %w", err)
	}

	if len(mac) != 6 {
		return MAC{}, fmt.Errorf("parse mac address %q: expected 6 bytes, got %d", m, len(mac))
	}

	return MAC(mac), nil
}

type StackMember struct {
	MAC          MAC        `json:"mac"`
	Slot         int        `json:"slot"`
	Role         MemberRole `json:"role"`
	Model        string     `json:"model"`
	SerialNumber string     `json:"serial_number"`
}

type MemberRole string

const (
	RoleMember  MemberRole = "member"
	RolePrimary MemberRole = "primary"
	RoleStandby MemberRole = "standby"
)

type Neighbor struct {
	LocalPort    string       `json:"local_port"`
	RemotePort   string       `json:"remote_port"`
	Capabilities []Capability `json:"capabilities"`

	MAC      MAC        `json:"mac,omitzero"`
	Model    string     `json:"model,omitzero"`
	MgmtIP   netip.Addr `json:"mgmt_ip,omitzero"`
	Hostname string     `json:"hostname,omitzero"`
}

type Capability string

const (
	CapabilitySwitch      Capability = "switch"
	CapabilityRouter      Capability = "router"
	CapabilityAccessPoint Capability = "access_point"
)
