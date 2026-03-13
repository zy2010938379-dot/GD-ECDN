package api

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miekg/dns"
)

func TestAddRecordToZoneReplacesExistingCNAMEWithSameName(t *testing.T) {
	t.Parallel()

	zoneFile := writeTempZoneFile(t, `
$ORIGIN example.com.
$TTL 3600

@       IN      SOA     ns1.example.com. admin.example.com. (
                        2024010101 ; serial
                        3600       ; refresh
                        1800       ; retry
                        604800     ; expire
                        86400 )    ; minimum

@       IN      NS      ns1.example.com.
@       IN      NS      ns2.example.com.
test    3600    IN      CNAME   old.example.com.
`)

	a := &API{ZoneFile: zoneFile}
	if err := a.addRecordToZone("example.com", "test", "CNAME", "new.example.com.", 3600); err != nil {
		t.Fatalf("addRecordToZone failed: %v", err)
	}

	content := readZoneFile(t, zoneFile)
	if got := strings.Count(content, "test\t3600\tIN\tCNAME\t"); got != 1 {
		t.Fatalf("expected exactly one test CNAME record, got %d\n%s", got, content)
	}
	if !strings.Contains(content, "test\t3600\tIN\tCNAME\tnew.example.com.") {
		t.Fatalf("expected updated CNAME target in zone file\n%s", content)
	}
	if strings.Contains(content, "old.example.com.") {
		t.Fatalf("old CNAME target should be removed\n%s", content)
	}
	if strings.Contains(content, "@       IN      NS      ns2.example.com.\n\ntest\t3600\tIN\tCNAME\tnew.example.com.") {
		t.Fatalf("replacement should not leave an empty line before the new record\n%s", content)
	}
}

func TestDeleteRecordFromZoneOnlyDeletesCurrentZone(t *testing.T) {
	t.Parallel()

	zoneFile := writeTempZoneFile(t, `
$ORIGIN example.com.
$TTL 3600

@       IN      SOA     ns1.example.com. admin.example.com. (
                        2024010101 ; serial
                        3600       ; refresh
                        1800       ; retry
                        604800     ; expire
                        86400 )    ; minimum

@       IN      NS      ns1.example.com.
@       IN      NS      ns2.example.com.
test    3600    IN      CNAME   example-target.example.com.

$ORIGIN other.com.
$TTL 3600

@       IN      SOA     ns1.other.com. admin.other.com. (
                        2024010101 ; serial
                        3600       ; refresh
                        1800       ; retry
                        604800     ; expire
                        86400 )    ; minimum

@       IN      NS      ns1.other.com.
@       IN      NS      ns2.other.com.
test    3600    IN      CNAME   other-target.other.com.
`)

	a := &API{ZoneFile: zoneFile}
	if err := a.deleteRecordFromZone("example.com", "test", "CNAME"); err != nil {
		t.Fatalf("deleteRecordFromZone failed: %v", err)
	}

	content := readZoneFile(t, zoneFile)
	if strings.Contains(content, "example-target.example.com.") {
		t.Fatalf("record in example.com should be removed\n%s", content)
	}
	if !strings.Contains(content, "other-target.other.com.") {
		t.Fatalf("record in other.com should remain\n%s", content)
	}
}

func writeTempZoneFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	filename := filepath.Join(dir, "zones.db")
	if err := os.WriteFile(filename, []byte(strings.TrimSpace(content)+"\n"), 0644); err != nil {
		t.Fatalf("write temp zone file failed: %v", err)
	}
	return filename
}

func readZoneFile(t *testing.T, filename string) string {
	t.Helper()

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read zone file failed: %v", err)
	}
	return string(content)
}

func TestECSSubnetFromMsg(t *testing.T) {
	t.Parallel()

	msg := new(dns.Msg)
	msg.SetQuestion("edge.example.", dns.TypeA)
	msg.SetEdns0(1232, false)
	opt := msg.IsEdns0()
	opt.Option = append(opt.Option, &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        1,
		SourceNetmask: 24,
		SourceScope:   16,
		Address:       net.ParseIP("61.142.56.193").To4(),
	})

	subnet := ecsSubnetFromMsg(msg)
	if subnet == nil {
		t.Fatalf("expected ECS subnet to be extracted")
	}
	if subnet.Family != 1 || subnet.SourceNetmask != 24 || subnet.SourceScope != 16 {
		t.Fatalf("unexpected ECS fields: family=%d source=%d scope=%d", subnet.Family, subnet.SourceNetmask, subnet.SourceScope)
	}
}

func TestECSDetailsIncludesExpectedFields(t *testing.T) {
	t.Parallel()

	subnet := &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        1,
		SourceNetmask: 32,
		SourceScope:   24,
		Address:       net.ParseIP("61.142.56.193").To4(),
	}

	details := ecsDetails(subnet)
	expectedParts := []string{
		"type=41",
		"option_code=8",
		"option_length=8",
		"family=1",
		"source_prefix=32",
		"scope_prefix=24",
		"address=61.142.56.193",
	}
	for _, part := range expectedParts {
		if !strings.Contains(details, part) {
			t.Fatalf("details should contain %q, got: %s", part, details)
		}
	}
}

func TestECSOptionLengthClamp(t *testing.T) {
	t.Parallel()

	ipv4TooLong := &dns.EDNS0_SUBNET{
		Family:        1,
		SourceNetmask: 40,
		Address:       net.ParseIP("61.142.56.193").To4(),
	}
	if got := ecsOptionLength(ipv4TooLong); got != 8 {
		t.Fatalf("expected ipv4 option length 8, got %d", got)
	}

	ipv6 := &dns.EDNS0_SUBNET{
		Family:        2,
		SourceNetmask: 56,
		Address:       net.ParseIP("2001:db8::1"),
	}
	if got := ecsOptionLength(ipv6); got != 11 {
		t.Fatalf("expected ipv6 option length 11, got %d", got)
	}
}

func TestParseBoolOption(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value string
		want  bool
		ok    bool
	}{
		{value: "on", want: true, ok: true},
		{value: "off", want: false, ok: true},
		{value: "true", want: true, ok: true},
		{value: "false", want: false, ok: true},
		{value: "1", want: true, ok: true},
		{value: "0", want: false, ok: true},
		{value: "YES", want: true, ok: true},
		{value: "No", want: false, ok: true},
		{value: "maybe", ok: false},
	}

	for _, tc := range cases {
		got, err := parseBoolOption(tc.value)
		if tc.ok && err != nil {
			t.Fatalf("value %q should parse, got err=%v", tc.value, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("value %q should fail parsing", tc.value)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("value %q parsed as %v, want %v", tc.value, got, tc.want)
		}
	}
}
/*
test_esc_log:
dig @127.0.0.1 -p 8053 www.example.com A +subnet=61.142.56.193/32 +noall +answer
*/
