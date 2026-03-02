package utils

import (
	"context"
	"net"
	"regexp"
	"sync"
	"time"

	"golang.org/x/net/idna"
)

func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func DnsCheck(domains []string, nodeIps []string) map[string]bool {
	var wg sync.WaitGroup
	ipMap := &sync.Map{}
	concurrent := 10
	counter := make(chan struct{}, concurrent)

	for _, domain := range domains {
		counter <- struct{}{}
		wg.Add(1)
		go func(domain string) {
			defer func() {
				<-counter
				wg.Done()
			}()
			// 中文域名转换
			if !regexp.MustCompile(`^[a-zA-Z0-9-.]+$`).MatchString(domain) {
				unicodeDomain, err := idna.ToASCII(domain)
				if err == nil && len(unicodeDomain) > 0 {
					domain = unicodeDomain
				}
			}
			dnsIps, err := DigTraceIP(domain)
			if err != nil || len(dnsIps) < 1 {
				ipMap.Store(domain, false)
			} else {
				var flag = false
				for _, ip := range dnsIps {
					flag = stringInSlice(ip, nodeIps)
					if flag {
						break
					}
				}
				ipMap.Store(domain, flag)
			}

		}(domain)
	}

	wg.Wait()

	resultMap := make(map[string]bool)
	ipMap.Range(func(key, value interface{}) bool {
		resultMap[key.(string)] = value.(bool)
		return true
	})

	return resultMap
}

func LookupIPWithTimeout(domain string, timeoutMS int64) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeoutMS))
	defer cancel()
	r := net.DefaultResolver
	ips, err := r.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

// DigTraceIP 执行DNS追踪，返回域名解析的IP地址列表
func DigTraceIP(domain string) ([]string, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return nil, err
	}
	return ips, nil
}
