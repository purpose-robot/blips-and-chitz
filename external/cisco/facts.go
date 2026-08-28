package cisco

import (
	"net/netip"
	"strings"

	"github.com/purpose-robot/blips-and-chitz/devices"
)

const filterFacts = `
	<native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native">
		<version/>
		<hostname/>
	</native>
	<device-hardware-data xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-device-hardware-oper">
		<device-hardware>
			<device-inventory/>
		</device-hardware>
	</device-hardware-data>
	<stack-oper-data xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-stack-oper"/>
	<cdp-neighbor-details xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-cdp-oper"/>`

const (
	roleActive    = "role-active"
	roleStandby   = "role-standby"
	hwTypeChassis = "hw-type-chassis"
)

type factsReply struct {
	Version           string            `xml:"data>native>version"`
	Hostname          string            `xml:"data>native>hostname"`
	StackMAC          string            `xml:"data>stack-oper-data>stack-info>stack-mac-address"`
	StackNodes        []stackNode       `xml:"data>stack-oper-data>stack-node"`
	CDPNeighbors      []cdpNeighbor     `xml:"data>cdp-neighbor-details>cdp-neighbor-detail"`
	DeviceInventories []deviceInventory `xml:"data>device-hardware-data>device-hardware>device-inventory"`
}

func (r *factsReply) parse() *devices.Facts {
	facts := &devices.Facts{
		Hostname:        r.Hostname,
		SoftwareVersion: r.Version,
	}

	if mac, err := devices.ParseMAC(r.StackMAC); err == nil {
		facts.MAC = mac
	}

	for _, detail := range r.CDPNeighbors {
		facts.Neighbors = append(facts.Neighbors, detail.parse())
	}

	models := r.chassisModels()

	for _, node := range r.StackNodes {
		facts.StackMembers = append(facts.StackMembers, node.parse(models[node.ChassisNumber]))
	}

	return facts
}

func (r *factsReply) chassisModels() map[int]string {
	models := make(map[int]string, len(r.DeviceInventories))

	for _, item := range r.DeviceInventories {
		if item.HWType == hwTypeChassis {
			models[item.HWDevIndex] = item.PartNumber
		}
	}

	return models
}

type deviceInventory struct {
	HWType     string `xml:"hw-type"`
	PartNumber string `xml:"part-number"`
	HWDevIndex int    `xml:"hw-dev-index"`
}

type stackNode struct {
	MAC           string `xml:"mac-address"`
	Role          string `xml:"role"`
	SerialNumber  string `xml:"serial-number"`
	ChassisNumber int    `xml:"chassis-number"`
}

func parseMemberRole(role string) devices.MemberRole {
	switch role {
	case roleActive:
		return devices.RolePrimary

	case roleStandby:
		return devices.RoleStandby

	default:
		return devices.RoleMember
	}
}

func (n stackNode) parse(model string) devices.StackMember {
	member := devices.StackMember{
		Slot:         n.ChassisNumber,
		Role:         parseMemberRole(n.Role),
		Model:        model,
		SerialNumber: n.SerialNumber,
	}

	if mac, err := devices.ParseMAC(n.MAC); err == nil {
		member.MAC = mac
	}

	return member
}

type cdpNeighbor struct {
	LocalIntfName   string `xml:"local-intf-name"`
	PortID          string `xml:"port-id"`
	IPAddress       string `xml:"ip-address"`
	DeviceName      string `xml:"device-name"`
	Capabilities    string `xml:"capability"`
	Platform        string `xml:"platform-name"`
	NeighborPortMAC string `xml:"neighbor-port-mac"`
}

func (n cdpNeighbor) parse() devices.Neighbor {
	neighbor := devices.Neighbor{
		LocalPort:    n.LocalIntfName,
		RemotePort:   n.PortID,
		Model:        n.Platform,
		Hostname:     n.DeviceName,
		Capabilities: parseCapabilities(n.Capabilities),
	}

	if ip, err := netip.ParseAddr(n.IPAddress); err == nil {
		neighbor.IPAddress = ip
	}

	if mac, err := devices.ParseMAC(n.NeighborPortMAC); err == nil {
		neighbor.MAC = mac
	}

	return neighbor
}

func parseCapabilities(names string) []devices.Capability {
	var capabilities []devices.Capability

	for name := range strings.FieldsSeq(names) {
		switch name {
		case "Switch":
			capabilities = append(capabilities, devices.CapabilitySwitch)

		case "Router":
			capabilities = append(capabilities, devices.CapabilityRouter)

		case "Trans-Bridge":
			capabilities = append(capabilities, devices.CapabilityAccessPoint)
		}
	}

	return capabilities
}
