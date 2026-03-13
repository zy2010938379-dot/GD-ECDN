package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/coredns/coredns/plugin/metadata"
	pkgrand "github.com/coredns/coredns/plugin/pkg/rand"
	"github.com/miekg/dns"
)

const (
	defaultGoEdgeCityNodeRoutingRefresh = 30 * time.Second
	defaultGoEdgeCityNodeRoutingTimeout = 3 * time.Second
	defaultGoEdgeCityNodeRoutingTTL     = uint32(60)
	defaultGoEdgeCityNodeRoutingRole    = "node"
)

// GoEdgeCityNodeRoutingConfig controls selecting node IPs from a GoEdge node cluster.
type GoEdgeCityNodeRoutingConfig struct {
	Enabled         bool          `json:"enabled,omitempty"`
	MySQLDSN        string        `json:"mysql_dsn,omitempty"`
	ClusterID       int64         `json:"cluster_id,omitempty"`
	RefreshInterval time.Duration `json:"refresh_interval,omitempty"`
	QueryTimeout    time.Duration `json:"query_timeout,omitempty"`
	TTL             uint32        `json:"ttl,omitempty"`
	FQDN            string        `json:"fqdn,omitempty"`
	NodeRole        string        `json:"node_role,omitempty"`
}

func (c GoEdgeCityNodeRoutingConfig) withDefaults() GoEdgeCityNodeRoutingConfig {
	cfg := c
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = defaultGoEdgeCityNodeRoutingRefresh
	}
	if cfg.QueryTimeout <= 0 {
		cfg.QueryTimeout = defaultGoEdgeCityNodeRoutingTimeout
	}
	if cfg.TTL == 0 {
		cfg.TTL = defaultGoEdgeCityNodeRoutingTTL
	}
	if strings.TrimSpace(cfg.NodeRole) == "" {
		cfg.NodeRole = defaultGoEdgeCityNodeRoutingRole
	}
	cfg.FQDN = normalizeFQDN(cfg.FQDN)
	return cfg
}

type goedgeCityNodeRouter struct {
	cfg  GoEdgeCityNodeRoutingConfig
	db   *sql.DB
	rand *pkgrand.Rand

	mu       sync.RWMutex
	snapshot goedgeCityNodeSnapshot
}

type goedgeCityNodeSnapshot struct {
	fqdn string

	regionIPv4 map[int64][]string
	regionIPv6 map[int64][]string
	allIPv4    []string
	allIPv6    []string
}

func newGoedgeCityNodeRouter(cfg GoEdgeCityNodeRoutingConfig) (*goedgeCityNodeRouter, error) {
	cfg = cfg.withDefaults()
	if strings.TrimSpace(cfg.MySQLDSN) == "" {
		return nil, fmt.Errorf("goedge-city-node-routing-mysql-dsn is required")
	}
	if cfg.ClusterID <= 0 {
		return nil, fmt.Errorf("goedge-city-node-routing-cluster-id is required")
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql for city node routing: %w", err)
	}
	db.SetMaxIdleConns(2)
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(context.Background(), cfg.QueryTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping mysql for city node routing: %w", err)
	}

	router := &goedgeCityNodeRouter{
		cfg:  cfg,
		db:   db,
		rand: pkgrand.New(time.Now().UnixNano()),
	}

	if err := router.reload(); err != nil {
		_ = db.Close()
		return nil, err
	}

	go router.refreshLoop()
	return router, nil
}

func (r *goedgeCityNodeRouter) refreshLoop() {
	ticker := time.NewTicker(r.cfg.RefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := r.reload(); err != nil {
			log.Printf("[api] goedge city node routing reload failed: %v", err)
		}
	}
}

func (r *goedgeCityNodeRouter) reload() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.QueryTimeout)
	defer cancel()

	snapshot, err := r.loadSnapshot(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.snapshot = snapshot
	r.mu.Unlock()
	return nil
}

func (r *goedgeCityNodeRouter) loadSnapshot(ctx context.Context) (goedgeCityNodeSnapshot, error) {
	var (
		clusterDNSName sql.NullString
		domainName     sql.NullString
	)

	err := r.db.QueryRowContext(ctx, `
SELECT c.dnsName, IFNULL(d.name, '')
FROM edgeNodeClusters c
LEFT JOIN edgeDNSDomains d ON d.id = c.dnsDomainId AND d.state = 1
WHERE c.id = ? AND c.state = 1 AND c.isOn = 1
LIMIT 1
`, r.cfg.ClusterID).Scan(&clusterDNSName, &domainName)
	if err != nil {
		if err == sql.ErrNoRows {
			return goedgeCityNodeSnapshot{}, fmt.Errorf("cluster %d not found or not enabled", r.cfg.ClusterID)
		}
		return goedgeCityNodeSnapshot{}, fmt.Errorf("query cluster dns info failed: %w", err)
	}

	fqdn := r.cfg.FQDN
	if fqdn == "" {
		dnsName := strings.TrimSpace(clusterDNSName.String)
		domain := strings.TrimSpace(domainName.String)
		if dnsName != "" && domain != "" {
			fqdn = normalizeFQDN(dnsName + "." + domain)
		}
	}
	if fqdn == "" {
		return goedgeCityNodeSnapshot{}, fmt.Errorf("unable to infer cluster fqdn, set goedge-city-node-routing-fqdn explicitly")
	}

	nodeRegions, nodeIDs, err := r.loadNodeRegions(ctx)
	if err != nil {
		return goedgeCityNodeSnapshot{}, err
	}

	regionIPv4Sets := map[int64]map[string]struct{}{}
	regionIPv6Sets := map[int64]map[string]struct{}{}
	allIPv4Set := map[string]struct{}{}
	allIPv6Set := map[string]struct{}{}

	if len(nodeIDs) > 0 {
		ipRows, err := r.loadNodeIPs(ctx, nodeIDs)
		if err != nil {
			return goedgeCityNodeSnapshot{}, err
		}

		for _, row := range ipRows {
			addr, ok := parseAddr(row.ip)
			if !ok {
				continue
			}

			regionID, ok := nodeRegions[row.nodeID]
			if !ok {
				continue
			}

			ipText := addr.String()
			if addr.Is4() {
				allIPv4Set[ipText] = struct{}{}
				if regionID > 0 {
					addRegionIP(regionIPv4Sets, regionID, ipText)
				}
			} else {
				allIPv6Set[ipText] = struct{}{}
				if regionID > 0 {
					addRegionIP(regionIPv6Sets, regionID, ipText)
				}
			}
		}
	}

	return goedgeCityNodeSnapshot{
		fqdn:       fqdn,
		regionIPv4: regionSetToSlice(regionIPv4Sets),
		regionIPv6: regionSetToSlice(regionIPv6Sets),
		allIPv4:    setToSortedSlice(allIPv4Set),
		allIPv6:    setToSortedSlice(allIPv6Set),
	}, nil
}

type nodeIPRow struct {
	nodeID int64
	ip     string
}

func (r *goedgeCityNodeRouter) loadNodeRegions(ctx context.Context) (map[int64]int64, []int64, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, IFNULL(regionId, 0)
FROM edgeNodes
WHERE state = 1
  AND isOn = 1
  AND isUp = 1
  AND isInstalled = 1
  AND clusterId = ?
`, r.cfg.ClusterID)
	if err != nil {
		return nil, nil, fmt.Errorf("query edge nodes failed: %w", err)
	}
	defer rows.Close()

	nodeRegions := map[int64]int64{}
	nodeIDs := make([]int64, 0, 32)
	for rows.Next() {
		var nodeID int64
		var regionID int64
		if err := rows.Scan(&nodeID, &regionID); err != nil {
			return nil, nil, fmt.Errorf("scan edge node failed: %w", err)
		}
		nodeIDs = append(nodeIDs, nodeID)
		nodeRegions[nodeID] = regionID
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate edge nodes failed: %w", err)
	}
	return nodeRegions, nodeIDs, nil
}

func (r *goedgeCityNodeRouter) loadNodeIPs(ctx context.Context, nodeIDs []int64) ([]nodeIPRow, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, 0, len(nodeIDs))
	args := make([]any, 0, len(nodeIDs)+1)
	args = append(args, r.cfg.NodeRole)
	for _, id := range nodeIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := `
SELECT nodeId, ip
FROM edgeNodeIPAddresses
WHERE state = 1
  AND role = ?
  AND isOn = 1
  AND isUp = 1
  AND canAccess = 1
  AND ip IS NOT NULL
  AND ip != ''
  AND nodeId IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query node ips failed: %w", err)
	}
	defer rows.Close()

	result := make([]nodeIPRow, 0, 64)
	for rows.Next() {
		var row nodeIPRow
		if err := rows.Scan(&row.nodeID, &row.ip); err != nil {
			return nil, fmt.Errorf("scan node ip failed: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node ips failed: %w", err)
	}
	return result, nil
}

func addRegionIP(regionMap map[int64]map[string]struct{}, regionID int64, ip string) {
	bucket, ok := regionMap[regionID]
	if !ok {
		bucket = map[string]struct{}{}
		regionMap[regionID] = bucket
	}
	bucket[ip] = struct{}{}
}

func regionSetToSlice(regionMap map[int64]map[string]struct{}) map[int64][]string {
	result := make(map[int64][]string, len(regionMap))
	for regionID, ipSet := range regionMap {
		result[regionID] = setToSortedSlice(ipSet)
	}
	return result
}

func setToSortedSlice(set map[string]struct{}) []string {
	result := make([]string, 0, len(set))
	for item := range set {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func parseAddr(ip string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func normalizeFQDN(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return ""
	}
	return name + "."
}

func (r *goedgeCityNodeRouter) selectNodeIP(qname string, qtype uint16, regionID int64) (string, bool) {
	r.mu.RLock()
	snapshot := r.snapshot
	r.mu.RUnlock()

	if snapshot.fqdn == "" || normalizeFQDN(qname) != snapshot.fqdn {
		return "", false
	}

	var (
		allPool    []string
		regionPool map[int64][]string
	)
	switch qtype {
	case dns.TypeA:
		allPool = snapshot.allIPv4
		regionPool = snapshot.regionIPv4
	case dns.TypeAAAA:
		allPool = snapshot.allIPv6
		regionPool = snapshot.regionIPv6
	default:
		return "", false
	}

	if regionID > 0 {
		if scoped := regionPool[regionID]; len(scoped) > 0 {
			return scoped[r.rand.Int()%len(scoped)], true
		}
	}

	if len(allPool) == 0 {
		return "", false
	}
	return allPool[r.rand.Int()%len(allPool)], true
}

func metadataValue(ctx context.Context, key string) string {
	valueFunc := metadata.ValueFunc(ctx, key)
	if valueFunc == nil {
		return ""
	}
	return strings.TrimSpace(valueFunc())
}

func parseMetadataInt64(ctx context.Context, key string) int64 {
	value := metadataValue(ctx, key)
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (a *API) serveCityNodeRoute(ctx context.Context, w dns.ResponseWriter, req *dns.Msg) (bool, int, error) {
	if a.cityNodeRouter == nil || req == nil || len(req.Question) == 0 {
		return false, 0, nil
	}

	question := req.Question[0]
	if question.Qclass != dns.ClassINET {
		return false, 0, nil
	}
	if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
		return false, 0, nil
	}

	regionID := parseMetadataInt64(ctx, "geoip/goedge/region/id")
	ipText, ok := a.cityNodeRouter.selectNodeIP(question.Name, question.Qtype, regionID)
	if !ok {
		return false, 0, nil
	}

	ip := net.ParseIP(ipText)
	if ip == nil {
		return false, 0, nil
	}

	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true

	hdr := dns.RR_Header{
		Name:   question.Name,
		Rrtype: question.Qtype,
		Class:  dns.ClassINET,
		Ttl:    a.cityNodeRouter.cfg.TTL,
	}

	switch question.Qtype {
	case dns.TypeA:
		ip4 := ip.To4()
		if ip4 == nil {
			return false, 0, nil
		}
		resp.Answer = append(resp.Answer, &dns.A{Hdr: hdr, A: ip4})
	case dns.TypeAAAA:
		if ip.To4() != nil {
			return false, 0, nil
		}
		ip6 := ip.To16()
		if ip6 == nil {
			return false, 0, nil
		}
		resp.Answer = append(resp.Answer, &dns.AAAA{Hdr: hdr, AAAA: ip6})
	default:
		return false, 0, nil
	}

	if err := w.WriteMsg(resp); err != nil {
		return true, dns.RcodeServerFailure, err
	}
	return true, dns.RcodeSuccess, nil
}
