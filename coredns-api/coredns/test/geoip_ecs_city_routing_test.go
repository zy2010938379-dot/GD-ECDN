package test

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	"github.com/miekg/dns"
)

func TestGeoIPECSCityRoutingCityFirstAndFallback(t *testing.T) {
	const citySampleIP = "61.142.56.193"
	cityID := lookupCityID(t, citySampleIP)

	// Use a temporary instance to reserve one shared port for all server blocks.
	tmp, addr, _, err := CoreDNSServerAndPorts(`edge.example:0 {
		erratic
	}`)
	if err != nil {
		t.Fatalf("Could not get CoreDNS serving instance: %s", err)
	}
	port := addr[strings.LastIndex(addr, ":")+1:]
	tmp.Stop()

	cityUpstream, err := CoreDNSServer(`edge.example:0 {
		hosts {
			203.0.113.10 edge.example
			fallthrough
		}
	}`)
	if err != nil {
		t.Fatalf("Could not get city upstream instance: %s", err)
	}
	defer cityUpstream.Stop()

	cityUpstreamAddr, _ := CoreDNSServerPorts(cityUpstream, 0)
	if cityUpstreamAddr == "" {
		t.Fatalf("Could not get city upstream UDP listening port")
	}

	corefile := `
edge.example:` + port + ` {
	view city-guangzhou {
		expr metadata('geoip/goedge/city/id') == '` + cityID + `'
	}
	geoip {
		edns-subnet
		ecs-fallback resolver-ip
		goedge-city
	}
	metadata
	# 127.0.0.1:1 is intentionally unavailable to assert we skip unhealthy candidate.
	forward . 127.0.0.1:1 ` + cityUpstreamAddr + ` {
		policy sequential
		max_fails 1
		health_check 5ms
	}
}

edge.example:` + port + ` {
	hosts {
		203.0.113.2 edge.example
		fallthrough
	}
}
`

	i, queryAddr, _, err := CoreDNSServerAndPorts(corefile)
	if err != nil {
		t.Fatalf("Could not get CoreDNS serving instance: %s", err)
	}
	defer i.Stop()

	t.Run("city-hit-uses-city-candidates", func(t *testing.T) {
		resp := queryWithECS(t, queryAddr, "edge.example.", citySampleIP, 1, 32)
		gotIP := mustSingleA(t, resp)
		if gotIP != "203.0.113.10" {
			t.Fatalf("expected city candidate IP 203.0.113.10, got %s", gotIP)
		}
	})

	t.Run("city-miss-falls-back-default", func(t *testing.T) {
		resp := queryWithECS(t, queryAddr, "edge.example.", "203.0.113.1", 1, 32)
		gotIP := mustSingleA(t, resp)
		if gotIP != "203.0.113.2" {
			t.Fatalf("expected fallback IP 203.0.113.2, got %s", gotIP)
		}
	})

	t.Run("no-ecs-keeps-legacy-routing", func(t *testing.T) {
		resp := queryWithECS(t, queryAddr, "edge.example.", "", 0, 0)
		gotIP := mustSingleA(t, resp)
		if gotIP != "203.0.113.2" {
			t.Fatalf("expected fallback IP 203.0.113.2 without ECS, got %s", gotIP)
		}
	})

	t.Run("malformed-ecs-keeps-valid-response", func(t *testing.T) {
		// family=2 with IPv4 payload is wire-valid but should be treated as malformed by effective IP parsing.
		resp := queryWithECS(t, queryAddr, "edge.example.", citySampleIP, 2, 32)
		gotIP := mustSingleA(t, resp)
		if gotIP != "203.0.113.2" {
			t.Fatalf("expected fallback IP 203.0.113.2 on malformed ECS, got %s", gotIP)
		}
	})
}

func queryWithECS(t *testing.T, addr, qname, ecsIP string, family uint16, sourceMask uint8) *dns.Msg {
	t.Helper()

	msg := new(dns.Msg)
	msg.SetQuestion(qname, dns.TypeA)
	msg.SetEdns0(4096, false)

	if opt := msg.IsEdns0(); opt != nil && ecsIP != "" {
		opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
			Code:          dns.EDNS0SUBNET,
			Family:        family,
			SourceNetmask: sourceMask,
			Address:       netParseIP(t, ecsIP),
		})
	}

	client := &dns.Client{Timeout: 200 * time.Millisecond}
	resp, _, err := client.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("failed to exchange DNS message: %v", err)
	}
	if resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("expected success rcode, got %d", resp.Rcode)
	}
	return resp
}

func netParseIP(t *testing.T, ip string) []byte {
	t.Helper()
	parsed := net.ParseIP(ip)
	if parsed == nil {
		t.Fatalf("invalid ECS ip: %s", ip)
	}
	return parsed
}

func lookupCityID(t *testing.T, ip string) string {
	t.Helper()

	if err := iplibrary.InitDefault(); err != nil {
		t.Fatalf("failed to init goedge ip library: %v", err)
	}

	result := iplibrary.LookupIP(ip)
	if result == nil || !result.IsOk() || result.CityId() <= 0 {
		t.Fatalf("failed to resolve city id for ip %s", ip)
	}

	return strconv.FormatInt(result.CityId(), 10)
}

func mustSingleA(t *testing.T, resp *dns.Msg) string {
	t.Helper()
	for _, answer := range resp.Answer {
		if a, ok := answer.(*dns.A); ok {
			return a.A.String()
		}
	}
	t.Fatalf("expected at least one A answer, got %#v", resp.Answer)
	return ""
}
