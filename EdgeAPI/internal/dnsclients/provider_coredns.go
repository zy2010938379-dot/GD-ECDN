package dnsclients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	teaconst "github.com/TeaOSLab/EdgeAPI/internal/const"
	"github.com/TeaOSLab/EdgeAPI/internal/dnsclients/dnstypes"
	"github.com/TeaOSLab/EdgeAPI/internal/errors"
	"github.com/iwind/TeaGo/maps"
)

// CoreDNSProvider CoreDNS DNS服务商
// 注意：CoreDNS本身不提供HTTP API接口
// 此提供商仅作为示例，需要用户自行实现API网关或使用第三方工具
type CoreDNSProvider struct {
	url    string // CoreDNS API网关地址，例如：http://localhost:8080
	apiKey string // API密钥（可选）

	ProviderId int64

	BaseProvider
}

// Auth 认证
// 参数：
//   - url: CoreDNS API网关地址，例如：http://localhost:8080
//   - apiKey: API密钥（可选）
func (this *CoreDNSProvider) Auth(params maps.Map) error {
	this.url = params.GetString("url")
	if len(this.url) == 0 {
		return errors.New("'url' should not be empty")
	}

	// 确保URL以/结尾
	if this.url[len(this.url)-1] != '/' {
		this.url += "/"
	}

	this.apiKey = params.GetString("apiKey")

	return nil
}

// MaskParams 对参数进行掩码
func (this *CoreDNSProvider) MaskParams(params maps.Map) {
	params["apiKey"] = "***"
}

// GetDomains 获取所有域名列表
func (this *CoreDNSProvider) GetDomains() (domains []string, err error) {
	resp, err := this.request("GET", "zones", nil)
	if err != nil {
		return nil, err
	}

	var zoneResponse struct {
		Zones []struct {
			Name string `json:"name"`
		} `json:"zones"`
	}

	err = json.Unmarshal(resp, &zoneResponse)
	if err != nil {
		return nil, fmt.Errorf("parse zones response failed: %w", err)
	}

	for _, zone := range zoneResponse.Zones {
		domains = append(domains, zone.Name)
	}

	return domains, nil
}

// GetRecords 获取域名解析记录列表
func (this *CoreDNSProvider) GetRecords(domain string) (records []*dnstypes.Record, err error) {
	resp, err := this.request("GET", "zones/"+url.QueryEscape(domain)+"/records", nil)
	if err != nil {
		return nil, err
	}

	var zoneResponse struct {
		Records []struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			TTL   int32  `json:"ttl"`
			Value string `json:"value"`
		} `json:"records"`
	}

	err = json.Unmarshal(resp, &zoneResponse)
	if err != nil {
		return nil, fmt.Errorf("parse records response failed: %w", err)
	}

	for _, record := range zoneResponse.Records {
		records = append(records, &dnstypes.Record{
			Name:  record.Name,
			Type:  dnstypes.RecordType(record.Type),
			TTL:   record.TTL,
			Value: record.Value,
		})
	}

	return records, nil
}

// GetRoutes 读取域名支持的线路数据
// CoreDNS支持默认线路
func (this *CoreDNSProvider) GetRoutes(domain string) (routes []*dnstypes.Route, err error) {
	return []*dnstypes.Route{
		{
			Name: "默认",
			Code: "default",
		},
	}, nil
}

// QueryRecord 查询单个记录
func (this *CoreDNSProvider) QueryRecord(domain string, name string, recordType dnstypes.RecordType) (*dnstypes.Record, error) {
	records, err := this.QueryRecords(domain, name, recordType)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		return records[0], nil
	}

	return nil, nil
}

// QueryRecords 查询多个记录
func (this *CoreDNSProvider) QueryRecords(domain string, name string, recordType dnstypes.RecordType) (result []*dnstypes.Record, err error) {
	allRecords, err := this.GetRecords(domain)
	if err != nil {
		return nil, err
	}

	for _, record := range allRecords {
		if record.Name == name && record.Type == recordType {
			result = append(result, record)
		}
	}

	return result, nil
}

// AddRecord 添加记录
func (this *CoreDNSProvider) AddRecord(domain string, newRecord *dnstypes.Record) error {
	requestBody := map[string]interface{}{
		"name":  newRecord.Name,
		"type":  newRecord.Type,
		"ttl":   newRecord.TTL,
		"value": newRecord.Value,
	}

	// CoreDNS不支持线路功能，忽略线路参数
	// 但为了兼容GoEdge系统，我们仍然处理记录

	_, err := this.request("POST", "zones/"+url.QueryEscape(domain)+"/records", requestBody)
	return this.WrapError(err, domain, newRecord)
}

// UpdateRecord 修改记录
func (this *CoreDNSProvider) UpdateRecord(domain string, record *dnstypes.Record, newRecord *dnstypes.Record) error {
	// CoreDNS API通常通过删除旧记录并添加新记录来实现更新
	err := this.DeleteRecord(domain, record)
	if err != nil {
		return err
	}

	return this.AddRecord(domain, newRecord)
}

// DeleteRecord 删除记录
func (this *CoreDNSProvider) DeleteRecord(domain string, record *dnstypes.Record) error {
	// 构建删除请求URL
	url := fmt.Sprintf("zones/%s/records?name=%s&type=%s",
		url.QueryEscape(domain),
		url.QueryEscape(record.Name),
		url.QueryEscape(string(record.Type)))

	_, err := this.request("DELETE", url, nil)
	return this.WrapError(err, domain, record)
}

// DefaultRoute 默认线路
// CoreDNS支持默认线路
func (this *CoreDNSProvider) DefaultRoute() string {
	return "default"
}

// SetMinTTL 设置最小TTL
func (this *CoreDNSProvider) SetMinTTL(ttl int32) {
	this.BaseProvider.SetMinTTL(ttl)
}

// MinTTL 最小TTL
func (this *CoreDNSProvider) MinTTL() int32 {
	return this.BaseProvider.MinTTL()
}

// 发送HTTP请求到CoreDNS API
func (this *CoreDNSProvider) request(method string, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, this.url+path, reqBody)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("User-Agent", teaconst.ProductName+"/"+teaconst.Version)
	req.Header.Set("Content-Type", "application/json")

	// 如果有API密钥，添加到请求头
	if len(this.apiKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+this.apiKey)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("CoreDNS API returned status %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respData, nil
}
