package geoip

import (
	"context"
	"net/netip"

	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/request"
)

func (g GeoIP) resolveEffectiveClientIP(state request.Request) (netip.Addr, string, string, bool) {
	ip, source, fallbackReason := state.EffectiveClientIP(g.edns0)
	if !ip.IsValid() {
		log.Debugf("Failed to parse effective client IP, source=%q reason=%q remote=%q", source, fallbackReason, state.IP())
		return netip.Addr{}, source, fallbackReason, false
	}

	if source == request.ClientIPSourceFallback && g.edns0 && g.ecsFallbackPolicy == ECSFallbackPolicyDisabled {
		log.Debugf("Skip geoip lookup due to fallback policy disabled, reason=%q remote=%q", fallbackReason, state.IP())
		return netip.Addr{}, source, fallbackReason, false
	}

	return ip, source, fallbackReason, true
}

func (g GeoIP) setClientIPMetadata(ctx context.Context, clientIP, source, fallbackReason string) {
	metadata.SetValueFunc(ctx, pluginName+"/client/ip", func() string {
		return clientIP
	})
	metadata.SetValueFunc(ctx, pluginName+"/client/ip_source", func() string {
		return source
	})
	metadata.SetValueFunc(ctx, pluginName+"/client/ip_fallback_reason", func() string {
		return fallbackReason
	})
}
