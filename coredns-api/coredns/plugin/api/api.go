package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

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
	Address  string `json:"address,omitempty"`
	APIKey   string `json:"apikey,omitempty"`
	ZoneFile string `json:"zone_file,omitempty"`

	// Internal state
	mu         sync.RWMutex
	httpServer *http.Server
}

// ServeDNS implements the plugin.Handler interface
func (a *API) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
}

// Name implements the plugin.Handler interface
func (a *API) Name() string { return "api" }

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

	// Insert record in the correct zone
	zoneStart := strings.Index(zoneContent, "$ORIGIN "+zone+".")
	if zoneStart == -1 {
		// Should not happen since we just added the zone
		zoneContent += recordLine
	} else {
		// Find the end of this zone (next $ORIGIN or end of file)
		remaining := zoneContent[zoneStart:]
		nextOrigin := strings.Index(remaining, "\n$ORIGIN ")

		var insertionPoint int
		if nextOrigin == -1 {
			// No next zone, insert at the end
			insertionPoint = len(zoneContent)
		} else {
			// Insert before next zone
			insertionPoint = zoneStart + nextOrigin
		}

		// Insert the record at the end of the current zone
		zoneContent = zoneContent[:insertionPoint] + recordLine + zoneContent[insertionPoint:]
	}

	// Write back to file
	return os.WriteFile(a.ZoneFile, []byte(zoneContent), 0644)
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
			default:
				return c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	// Set default address if not specified
	if api.Address == "" {
		api.Address = ":8080"
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
	// Read existing content
	content, err := os.ReadFile(a.ZoneFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	zoneContent := string(content)

	// Build pattern to match the record line
	pattern := fmt.Sprintf(`(?m)^%s\s+\d+\s+IN\s+%s\s+.*$`, regexp.QuoteMeta(name), regexp.QuoteMeta(recordType))
	re := regexp.MustCompile(pattern)

	// Remove the matching record
	zoneContent = re.ReplaceAllString(zoneContent, "")

	// Clean up empty lines
	zoneContent = strings.ReplaceAll(zoneContent, "\n\n\n", "\n\n")
	zoneContent = strings.TrimSpace(zoneContent) + "\n"

	// Write back to file
	return os.WriteFile(a.ZoneFile, []byte(zoneContent), 0644)
}
