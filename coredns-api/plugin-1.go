package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

func init() {
	caddy.RegisterPlugin("api", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

// API represents the API plugin
type API struct {
	Next     plugin.Handler
	Address  string
	APIKey   string
	ZoneFile string
	mu       sync.RWMutex
}

// ServeDNS implements the plugin.Handler interface
func (a *API) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
}

// Name implements the plugin.Handler interface
func (a *API) Name() string { return "api" }

// setup function to initialize the API plugin
func setup(c *caddy.Controller) error {
	api := &API{}

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) > 0 {
			return plugin.Error("api", c.ArgErr())
		}

		for c.NextBlock() {
			switch c.Val() {
			case "address":
				if !c.NextArg() {
					return plugin.Error("api", c.ArgErr())
				}
				api.Address = c.Val()
			case "apikey":
				if !c.NextArg() {
					return plugin.Error("api", c.ArgErr())
				}
				api.APIKey = c.Val()
			case "zone_file":
				if !c.NextArg() {
					return plugin.Error("api", c.ArgErr())
				}
				api.ZoneFile = c.Val()
			default:
				return plugin.Error("api", c.Errf("unknown property '%s'", c.Val()))
			}
		}
	}

	// Set defaults if not provided
	if api.Address == "" {
		api.Address = ":8080"
	}

	if api.ZoneFile == "" {
		api.ZoneFile = "/etc/coredns/zones.db"
	}

	// Start HTTP server in a goroutine
	go api.startHTTPServer()

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		api.Next = next
		return api
	})

	return nil
}

// startHTTPServer starts the HTTP API server
func (a *API) startHTTPServer() {
	http.HandleFunc("/domains", a.handleDomains)
	http.HandleFunc("/domains/", a.handleDomainRecords)

	log.Printf("[API] Starting HTTP server on %s\n", a.Address)
	if err := http.ListenAndServe(a.Address, nil); err != nil {
		log.Printf("[API] Failed to start HTTP server: %v\n", err)
	}
}

// handleDomains handles GET /domains endpoint
func (a *API) handleDomains(w http.ResponseWriter, r *http.Request) {
	if !a.authenticate(w, r) {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read zone file and extract domains
	content, err := ioutil.ReadFile(a.ZoneFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read zone file: %v", err), http.StatusInternalServerError)
		return
	}

	domains := extractDomains(string(content))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

// handleDomainRecords handles domain-specific record operations
func (a *API) handleDomainRecords(w http.ResponseWriter, r *http.Request) {
	if !a.authenticate(w, r) {
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/domains/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	domain := pathParts[0]

	switch r.Method {
	case "GET":
		a.getRecords(w, r, domain)
	case "POST":
		a.addRecord(w, r, domain)
	case "PUT":
		if len(pathParts) >= 2 {
			a.updateRecord(w, r, domain, pathParts[1])
		} else {
			http.Error(w, "Record ID required", http.StatusBadRequest)
		}
	case "DELETE":
		if len(pathParts) >= 2 {
			a.deleteRecord(w, r, domain, pathParts[1])
		} else {
			http.Error(w, "Record ID required", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// authenticate checks API key authentication
func (a *API) authenticate(w http.ResponseWriter, r *http.Request) bool {
	if a.APIKey == "" {
		return true // No authentication required
	}

	providedKey := r.Header.Get("X-API-Key")
	if providedKey != a.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// extractDomains extracts domain names from zone file content
func extractDomains(content string) []string {
	var domains []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, " IN SOA ") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				domains = append(domains, parts[0])
			}
		}
	}

	return domains
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// getRecords retrieves all records for a domain
func (a *API) getRecords(w http.ResponseWriter, r *http.Request, domain string) {
	// Implementation for getting records
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]DNSRecord{})
}

// addRecord adds a new record to a domain
func (a *API) addRecord(w http.ResponseWriter, r *http.Request, domain string) {
	var record DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Implementation for adding record
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// updateRecord updates an existing record
func (a *API) updateRecord(w http.ResponseWriter, r *http.Request, domain, recordID string) {
	var record DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Implementation for updating record
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// deleteRecord deletes a record
func (a *API) deleteRecord(w http.ResponseWriter, r *http.Request, domain, recordID string) {
	// Implementation for deleting record
	w.WriteHeader(http.StatusNoContent)
}

// main function for standalone testing
func main() {
	fmt.Println("CoreDNS API Plugin")
	fmt.Println("This plugin is designed to be used with CoreDNS.")
	fmt.Println("To use it, add 'api' directive to your Corefile.")
}
