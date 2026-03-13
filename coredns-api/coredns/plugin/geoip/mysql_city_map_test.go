package geoip

import "testing"

func TestParseGoedgeCityPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{input: "61.142.56.193/32", want: "61.142.56.193/32"},
		{input: "61.142.56.193", want: "61.142.56.193/32"},
		{input: "2001:db8::/56", want: "2001:db8::/56"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseGoedgeCityPrefix(tc.input)
			if err != nil {
				t.Fatalf("parseGoedgeCityPrefix(%q) failed: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("parseGoedgeCityPrefix(%q)=%q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestParseGoedgeCityMySQLRow(t *testing.T) {
	t.Parallel()

	row := map[string]string{
		"ecs_subnet":   "61.142.56.193/32",
		"source_group": "广州电信",
		"region_id":    "8",
		"region_name":  "广州区域",
		"city_id":      "440100",
	}

	record, ok, err := parseGoedgeCityMySQLRow(row)
	if err != nil {
		t.Fatalf("parseGoedgeCityMySQLRow failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected mysql row to be accepted")
	}
	if record.prefix.String() != "61.142.56.193/32" {
		t.Fatalf("unexpected prefix: %s", record.prefix.String())
	}
	if record.CityID != 440100 {
		t.Fatalf("unexpected city id: %d", record.CityID)
	}
	if record.RegionID != 8 {
		t.Fatalf("unexpected region id: %d", record.RegionID)
	}
	if record.RegionName != "广州区域" {
		t.Fatalf("unexpected region name: %q", record.RegionName)
	}
	if record.CityName != "广州电信" {
		t.Fatalf("unexpected city name: %q", record.CityName)
	}
	if record.ProviderName != "广州电信" {
		t.Fatalf("unexpected provider name: %q", record.ProviderName)
	}
}

func TestParseGoedgeCityMySQLRowDisabled(t *testing.T) {
	t.Parallel()

	row := map[string]string{
		"ecs_subnet": "61.142.56.193/32",
		"city_name":  "广州",
		"enabled":    "0",
	}

	_, ok, err := parseGoedgeCityMySQLRow(row)
	if err != nil {
		t.Fatalf("parseGoedgeCityMySQLRow failed: %v", err)
	}
	if ok {
		t.Fatalf("expected disabled mysql row to be skipped")
	}
}

func TestQuoteMySQLIdentifierPath(t *testing.T) {
	t.Parallel()

	got, err := quoteMySQLIdentifierPath("geo.city_map")
	if err != nil {
		t.Fatalf("quoteMySQLIdentifierPath failed: %v", err)
	}
	if got != "`geo`.`city_map`" {
		t.Fatalf("unexpected quoted path: %s", got)
	}

	if _, err := quoteMySQLIdentifierPath("geo.city-map"); err == nil {
		t.Fatalf("expected invalid identifier to fail")
	}
}
