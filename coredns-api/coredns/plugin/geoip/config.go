package geoip

import "time"

type ECSFallbackPolicy string

const (
	ECSFallbackPolicyResolverIP ECSFallbackPolicy = "resolver-ip"
	ECSFallbackPolicyDisabled   ECSFallbackPolicy = "disabled"
)

const (
	DefaultGoEdgeCityMySQLTable        = "edns_city_mapping"
	DefaultGoEdgeCityMySQLRefresh      = 60 * time.Second
	DefaultGoEdgeCityMySQLQueryTimeout = 3 * time.Second
)

type GoEdgeCityMySQLConfig struct {
	DSN             string
	Table           string
	Query           string
	RefreshInterval time.Duration
	QueryTimeout    time.Duration
}

func (c GoEdgeCityMySQLConfig) Enabled() bool {
	return c.DSN != ""
}
