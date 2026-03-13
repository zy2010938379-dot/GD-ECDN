package geoip

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
)

var (
	fixturesDir   = "./testdata"
	cityDBPath    = filepath.Join(fixturesDir, "GeoLite2-City.mmdb")
	asnDBPath     = filepath.Join(fixturesDir, "GeoLite2-ASN.mmdb")
	unknownDBPath = filepath.Join(fixturesDir, "GeoLite2-UnknownDbType.mmdb")
)

func TestProbingIP(t *testing.T) {
	if !probingIP.IsValid() {
		t.Fatalf("Invalid probing IP: %q", probingIP)
	}
}

func TestSetup(t *testing.T) {
	c := caddy.NewTestController("dns", fmt.Sprintf("%s %s", pluginName, cityDBPath))
	plugins := dnsserver.GetConfig(c).Plugin
	if len(plugins) != 0 {
		t.Fatalf("Expected zero plugins after setup, %d found", len(plugins))
	}
	if err := setup(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}
	plugins = dnsserver.GetConfig(c).Plugin
	if len(plugins) != 1 {
		t.Fatalf("Expected one plugin after setup, %d found", len(plugins))
	}
}

func TestGeoIPParse(t *testing.T) {
	c := caddy.NewTestController("dns", fmt.Sprintf("%s %s", pluginName, cityDBPath))
	if err := setup(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	tests := []struct {
		shouldErr      bool
		config         string
		expectedErr    string
		expectedDBType int
		expectDB       bool
	}{
		// Valid - City database
		{false, fmt.Sprintf("%s %s\n", pluginName, cityDBPath), "", city, true},
		{false, fmt.Sprintf("%s %s {\n\tedns-subnet\n}", pluginName, cityDBPath), "", city, true},
		{false, fmt.Sprintf("%s %s {\n\tedns-subnet\n\tecs-fallback disabled\n}", pluginName, cityDBPath), "", city, true},
		// Valid - ASN database
		{false, fmt.Sprintf("%s %s\n", pluginName, asnDBPath), "", asn, true},
		{false, fmt.Sprintf("%s %s {\n\tedns-subnet\n}", pluginName, asnDBPath), "", asn, true},
		// Valid - GoEdge city library only
		{false, fmt.Sprintf("%s {\n\tgoedge-city\n}\n", pluginName), "", 0, false},

		// Invalid
		{true, pluginName, "Wrong argument count", 0, false},
		{true, fmt.Sprintf("%s %s {\n\tlanguages en fr es zh-CN\n}\n", pluginName, cityDBPath), "unknown property \"languages\"", 0, false},
		{true, fmt.Sprintf("%s %s\n%s %s\n", pluginName, cityDBPath, pluginName, cityDBPath), "configuring multiple databases is not supported", 0, false},
		{true, fmt.Sprintf("%s 1 2 3", pluginName), "Wrong argument count", 0, false},
		{true, fmt.Sprintf("%s { }", pluginName), "Wrong argument count", 0, false},
		{true, fmt.Sprintf("%s /dbpath { city }", pluginName), "unknown property \"city\"", 0, false},
		{true, fmt.Sprintf("%s /invalidPath\n", pluginName), "failed to open database file: open /invalidPath: no such file or directory", 0, false},
		{true, fmt.Sprintf("%s %s\n", pluginName, unknownDBPath), "reader does not support the \"UnknownDbType\" database type", 0, false},
		{true, fmt.Sprintf("%s %s {\n\tecs-fallback nope\n}", pluginName, cityDBPath), "unknown ecs-fallback policy", 0, false},
		{true, fmt.Sprintf("%s {\n\tgoedge-city-mysql-table city_map\n\tgoedge-city\n}\n", pluginName), "goedge-city-mysql-dsn is required", 0, false},
		{true, fmt.Sprintf("%s %s {\n\tgoedge-city-mysql-dsn user:pass@tcp(127.0.0.1:3306)/geo\n}\n", pluginName, cityDBPath), "requires goedge-city", 0, false},
		{true, fmt.Sprintf("%s {\n\tgoedge-city\n\tgoedge-city-mysql-refresh abc\n}\n", pluginName), "invalid goedge-city-mysql-refresh value", 0, false},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.config)
		geoIP, err := geoipParse(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: expected error but found none for input %s", i, test.config)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: expected no error but found one for input %s, got: %v", i, test.config, err)
			}

			if !strings.Contains(err.Error(), test.expectedErr) {
				t.Errorf("Test %d: expected error to contain: %v, found error: %v, input: %s", i, test.expectedErr, err, test.config)
			}
			continue
		}

		if test.expectDB && geoIP.db.Reader == nil {
			t.Errorf("Test %d: expected database reader to be initialized", i)
		}
		if !test.expectDB && geoIP.db.Reader != nil {
			t.Errorf("Test %d: expected database reader to be nil", i)
		}

		if test.expectedDBType > 0 && geoIP.db.provides&test.expectedDBType == 0 {
			t.Errorf("Test %d: expected db type %d not found, database file provides %d", i, test.expectedDBType, geoIP.db.provides)
		}
	}
}
