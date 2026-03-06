package request

import (
	"net"
	"testing"

	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

func TestEffectiveClientIP_PreferECS(t *testing.T) {
	state := Request{
		Req: new(dns.Msg),
		W:   &test.ResponseWriter{RemoteIP: "10.240.0.1"},
	}

	state.Req.SetEdns0(4096, false)
	if opt := state.Req.IsEdns0(); opt != nil {
		opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
			Code:          dns.EDNS0SUBNET,
			Family:        1,
			SourceNetmask: 32,
			Address:       net.ParseIP("61.142.56.193"),
		})
	}

	ip, source, reason := state.EffectiveClientIP(true)
	if ip.String() != "61.142.56.193" {
		t.Fatalf("expected ECS IP, got %s", ip.String())
	}
	if source != ClientIPSourceECS {
		t.Fatalf("expected source %q, got %q", ClientIPSourceECS, source)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}

func TestEffectiveClientIP_NoECSFallback(t *testing.T) {
	state := Request{
		Req: new(dns.Msg),
		W:   &test.ResponseWriter{RemoteIP: "10.240.0.1"},
	}

	ip, source, reason := state.EffectiveClientIP(true)
	if ip.String() != "10.240.0.1" {
		t.Fatalf("expected resolver IP fallback, got %s", ip.String())
	}
	if source != ClientIPSourceFallback {
		t.Fatalf("expected source %q, got %q", ClientIPSourceFallback, source)
	}
	if reason != ClientIPFallbackECSMissing {
		t.Fatalf("expected reason %q, got %q", ClientIPFallbackECSMissing, reason)
	}
}

func TestEffectiveClientIP_MalformedECSFallback(t *testing.T) {
	state := Request{
		Req: new(dns.Msg),
		W:   &test.ResponseWriter{RemoteIP: "10.240.0.1"},
	}

	state.Req.SetEdns0(4096, false)
	if opt := state.Req.IsEdns0(); opt != nil {
		opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
			Code:          dns.EDNS0SUBNET,
			Family:        1,
			SourceNetmask: 40,
			Address:       net.ParseIP("61.142.56.193"),
		})
	}

	ip, source, reason := state.EffectiveClientIP(true)
	if ip.String() != "10.240.0.1" {
		t.Fatalf("expected resolver IP fallback, got %s", ip.String())
	}
	if source != ClientIPSourceFallback {
		t.Fatalf("expected source %q, got %q", ClientIPSourceFallback, source)
	}
	if reason != ClientIPFallbackECSMalformed {
		t.Fatalf("expected reason %q, got %q", ClientIPFallbackECSMalformed, reason)
	}
}

func TestEffectiveClientIP_IPv6Prefix(t *testing.T) {
	state := Request{
		Req: new(dns.Msg),
		W:   &test.ResponseWriter{RemoteIP: "127.0.0.1"},
	}

	state.Req.SetEdns0(4096, false)
	if opt := state.Req.IsEdns0(); opt != nil {
		opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
			Code:          dns.EDNS0SUBNET,
			Family:        2,
			SourceNetmask: 56,
			Address:       net.ParseIP("2406:8c00:0:3401:133:18:168:70"),
		})
	}

	ip, source, reason := state.EffectiveClientIP(true)
	if ip.String() != "2406:8c00:0:3400::" {
		t.Fatalf("expected masked IPv6 ECS IP, got %s", ip.String())
	}
	if source != ClientIPSourceECS {
		t.Fatalf("expected source %q, got %q", ClientIPSourceECS, source)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}
