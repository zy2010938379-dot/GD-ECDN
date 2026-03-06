package geoip

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

const pluginName = "geoip"

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	geoip, err := geoipParse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		geoip.Next = next
		return geoip
	})

	return nil
}

func geoipParse(c *caddy.Controller) (*GeoIP, error) {
	var dbPath string
	var edns0 bool
	fallbackPolicy := ECSFallbackPolicyResolverIP
	goedgeCity := false

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) > 1 {
			return nil, c.ArgErr()
		}
		if len(args) == 1 {
			if dbPath != "" {
				return nil, c.Errf("configuring multiple databases is not supported")
			}
			dbPath = args[0]
		} else if dbPath != "" {
			return nil, c.Errf("configuring multiple databases is not supported")
		}

		for c.NextBlock() {
			switch c.Val() {
			case "edns-subnet":
				if len(c.RemainingArgs()) != 0 {
					return nil, c.ArgErr()
				}
				edns0 = true
			case "ecs-fallback":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				switch ECSFallbackPolicy(args[0]) {
				case ECSFallbackPolicyResolverIP, ECSFallbackPolicyDisabled:
					fallbackPolicy = ECSFallbackPolicy(args[0])
				default:
					return nil, c.Errf("unknown ecs-fallback policy %q", args[0])
				}
			case "goedge-city":
				if len(c.RemainingArgs()) != 0 {
					return nil, c.ArgErr()
				}
				goedgeCity = true
			default:
				return nil, c.Errf("unknown property %q", c.Val())
			}
		}
	}

	if dbPath == "" && !goedgeCity {
		return nil, c.ArgErr()
	}

	geoIP, err := newGeoIP(dbPath, edns0, fallbackPolicy, goedgeCity)
	if err != nil {
		return geoIP, c.Err(err.Error())
	}
	return geoIP, nil
}
