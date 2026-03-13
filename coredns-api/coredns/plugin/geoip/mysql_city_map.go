package geoip

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type goedgeCityMySQLStore struct {
	db              *sql.DB
	query           string
	refreshInterval time.Duration
	queryTimeout    time.Duration

	mu      sync.RWMutex
	records []goedgeCityMySQLRecord
}

type goedgeCityMySQLRecord struct {
	prefix      netip.Prefix
	SourceGroup string
	RegionID    int64
	RegionName  string

	CountryID    int64
	CountryName  string
	ProvinceID   int64
	ProvinceName string
	CityID       int64
	CityName     string
	ProviderID   int64
	ProviderName string
	Summary      string
}

func newGoEdgeCityMySQLStore(cfg GoEdgeCityMySQLConfig) (*goedgeCityMySQLStore, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, nil
	}

	refreshInterval := cfg.RefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = DefaultGoEdgeCityMySQLRefresh
	}

	queryTimeout := cfg.QueryTimeout
	if queryTimeout <= 0 {
		queryTimeout = DefaultGoEdgeCityMySQLQueryTimeout
	}

	query := strings.TrimSpace(cfg.Query)
	if query == "" {
		tableName := strings.TrimSpace(cfg.Table)
		if tableName == "" {
			tableName = DefaultGoEdgeCityMySQLTable
		}

		quoted, err := quoteMySQLIdentifierPath(tableName)
		if err != nil {
			return nil, err
		}
		query = "SELECT * FROM " + quoted
	}

	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	db.SetMaxIdleConns(2)
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), queryTimeout)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	store := &goedgeCityMySQLStore{
		db:              db,
		query:           query,
		refreshInterval: refreshInterval,
		queryTimeout:    queryTimeout,
	}

	if err := store.reload(); err != nil {
		_ = db.Close()
		return nil, err
	}

	go store.refreshLoop()
	return store, nil
}

func (s *goedgeCityMySQLStore) refreshLoop() {
	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.reload(); err != nil {
			log.Warningf("goedge-city mysql mapping reload failed: %v", err)
		}
	}
}

func (s *goedgeCityMySQLStore) reload() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.queryTimeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, s.query)
	if err != nil {
		return fmt.Errorf("query mysql mappings failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("read mysql columns failed: %w", err)
	}

	columnKeys := make([]string, len(columns))
	for i, col := range columns {
		columnKeys[i] = strings.ToLower(strings.TrimSpace(col))
	}

	records := make([]goedgeCityMySQLRecord, 0, 64)
	values := make([]sql.NullString, len(columnKeys))
	scanArgs := make([]any, len(columnKeys))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		for i := range values {
			values[i] = sql.NullString{}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return fmt.Errorf("scan mysql mapping row failed: %w", err)
		}

		row := make(map[string]string, len(columnKeys))
		for i, key := range columnKeys {
			if values[i].Valid {
				row[key] = strings.TrimSpace(values[i].String)
			}
		}

		record, ok, err := parseGoedgeCityMySQLRow(row)
		if err != nil {
			log.Warningf("skip mysql mapping row due to parse error: %v", err)
			continue
		}
		if !ok {
			continue
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate mysql mapping rows failed: %w", err)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].prefix.Bits() > records[j].prefix.Bits()
	})

	s.mu.Lock()
	s.records = records
	s.mu.Unlock()
	return nil
}

func (s *goedgeCityMySQLStore) Lookup(clientIP netip.Addr, source, fallbackReason string) (GoEdgeCityMappingResult, bool) {
	ip := clientIP.Unmap()

	s.mu.RLock()
	records := s.records
	s.mu.RUnlock()

	for _, record := range records {
		if !record.prefix.Contains(ip) {
			continue
		}

		return GoEdgeCityMappingResult{
			EffectiveClientIP: clientIP.String(),
			ClientIPSource:    source,
			FallbackReason:    fallbackReason,
			SourceGroup:       record.SourceGroup,
			RegionID:          record.RegionID,
			RegionName:        record.RegionName,
			CountryID:         record.CountryID,
			CountryName:       record.CountryName,
			ProvinceID:        record.ProvinceID,
			ProvinceName:      record.ProvinceName,
			CityID:            record.CityID,
			CityName:          record.CityName,
			ProviderID:        record.ProviderID,
			ProviderName:      record.ProviderName,
			Summary:           record.Summary,
			Hit:               true,
		}, true
	}

	return GoEdgeCityMappingResult{}, false
}

func parseGoedgeCityMySQLRow(row map[string]string) (goedgeCityMySQLRecord, bool, error) {
	var record goedgeCityMySQLRecord

	if enabledText, ok := rowValue(row, "enabled", "is_enabled", "status"); ok {
		enabled, err := parseOptionalBool(enabledText)
		if err == nil && !enabled {
			return record, false, nil
		}
	}

	subnetText, ok := rowValue(row,
		"ecs_subnet",
		"mapping_subnet",
		"mapping_cidr",
		"subnet",
		"prefix",
		"cidr",
		"ecs_ip",
		"mapping_ip",
		"ip",
	)
	if !ok {
		return record, false, nil
	}

	prefix, err := parseGoedgeCityPrefix(subnetText)
	if err != nil {
		return record, false, fmt.Errorf("invalid subnet %q: %w", subnetText, err)
	}
	record.prefix = prefix.Masked()

	sourceGroup, _ := rowValue(row, "source_group", "edns_source_group", "edns_user_source_group", "group_name")
	record.SourceGroup = sourceGroup

	record.CountryID = parseOptionalInt64(row, "country_id")
	record.CountryName, _ = rowValue(row, "country_name", "country")
	record.RegionID = parseOptionalInt64(row, "region_id", "regionid", "node_region_id")
	record.RegionName, _ = rowValue(row, "region_name", "region", "region_class")
	record.ProvinceID = parseOptionalInt64(row, "province_id")
	record.ProvinceName, _ = rowValue(row, "province_name", "province")
	record.CityID = parseOptionalInt64(row, "city_id")
	record.CityName, _ = rowValue(row, "city_name", "city")
	record.ProviderID = parseOptionalInt64(row, "provider_id")
	record.ProviderName, _ = rowValue(row, "provider_name", "provider", "isp_name", "operator_name")

	if record.CityName == "" {
		record.CityName = sourceGroup
	}
	if record.ProviderName == "" {
		record.ProviderName = sourceGroup
	}

	record.Summary, _ = rowValue(row, "summary")
	if record.Summary == "" {
		record.Summary = buildGoEdgeSummary(record.CountryName, record.ProvinceName, record.CityName)
	}

	if record.CityID <= 0 && strings.TrimSpace(record.CityName) == "" {
		return record, false, nil
	}

	return record, true, nil
}

func parseGoedgeCityPrefix(raw string) (netip.Prefix, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return netip.Prefix{}, fmt.Errorf("empty subnet")
	}

	if strings.Contains(trimmed, "/") {
		prefix, err := netip.ParsePrefix(trimmed)
		if err != nil {
			return netip.Prefix{}, err
		}
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(trimmed)
	if err != nil {
		return netip.Prefix{}, err
	}
	addr = addr.Unmap()

	bits := 128
	if addr.Is4() {
		bits = 32
	}
	return netip.PrefixFrom(addr, bits).Masked(), nil
}

func rowValue(row map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		return value, true
	}
	return "", false
}

func parseOptionalInt64(row map[string]string, keys ...string) int64 {
	value, ok := rowValue(row, keys...)
	if !ok {
		return 0
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseOptionalBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "on", "yes", "enabled":
		return true, nil
	case "0", "f", "false", "off", "no", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value %q", value)
	}
}

var mysqlIdentifierRe = regexp.MustCompile(`^[0-9A-Za-z_]+$`)

func quoteMySQLIdentifierPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("mysql table name is empty")
	}

	parts := strings.Split(path, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !mysqlIdentifierRe.MatchString(part) {
			return "", fmt.Errorf("invalid mysql table identifier %q", part)
		}
		quoted = append(quoted, "`"+part+"`")
	}
	return strings.Join(quoted, "."), nil
}
