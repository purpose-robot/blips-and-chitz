package cisco

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/purpose-robot/blips-and-chitz/devices"
	"github.com/purpose-robot/blips-and-chitz/internal/netconf"
)

type DeviceGateway struct {
	username string
	password string
}

func NewDeviceGateway(username, password string) *DeviceGateway {
	return &DeviceGateway{
		username: username,
		password: password,
	}
}

func (d *DeviceGateway) GatherFacts(ctx context.Context, ipAddress netip.Addr) (*devices.Facts, error) {
	session, err := netconf.Dial(ctx, ipAddress.String(), d.username, d.password)
	if err != nil {
		return nil, fmt.Errorf("ext.cisco.gatherFacts: %w", err)
	}

	defer func() {
		_ = session.Close(context.WithoutCancel(ctx))
	}()

	reply := new(factsReply)

	err = session.Get(ctx, filterFacts, reply)
	if err != nil {
		return nil, fmt.Errorf("ext.cisco.gatherFacts: %w", err)
	}

	return reply.parse(), nil
}
