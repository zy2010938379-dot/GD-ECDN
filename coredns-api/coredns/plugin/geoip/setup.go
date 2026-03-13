package geoip

import (
	"strings"
	"time"

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
	goedgeCityMySQL := GoEdgeCityMySQLConfig{}

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
			case "goedge-city-mysql-dsn":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				goedgeCityMySQL.DSN = args[0]
			case "goedge-city-mysql-table":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				goedgeCityMySQL.Table = args[0]
			case "goedge-city-mysql-query":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				goedgeCityMySQL.Query = args[0]
			case "goedge-city-mysql-refresh":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				dur, err := time.ParseDuration(args[0])
				if err != nil {
					return nil, c.Errf("invalid goedge-city-mysql-refresh value %q: %v", args[0], err)
				}
				goedgeCityMySQL.RefreshInterval = dur
			case "goedge-city-mysql-timeout":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return nil, c.ArgErr()
				}
				dur, err := time.ParseDuration(args[0])
				if err != nil {
					return nil, c.Errf("invalid goedge-city-mysql-timeout value %q: %v", args[0], err)
				}
				goedgeCityMySQL.QueryTimeout = dur
			default:
				return nil, c.Errf("unknown property %q", c.Val())
			}
		}
	}

	if dbPath == "" && !goedgeCity {
		return nil, c.ArgErr()
	}

	if goedgeCityMySQL.Enabled() && !goedgeCity {
		return nil, c.Errf("goedge-city-mysql-* requires goedge-city to be enabled")
	}
	if !goedgeCityMySQL.Enabled() && (strings.TrimSpace(goedgeCityMySQL.Table) != "" || strings.TrimSpace(goedgeCityMySQL.Query) != "" || goedgeCityMySQL.RefreshInterval > 0 || goedgeCityMySQL.QueryTimeout > 0) {
		return nil, c.Errf("goedge-city-mysql-dsn is required when mysql options are set")
	}

	geoIP, err := newGeoIPWithMySQLConfig(dbPath, edns0, fallbackPolicy, goedgeCity, goedgeCityMySQL)
	if err != nil {
		return geoIP, c.Err(err.Error())
	}
	return geoIP, nil
}
