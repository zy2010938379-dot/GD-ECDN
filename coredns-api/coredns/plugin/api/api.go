package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

func init() { plugin.Register("api", setup) }

// API represents the API plugin instance
type API struct {
	Next plugin.Handler

	// Configuration
	Address         string                      `json:"address,omitempty"`
	APIKey          string                      `json:"apikey,omitempty"`
	ZoneFile        string                      `json:"zone_file,omitempty"`
	ECSLog          bool                        `json:"ecs_log,omitempty"`
	CityNodeRouting GoEdgeCityNodeRoutingConfig `json:"city_node_routing,omitempty"`

	// Internal state
	mu             sync.RWMutex
	httpServer     *http.Server
	cityNodeRouter *goedgeCityNodeRouter
}

type ecsLoggingResponseWriter struct {
	dns.ResponseWriter
	req        *dns.Msg
	remoteAddr net.Addr
}

func (w *ecsLoggingResponseWriter) WriteMsg(res *dns.Msg) error {
	logECSResponse(w.req, res, w.remoteAddr)
	return w.ResponseWriter.WriteMsg(res)
}

func (w *ecsLoggingResponseWriter) Write(buf []byte) (int, error) {
	if len(buf) > 0 {
		msg := new(dns.Msg)
		if err := msg.Unpack(buf); err == nil {
			logECSResponse(w.req, msg, w.remoteAddr)
		}
	}
	return w.ResponseWriter.Write(buf)
}

// ServeDNS implements the plugin.Handler interface
func (a *API) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	writer := w
	if a.ECSLog {
		remoteAddr := w.RemoteAddr()
		logECSRequest(r, remoteAddr)
		writer = &ecsLoggingResponseWriter{
			ResponseWriter: w,
			req:            r,
			remoteAddr:     remoteAddr,
		}
	}

	if handled, rcode, err := a.serveCityNodeRoute(ctx, writer, r); handled {
		return rcode, err
	}

	return plugin.NextOrFailure(a.Name(), a.Next, ctx, writer, r)
}

// Name implements the plugin.Handler interface
func (a *API) Name() string { return "api" }

func logECSRequest(msg *dns.Msg, remoteAddr net.Addr) {
	subnet := ecsSubnetFromMsg(msg)
	if subnet == nil {
		return
	}

	log.Printf("[api] ecs request qname=%s resolver=%s %s", questionName(msg), remoteAddrString(remoteAddr), ecsDetails(subnet))
}

func logECSResponse(req, res *dns.Msg, remoteAddr net.Addr) {
	reqSubnet := ecsSubnetFromMsg(req)
	resSubnet := ecsSubnetFromMsg(res)
	if reqSubnet == nil && resSubnet == nil {
		return
	}

	rcode := dns.RcodeSuccess
	if res != nil {
		rcode = res.Rcode
	}

	if resSubnet == nil {
		log.Printf("[api] ecs response qname=%s resolver=%s rcode=%s option=absent", questionName(req), remoteAddrString(remoteAddr), rcodeToString(rcode))
		return
	}

	log.Printf("[api] ecs response qname=%s resolver=%s rcode=%s %s", questionName(req), remoteAddrString(remoteAddr), rcodeToString(rcode), ecsDetails(resSubnet))
}

func ecsSubnetFromMsg(msg *dns.Msg) *dns.EDNS0_SUBNET {
	if msg == nil {
		return nil
	}

	opt := msg.IsEdns0()
	if opt == nil {
		return nil
	}

	for _, option := range opt.Option {
		if subnet, ok := option.(*dns.EDNS0_SUBNET); ok {
			return subnet
		}
	}

	return nil
}

func ecsDetails(subnet *dns.EDNS0_SUBNET) string {
	if subnet == nil {
		return ""
	}

	address := "<nil>"
	if subnet.Address != nil {
		address = subnet.Address.String()
	}

	return fmt.Sprintf(
		"type=%d option_code=%d option_length=%d family=%d source_prefix=%d scope_prefix=%d address=%s",
		dns.TypeOPT,
		subnet.Option(),
		ecsOptionLength(subnet),
		subnet.Family,
		subnet.SourceNetmask,
		subnet.SourceScope,
		address,
	)
}

func ecsOptionLength(subnet *dns.EDNS0_SUBNET) int {
	if subnet == nil {
		return 0
	}

	maxPrefixBits := 128
	switch subnet.Family {
	case 1:
		maxPrefixBits = 32
	case 2:
		maxPrefixBits = 128
	default:
		if subnet.Address != nil && subnet.Address.To4() != nil {
			maxPrefixBits = 32
		}
	}

	prefixBits := int(subnet.SourceNetmask)
	if prefixBits > maxPrefixBits {
		prefixBits = maxPrefixBits
	}
	if prefixBits < 0 {
		prefixBits = 0
	}

	// RFC 7871 section 6: FAMILY(2) + SOURCE PREFIX-LENGTH(1) + SCOPE PREFIX-LENGTH(1) + ADDRESS(n)
	return 4 + (prefixBits+7)/8
}

func questionName(msg *dns.Msg) string {
	if msg == nil || len(msg.Question) == 0 {
		return "."
	}
	return msg.Question[0].Name
}

func remoteAddrString(addr net.Addr) string {
	if addr == nil {
		return "unknown"
	}
	return addr.String()
}

func rcodeToString(rcode int) string {
	if s, ok := dns.RcodeToString[rcode]; ok {
		return s
	}
	return strconv.Itoa(rcode)
}

func parseBoolOption(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "on", "yes":
		return true, nil
	case "0", "f", "false", "off", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

// startHTTPServer starts the HTTP API server
func (a *API) startHTTPServer() error {
	mux := http.NewServeMux()

	// Register API endpoints (compatible with GoEdge expected endpoints)
	mux.HandleFunc("/zones", a.authMiddleware(a.handleZones))
	mux.HandleFunc("/zones/", a.authMiddleware(a.handleZoneRecords))

	a.httpServer = &http.Server{
		Addr:    a.Address,
		Handler: mux,
	}

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	log.Printf("API server started on %s", a.Address)
	return nil
}

// authMiddleware adds authentication to API endpoints
func (a *API) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.APIKey != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != a.APIKey {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		next(w, r)
	}
}

// handleZones returns list of all zones (domains)
func (a *API) handleZones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.getZones(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleZoneRecords handles zone-specific record operations
func (a *API) handleZoneRecords(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/zones/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Zone required", http.StatusBadRequest)
		return
	}

	zone := parts[0]

	if len(parts) >= 2 && parts[1] == "records" {
		// /zones/{zone}/records - get, add or delete records
		switch r.Method {
		case http.MethodGet:
			a.getZoneRecords(w, r, zone)
		case http.MethodPost:
			a.addRecord(w, r, zone)
		case http.MethodDelete:
			// Handle DELETE with query parameters: /zones/{zone}/records?name={name}&type={type}
			name := r.URL.Query().Get("name")
			typeParam := r.URL.Query().Get("type")
			if name == "" || typeParam == "" {
				http.Error(w, "Name and type parameters are required", http.StatusBadRequest)
				return
			}
			a.deleteRecordByQuery(w, r, zone, name, typeParam)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Invalid path", http.StatusNotFound)
	}
}

// getZones returns all zones (domains) from zone file
func (a *API) getZones(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	zones, err := a.parseZones()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"zones": zones,
	})
}

// getZoneRecords returns all records for a zone (domain)
func (a *API) getZoneRecords(w http.ResponseWriter, r *http.Request, zone string) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	records, err := a.parseZoneRecords(zone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records": records,
	})
}

// addRecord adds a new record to a zone
func (a *API) addRecord(w http.ResponseWriter, r *http.Request, zone string) {
	var record struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Value string `json:"value"`
		TTL   int    `json:"ttl,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if record.Name == "" || record.Type == "" || record.Value == "" {
		http.Error(w, "Name, type and value are required", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.addRecordToZone(zone, record.Name, record.Type, record.Value, record.TTL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Record added successfully",
	})
}

// updateRecord updates an existing record
func (a *API) updateRecord(w http.ResponseWriter, r *http.Request, zone, recordID string) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// deleteRecordByQuery deletes a record by name and type query parameters
func (a *API) deleteRecordByQuery(w http.ResponseWriter, r *http.Request, zone, name, recordType string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.deleteRecordFromZone(zone, name, recordType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Record deleted successfully",
	})
}

// parseZones extracts all zones (domains) from zone file
func (a *API) parseZones() ([]map[string]string, error) {
	content, err := os.ReadFile(a.ZoneFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]string{}, nil
		}
		return nil, err
	}

	re := regexp.MustCompile(`\$ORIGIN\s+(\S+)\.`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	zones := make([]map[string]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			zones = append(zones, map[string]string{
				"name": match[1],
			})
		}
	}

	return zones, nil
}

// parseZoneRecords extracts records for a specific zone (domain)
func (a *API) parseZoneRecords(zone string) ([]map[string]interface{}, error) {
	content, err := os.ReadFile(a.ZoneFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}

	records := []map[string]interface{}{}
	lines := strings.Split(string(content), "\n")

	inZone := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.Contains(line, "$ORIGIN "+zone+".") {
			inZone = true
			continue
		}

		if inZone && strings.HasPrefix(line, "$ORIGIN") {
			break
		}

		if inZone && !strings.HasPrefix(line, "$") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				record := map[string]interface{}{
					"name":  parts[0],
					"type":  parts[2],
					"value": strings.Join(parts[3:], " "),
				}

				// Parse TTL if present
				if ttl, err := strconv.Atoi(parts[1]); err == nil {
					record["ttl"] = ttl
				} else {
					record["ttl"] = 3600
				}

				records = append(records, record)
			}
		}
	}

	return records, nil
}

// addRecordToZone adds a record to the zone file
func (a *API) addRecordToZone(zone, name, recordType, value string, ttl int) error {
	if ttl == 0 {
		ttl = 3600
	}

	recordType = strings.ToUpper(recordType)

	// Read existing content
	content, err := os.ReadFile(a.ZoneFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	zoneContent := string(content)

	// Check if zone exists
	if !strings.Contains(zoneContent, "$ORIGIN "+zone+".") {
		// Add new zone at the end
		newZone := fmt.Sprintf(`
; %s zone
$ORIGIN %s.
$TTL 3600

@       IN      SOA     ns1.%s. admin.%s. (
					2024010101 ; serial
					3600       ; refresh
					1800       ; retry
					604800     ; expire
					86400 )    ; minimum

@       IN      NS      ns1.%s.
@       IN      NS      ns2.%s.
`, zone, zone, zone, zone, zone, zone)
		zoneContent += newZone
	}

	// Add record
	recordLine := fmt.Sprintf("%s\t%d\tIN\t%s\t%s\n", name, ttl, recordType, value)

	zoneStart, zoneEnd, found := findZoneBounds(zoneContent, zone)
	if !found {
		zoneContent += recordLine
	} else {
		zoneSection := zoneContent[zoneStart:zoneEnd]

		// Keep CNAME unique per owner name within the current zone.
		if recordType == "CNAME" {
			zoneSection = removeZoneRecords(zoneSection, name, recordType, "")
		} else {
			zoneSection = removeZoneRecords(zoneSection, name, recordType, value)
		}

		if zoneSection != "" && !strings.HasSuffix(zoneSection, "\n") {
			zoneSection += "\n"
		}
		zoneSection += recordLine
		zoneContent = zoneContent[:zoneStart] + zoneSection + zoneContent[zoneEnd:]
	}

	// Write back to file
	return os.WriteFile(a.ZoneFile, []byte(cleanZoneContent(zoneContent)), 0644)
}

// setup function initializes the plugin
func setup(c *caddy.Controller) error {
	api := &API{}

	for c.Next() {
		// Parse plugin configuration
		for c.NextBlock() {
			switch c.Val() {
			case "address":
				if !c.NextArg() {
					return c.ArgErr()
				}
				api.Address = c.Val()
			case "apikey":
				if !c.NextArg() {
					return c.ArgErr()
				}
				api.APIKey = c.Val()
			case "zone_file":
				if !c.NextArg() {
					return c.ArgErr()
				}
				api.ZoneFile = c.Val()
			case "ecs_log":
				if !c.NextArg() {
					return c.ArgErr()
				}
				enabled, err := parseBoolOption(c.Val())
				if err != nil {
					return c.Errf("ecs_log expects on/off (or true/false), got %q", c.Val())
				}
				api.ECSLog = enabled
			case "goedge-city-node-routing":
				if !c.NextArg() {
					return c.ArgErr()
				}
				enabled, err := parseBoolOption(c.Val())
				if err != nil {
					return c.Errf("goedge-city-node-routing expects on/off (or true/false), got %q", c.Val())
				}
				api.CityNodeRouting.Enabled = enabled
			case "goedge-city-node-routing-mysql-dsn":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return c.ArgErr()
				}
				api.CityNodeRouting.MySQLDSN = strings.Join(args, "")
			case "goedge-city-node-routing-cluster-id":
				if !c.NextArg() {
					return c.ArgErr()
				}
				clusterID, err := strconv.ParseInt(c.Val(), 10, 64)
				if err != nil || clusterID <= 0 {
					return c.Errf("goedge-city-node-routing-cluster-id expects positive integer, got %q", c.Val())
				}
				api.CityNodeRouting.ClusterID = clusterID
			case "goedge-city-node-routing-refresh":
				if !c.NextArg() {
					return c.ArgErr()
				}
				dur, err := time.ParseDuration(c.Val())
				if err != nil || dur <= 0 {
					return c.Errf("invalid goedge-city-node-routing-refresh value %q", c.Val())
				}
				api.CityNodeRouting.RefreshInterval = dur
			case "goedge-city-node-routing-timeout":
				if !c.NextArg() {
					return c.ArgErr()
				}
				dur, err := time.ParseDuration(c.Val())
				if err != nil || dur <= 0 {
					return c.Errf("invalid goedge-city-node-routing-timeout value %q", c.Val())
				}
				api.CityNodeRouting.QueryTimeout = dur
			case "goedge-city-node-routing-ttl":
				if !c.NextArg() {
					return c.ArgErr()
				}
				ttl, err := strconv.ParseUint(c.Val(), 10, 32)
				if err != nil {
					return c.Errf("invalid goedge-city-node-routing-ttl value %q", c.Val())
				}
				api.CityNodeRouting.TTL = uint32(ttl)
			case "goedge-city-node-routing-fqdn":
				if !c.NextArg() {
					return c.ArgErr()
				}
				api.CityNodeRouting.FQDN = c.Val()
			case "goedge-city-node-routing-role":
				if !c.NextArg() {
					return c.ArgErr()
				}
				api.CityNodeRouting.NodeRole = c.Val()
			default:
				return c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	// Set default address if not specified
	if api.Address == "" {
		api.Address = ":8080"
	}

	if api.CityNodeRouting.Enabled {
		router, err := newGoedgeCityNodeRouter(api.CityNodeRouting)
		if err != nil {
			return err
		}
		api.cityNodeRouter = router
	}

	// Start HTTP server
	if err := api.startHTTPServer(); err != nil {
		return err
	}

	// Add to DNS server
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		api.Next = next
		return api
	})

	return nil
}

// deleteRecordFromZone deletes a record from the zone file by name and type
func (a *API) deleteRecordFromZone(zone, name, recordType string) error {
	recordType = strings.ToUpper(recordType)

	// Read existing content
	content, err := os.ReadFile(a.ZoneFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	zoneContent := string(content)

	zoneStart, zoneEnd, found := findZoneBounds(zoneContent, zone)
	if !found {
		return nil
	}

	zoneSection := zoneContent[zoneStart:zoneEnd]
	zoneSection = removeZoneRecords(zoneSection, name, recordType, "")
	zoneContent = zoneContent[:zoneStart] + zoneSection + zoneContent[zoneEnd:]

	// Write back to file
	return os.WriteFile(a.ZoneFile, []byte(cleanZoneContent(zoneContent)), 0644)
}

func findZoneBounds(content, zone string) (start, end int, found bool) {
	zoneMarker := "$ORIGIN " + zone + "."
	start = strings.Index(content, zoneMarker)
	if start == -1 {
		return 0, 0, false
	}

	remaining := content[start:]
	nextOrigin := strings.Index(remaining, "\n$ORIGIN ")
	if nextOrigin == -1 {
		return start, len(content), true
	}

	return start, start + nextOrigin + 1, true
}

func removeZoneRecords(zoneSection, name, recordType, value string) string {
	pattern := fmt.Sprintf(`(?m)^%s\s+\d+\s+IN\s+%s\s+`, regexp.QuoteMeta(name), regexp.QuoteMeta(recordType))
	if value != "" {
		pattern += regexp.QuoteMeta(value) + `\s*(?:\n|$)`
	} else {
		pattern += `.*(?:\n|$)`
	}

	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(zoneSection, "")
}

func cleanZoneContent(content string) string {
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	return content + "\n"
}
