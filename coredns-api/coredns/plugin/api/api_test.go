package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
