package dnsclients

import (
	"testing"

	"github.com/iwind/TeaGo/maps"
)

func TestCoreDNSProvider_Auth(t *testing.T) {
	provider := &CoreDNSProvider{}
	
	// 测试正常认证
	err := provider.Auth(maps.Map{
		"url": "http://localhost:8080",
		"apiKey": "test-key",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 测试缺少URL
	err = provider.Auth(maps.Map{
		"apiKey": "test-key",
	})
	if err == nil {
		t.Fatal("should return error when url is empty")
	}
}

func TestCoreDNSProvider_MaskParams(t *testing.T) {
	provider := &CoreDNSProvider{}
	
	params := maps.Map{
		"url":    "http://localhost:8080",
		"apiKey": "secret-key",
	}
	
	provider.MaskParams(params)
	
	if params.GetString("apiKey") != "***" {
		t.Fatal("apiKey should be masked")
	}
	
	if params.GetString("url") != "http://localhost:8080" {
		t.Fatal("url should not be masked")
	}
}