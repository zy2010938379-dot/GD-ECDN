package geoip

import (
	"context"
	"net/netip"
	"strconv"
	"strings"

	"github.com/TeaOSLab/EdgeCommon/pkg/iplibrary"
	"github.com/coredns/coredns/plugin/metadata"
)

type GoEdgeCityMappingResult struct {
	EffectiveClientIP string
	ClientIPSource    string
	FallbackReason    string

	CountryID    int64
	CountryName  string
	ProvinceID   int64
	ProvinceName string
	CityID       int64
	CityName     string
	ProviderID   int64
	ProviderName string
	Summary      string
	Hit          bool
}

func (g GeoIP) lookupGoEdgeCity(clientIP netip.Addr, source, fallbackReason string) GoEdgeCityMappingResult {
	result := GoEdgeCityMappingResult{
		EffectiveClientIP: clientIP.String(),
		ClientIPSource:    source,
		FallbackReason:    fallbackReason,
	}

	lookup := iplibrary.LookupIP(clientIP.String())
	if lookup == nil || !lookup.IsOk() {
		return result
	}

	result.CountryID = lookup.CountryId()
	result.CountryName = lookup.CountryName()
	result.ProvinceID = lookup.ProvinceId()
	result.ProvinceName = lookup.ProvinceName()
	result.CityID = lookup.CityId()
	result.CityName = lookup.CityName()
	result.ProviderID = lookup.ProviderId()
	result.ProviderName = lookup.ProviderName()
	result.Summary = lookup.Summary()
	result.Hit = result.CityID > 0 || len(strings.TrimSpace(result.CityName)) > 0
	return result
}

func (g GeoIP) setGoEdgeCityMetadata(ctx context.Context, result GoEdgeCityMappingResult) {
	metadata.SetValueFunc(ctx, pluginName+"/goedge/client/ip", func() string {
		return result.EffectiveClientIP
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/client/ip_source", func() string {
		return result.ClientIPSource
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/client/ip_fallback_reason", func() string {
		return result.FallbackReason
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/country/id", func() string {
		return strconv.FormatInt(result.CountryID, 10)
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/country/name", func() string {
		return result.CountryName
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/province/id", func() string {
		return strconv.FormatInt(result.ProvinceID, 10)
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/province/name", func() string {
		return result.ProvinceName
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/city/id", func() string {
		return strconv.FormatInt(result.CityID, 10)
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/city/name", func() string {
		return result.CityName
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/provider/id", func() string {
		return strconv.FormatInt(result.ProviderID, 10)
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/provider/name", func() string {
		return result.ProviderName
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/summary", func() string {
		return result.Summary
	})
	metadata.SetValueFunc(ctx, pluginName+"/goedge/city/hit", func() string {
		return strconv.FormatBool(result.Hit)
	})
}
