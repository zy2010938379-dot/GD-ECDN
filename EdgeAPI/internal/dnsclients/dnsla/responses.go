package dnsla

// DomainListResponse DNSLa 域名列表响应
type DomainListResponse struct {
	BaseResponse
	Data DomainListData `json:"data"`
}

type DomainListData struct {
	Results []DomainResult `json:"results"`
}

type DomainResult struct {
	Domain string `json:"domain"`
}

// RecordListResponse DNSLa 记录列表响应
type RecordListResponse struct {
	BaseResponse
	Data RecordListData `json:"data"`
}

type RecordListData struct {
	Results []RecordResult `json:"results"`
}

type RecordResult struct {
	Id       string `json:"id"`
	Host     string `json:"host"`
	Type     string `json:"type"`
	Data     string `json:"data"`
	LineCode string `json:"lineCode"`
	TTL      int32  `json:"ttl"`
}

// AllLineListResponse DNSLa 所有线路列表响应
type AllLineListResponse struct {
	BaseResponse
	Data []AllLineListResponseChild `json:"data"`
}

type AllLineListResponseChild struct {
	Id       string                     `json:"id"`
	Code     string                     `json:"code"`
	Name     string                     `json:"name"`
	Children []AllLineListResponseChild `json:"children"`
}

// RecordCreateResponse DNSLa 记录创建响应
type RecordCreateResponse struct {
	BaseResponse
	Data RecordCreateData `json:"data"`
}

type RecordCreateData struct {
	Id int64 `json:"id"`
}

// RecordUpdateResponse DNSLa 记录更新响应
type RecordUpdateResponse struct {
	BaseResponse
}

// RecordDeleteResponse DNSLa 记录删除响应
type RecordDeleteResponse struct {
	BaseResponse
}

// DomainResponse DNSLa 域名响应
type DomainResponse struct {
	BaseResponse
	Data DomainData `json:"data"`
}

type DomainData struct {
	Id string `json:"id"`
}

// BaseResponse DNSLa 基础响应
type BaseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Success 检查响应是否成功
func (r *BaseResponse) Success() bool {
	return r.Code == 200
}

// Error 返回错误信息
func (r *BaseResponse) Error() error {
	if r.Success() {
		return nil
	}
	return &DNSLaError{Code: r.Code, Message: r.Message}
}

// DNSLaError DNSLa 错误类型
type DNSLaError struct {
	Code    int
	Message string
}

func (e *DNSLaError) Error() string {
	return e.Message
}
