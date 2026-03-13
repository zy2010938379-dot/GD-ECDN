// Package geoip implements an MMDB database plugin for geo/network IP lookups.
package geoip

import (
	"context"
	"fmt"
	"net/netip"
	"path/filepath"
	"sync"

	"github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang/v2"
)

var log = clog.NewWithPlugin(pluginName)

// GeoIP is a plugin that adds geo location and network data to the request context by looking up
// an MMDB format database, and which data can be later consumed by other middlewares.
type GeoIP struct {
	Next              plugin.Handler
	db                db
	edns0             bool
	ecsFallbackPolicy ECSFallbackPolicy
	goedgeCity        bool
	goedgeCityMySQL   *goedgeCityMySQLStore
}

type db struct {
	*geoip2.Reader
	// provides defines the schemas that can be obtained by querying this database, by using
	// bitwise operations.
	provides int
}

const (
	city = 1 << iota
	asn
)

var probingIP = netip.MustParseAddr("127.0.0.1")

var (
	initGoEdgeLibraryOnce sync.Once
	initGoEdgeLibraryErr  error
)

func newGeoIP(dbPath string, edns0 bool, fallbackPolicy ECSFallbackPolicy, goedgeCity bool) (*GeoIP, error) {
	return newGeoIPWithMySQLConfig(dbPath, edns0, fallbackPolicy, goedgeCity, GoEdgeCityMySQLConfig{})
}

func newGeoIPWithMySQLConfig(dbPath string, edns0 bool, fallbackPolicy ECSFallbackPolicy, goedgeCity bool, mysqlCfg GoEdgeCityMySQLConfig) (*GeoIP, error) {
	db := db{}
	if dbPath != "" {
		reader, err := geoip2.Open(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database file: %v", err)
		}
		db.Reader = reader
		schemas := []struct {
			provides int
			name     string
			validate func() error
		}{
			{name: "city", provides: city, validate: func() error { _, err := reader.City(probingIP); return err }},
			{name: "asn", provides: asn, validate: func() error { _, err := reader.ASN(probingIP); return err }},
		}
		// Query the database to figure out the database type.
		for _, schema := range schemas {
			if err := schema.validate(); err != nil {
				// If we get an InvalidMethodError then we know this database does not provide that schema.
				if _, ok := err.(geoip2.InvalidMethodError); !ok {
					return nil, fmt.Errorf("unexpected failure looking up database %q schema %q: %v", filepath.Base(dbPath), schema.name, err)
				}
			} else {
				db.provides |= schema.provides
			}
		}
	}

	if db.provides == 0 && !goedgeCity {
		return nil, fmt.Errorf("database does not provide any supported schema (city, asn)")
	}

	if goedgeCity {
		initGoEdgeLibraryOnce.Do(func() {
			initGoEdgeLibraryErr = iplibrary.InitDefault()
		})
		if initGoEdgeLibraryErr != nil {
			return nil, fmt.Errorf("failed to initialize goedge ip library: %w", initGoEdgeLibraryErr)
		}
	}

	var cityStore *goedgeCityMySQLStore
	if mysqlCfg.Enabled() {
		store, err := newGoEdgeCityMySQLStore(mysqlCfg)
		if err != nil {
			return nil, err
		}
		cityStore = store
	}

	return &GeoIP{
		db:                db,
		edns0:             edns0,
		ecsFallbackPolicy: fallbackPolicy,
		goedgeCity:        goedgeCity,
		goedgeCityMySQL:   cityStore,
	}, nil
}

// ServeDNS implements the plugin.Handler interface.
func (g GeoIP) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(pluginName, g.Next, ctx, w, r)
}

// Metadata implements the metadata.Provider Interface in the metadata plugin, and is used to store
// the data associated with the source IP of every request.
func (g GeoIP) Metadata(ctx context.Context, state request.Request) context.Context {
	srcIP, source, fallbackReason, ok := g.resolveEffectiveClientIP(state)
	if !ok {
		return ctx
	}

	g.setClientIPMetadata(ctx, srcIP.String(), source, fallbackReason)
	g.observeClientIPSource(source, fallbackReason)

	if g.db.provides&city != 0 {
		data, err := g.db.City(srcIP)
		if err != nil {
			log.Debugf("Setting up city metadata failed due to database lookup error: %v", err)
			g.observeCityLookup(false)
		} else {
			g.setCityMetadata(ctx, data)
			g.observeCityLookup(data.City.Names.English != "")
		}
	}
	if g.db.provides&asn != 0 {
		data, err := g.db.ASN(srcIP)
		if err != nil {
			log.Debugf("Setting up asn metadata failed due to database lookup error: %v", err)
		} else {
			g.setASNMetadata(ctx, data)
		}
	}

	if g.goedgeCity {
		result := g.lookupGoEdgeCity(srcIP, source, fallbackReason)
		g.setGoEdgeCityMetadata(ctx, result)
		g.observeGoEdgeCityLookup(result)
	}

	return ctx
}

// Name implements the Handler interface.
func (g GeoIP) Name() string { return pluginName }
