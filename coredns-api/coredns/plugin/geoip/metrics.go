package geoip

import (
	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	effectiveClientIPSourceCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "geoip",
		Name:      "effective_client_ip_source_total",
		Help:      "Counter of selected effective client IP source.",
	}, []string{"source"})

	effectiveClientIPFallbackReasonCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "geoip",
		Name:      "effective_client_ip_fallback_reason_total",
		Help:      "Counter of ECS fallback reasons.",
	}, []string{"reason"})

	cityLookupCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "geoip",
		Name:      "city_lookup_total",
		Help:      "Counter of city lookup result from mmdb.",
	}, []string{"result"})

	goedgeCityLookupCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "geoip",
		Name:      "goedge_city_lookup_total",
		Help:      "Counter of city lookup result from GoEdge IP library.",
	}, []string{"result"})
)

func (g GeoIP) observeClientIPSource(source, fallbackReason string) {
	effectiveClientIPSourceCount.WithLabelValues(source).Inc()
	if fallbackReason != "" {
		effectiveClientIPFallbackReasonCount.WithLabelValues(fallbackReason).Inc()
	}
}

func (g GeoIP) observeCityLookup(hit bool) {
	result := "miss"
	if hit {
		result = "hit"
	}
	cityLookupCount.WithLabelValues(result).Inc()
}

func (g GeoIP) observeGoEdgeCityLookup(result GoEdgeCityMappingResult) {
	status := "miss"
	if result.Hit {
		status = "hit"
	}
	goedgeCityLookupCount.WithLabelValues(status).Inc()
}
