//go:build !mdns

package addr

import (
	"context"
	"errors"
)

var ErrMDNSDisabled = errors.New("addr: compilado sin mdns")

type MDNS struct{}

func NewMDNS(*LAN, int) (*MDNS, error)                             { return nil, ErrMDNSDisabled }
func (*MDNS) Kind() string                                         { return "mdns" }
func (*MDNS) Addresses(context.Context, string) ([]Address, error) { return nil, nil }
func (*MDNS) Close()                                               {}
