package request

import (
	"net/netip"

	"github.com/miekg/dns"
)

const (
	ClientIPSourceECS      = "ecs"
	ClientIPSourceFallback = "fallback"

	ClientIPFallbackECSDisabled   = "ecs_disabled"
	ClientIPFallbackECSMissing    = "ecs_missing"
	ClientIPFallbackECSMalformed  = "ecs_malformed"
	ClientIPFallbackRemoteInvalid = "remote_ip_invalid"
)

// EffectiveClientIP returns the client IP used for routing decisions.
// When preferECS is true it tries ECS first and falls back to resolver source IP.
func (r *Request) EffectiveClientIP(preferECS bool) (netip.Addr, string, string) {
	fallbackIP, fallbackOK := r.remoteClientIP()

	if !preferECS {
		if !fallbackOK {
			return netip.Addr{}, ClientIPSourceFallback, ClientIPFallbackRemoteInvalid
		}
		return fallbackIP, ClientIPSourceFallback, ClientIPFallbackECSDisabled
	}

	ecsIP, ecsReason := r.ecsClientIP()
	if ecsIP.IsValid() {
		return ecsIP, ClientIPSourceECS, ""
	}

	if !fallbackOK {
		return netip.Addr{}, ClientIPSourceFallback, ClientIPFallbackRemoteInvalid
	}

	return fallbackIP, ClientIPSourceFallback, ecsReason
}

func (r *Request) remoteClientIP() (netip.Addr, bool) {
	ip := r.IP()
	if len(ip) == 0 {
		return netip.Addr{}, false
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return netip.Addr{}, false
	}

	return addr.Unmap(), true
}

func (r *Request) ecsClientIP() (netip.Addr, string) {
	if r.Req == nil {
		return netip.Addr{}, ClientIPFallbackECSMissing
	}

	opt := r.Req.IsEdns0()
	if opt == nil {
		return netip.Addr{}, ClientIPFallbackECSMissing
	}

	for _, option := range opt.Option {
		subnet, ok := option.(*dns.EDNS0_SUBNET)
		if !ok {
			continue
		}

		addr, ok := netip.AddrFromSlice(subnet.Address)
		if !ok {
			return netip.Addr{}, ClientIPFallbackECSMalformed
		}
		addr = addr.Unmap()

		var totalBits int
		switch subnet.Family {
		case 1:
			totalBits = 32
			if !addr.Is4() {
				return netip.Addr{}, ClientIPFallbackECSMalformed
			}
		case 2:
			totalBits = 128
			if !addr.Is6() {
				return netip.Addr{}, ClientIPFallbackECSMalformed
			}
		case 0:
			// Some clients don't set Family correctly, infer from address to keep compatibility.
			if addr.Is4() {
				totalBits = 32
			} else if addr.Is6() {
				totalBits = 128
			} else {
				return netip.Addr{}, ClientIPFallbackECSMalformed
			}
		default:
			return netip.Addr{}, ClientIPFallbackECSMalformed
		}

		prefixLen := int(subnet.SourceNetmask)
		if prefixLen > totalBits {
			return netip.Addr{}, ClientIPFallbackECSMalformed
		}

		prefix := netip.PrefixFrom(addr, prefixLen).Masked()
		return prefix.Addr(), ""
	}

	return netip.Addr{}, ClientIPFallbackECSMissing
}
