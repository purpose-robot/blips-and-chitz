package netconf

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	scrapligonetconf "github.com/scrapli/scrapligo/v2/netconf"
	scrapligooptions "github.com/scrapli/scrapligo/v2/options"
)

type Session struct {
	host   string
	driver *scrapligonetconf.Netconf
}

func (s *Session) Get(ctx context.Context, filter string, reply any) error {
	opts := []scrapligonetconf.Option{
		scrapligonetconf.WithFilter(filter),
	}

	results, err := s.driver.Get(ctx, opts...)
	if err != nil {
		return fmt.Errorf("netconf.get: device %s: %v", s.host, err)
	}

	if results.Failed {
		return fmt.Errorf("netconf.get: device %s rejected rpc: %s", s.host, strings.Join(results.Errors, "; "))
	}

	err = xml.Unmarshal([]byte(results.Result), reply)
	if err != nil {
		return fmt.Errorf("netconf.get: device %s: parsing reply: %v", s.host, err)
	}

	return nil
}

func (s *Session) Close(ctx context.Context) error {
	results, err := s.driver.Close(ctx)
	if err != nil {
		return fmt.Errorf("netconf.close: device %s: %v", s.host, err)
	}

	if results.Failed {
		return fmt.Errorf("netconf.close: device %s rejected rpc: %s", s.host, strings.Join(results.Errors, "; "))
	}

	return nil
}

func Dial(ctx context.Context, host, username, password string) (*Session, error) {
	driver, err := scrapligonetconf.NewNetconf(
		host,
		scrapligooptions.WithUsername(username),
		scrapligooptions.WithPassword(password),
	)
	if err != nil {
		return nil, fmt.Errorf("netconf.dial: device %s: %v", host, err)
	}

	session := &Session{host: host, driver: driver}

	results, err := driver.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("netconf.dial: device %s: %v", host, err)
	}

	if results.Failed {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("netconf.dial: device %s rejected session: %s", host, strings.Join(results.Errors, "; "))
	}

	return session, nil
}
