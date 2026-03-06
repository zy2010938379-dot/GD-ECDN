package geoip

type ECSFallbackPolicy string

const (
	ECSFallbackPolicyResolverIP ECSFallbackPolicy = "resolver-ip"
	ECSFallbackPolicyDisabled   ECSFallbackPolicy = "disabled"
)
