package common

import (
	"net/netip"
	"strings"
)

func GetAddress(staticIp string) (netip.Addr, error) {
	var address netip.Addr

	if strings.TrimSpace(staticIp) == "" {
		return address, nil
	}

	return netip.ParseAddr(staticIp)
}
