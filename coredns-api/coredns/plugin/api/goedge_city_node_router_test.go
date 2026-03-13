package api

import (
	"testing"

	pkgrand "github.com/coredns/coredns/plugin/pkg/rand"
	"github.com/miekg/dns"
)

func TestGoedgeCityNodeRouterSelectNodeIPRegionFirstAndFallback(t *testing.T) {
	t.Parallel()

	router := &goedgeCityNodeRouter{
		rand: pkgrand.New(1),
		snapshot: goedgeCityNodeSnapshot{
			fqdn: "edge.example.com.",
			regionIPv4: map[int64][]string{
				8: {"203.0.113.11"},
			},
			allIPv4: []string{"203.0.113.20"},
		},
	}

	t.Run("region-hit", func(t *testing.T) {
		ip, ok := router.selectNodeIP("edge.example.com.", dns.TypeA, 8)
		if !ok {
			t.Fatalf("expected region hit")
		}
		if ip != "203.0.113.11" {
			t.Fatalf("expected region ip 203.0.113.11, got %s", ip)
		}
	})

	t.Run("region-miss-fallback-random-all", func(t *testing.T) {
		ip, ok := router.selectNodeIP("edge.example.com.", dns.TypeA, 99)
		if !ok {
			t.Fatalf("expected fallback hit")
		}
		if ip != "203.0.113.20" {
			t.Fatalf("expected fallback ip 203.0.113.20, got %s", ip)
		}
	})
}

func TestGoedgeCityNodeRouterSelectNodeIPQNameMismatch(t *testing.T) {
	t.Parallel()

	router := &goedgeCityNodeRouter{
		rand: pkgrand.New(1),
		snapshot: goedgeCityNodeSnapshot{
			fqdn:    "edge.example.com.",
			allIPv4: []string{"203.0.113.20"},
		},
	}

	if _, ok := router.selectNodeIP("other.example.com.", dns.TypeA, 0); ok {
		t.Fatalf("expected qname mismatch not handled")
	}
}
