# geoip

## Name

*geoip* - Lookup `.mmdb` ([MaxMind db file format](https://maxmind.github.io/MaxMind-DB/)) databases using the client IP, then add associated geoip data to the context request.

## Description

The *geoip* plugin allows you to enrich the data associated with Client IP addresses, e.g. geoip information like City, Country, and Network ASN. GeoIP data is commonly available in the `.mmdb` format, a database format that maps IPv4 and IPv6 addresses to data records using a binary search tree.

The data is added leveraging the *metadata* plugin, values can then be retrieved using it as well.

**Longitude example:**

```go
import (
    "strconv"
    "github.com/coredns/coredns/plugin/metadata"
)
// ...
if getLongitude := metadata.ValueFunc(ctx, "geoip/longitude"); getLongitude != nil {
    if longitude, err := strconv.ParseFloat(getLongitude(), 64); err == nil {
        // Do something useful with longitude.
    }
} else {
    // The metadata label geoip/longitude for some reason, was not set.
}
// ...
```

**City example:**

```go
import (
    "github.com/coredns/coredns/plugin/metadata"
)
// ...
if getCity := metadata.ValueFunc(ctx, "geoip/city/name"); getCity != nil {
    city := getCity()
    // Do something useful with city.
} else {
    // The metadata label geoip/city/name for some reason, was not set.
}
// ...
```

**ASN example:**

```go
import (
    "strconv"
    "github.com/coredns/coredns/plugin/metadata"
)
// ...
if getASN := metadata.ValueFunc(ctx, "geoip/asn/number"); getASN != nil {
    if asn, err := strconv.ParseUint(getASN(), 10, 32); err == nil {
        // Do something useful with asn.
    }
}
if getASNOrg := metadata.ValueFunc(ctx, "geoip/asn/org"); getASNOrg != nil {
    asnOrg := getASNOrg()
    // Do something useful with asnOrg.
}
// ...
```

## Databases

The supported databases use city schema such as `ASN`, `City`, and `Enterprise`. `.mmdb` files are generally supported, as long as their field names correctly map to the Metadata Labels below. Other database types with different schemas are not supported yet.

Free and commercial GeoIP `.mmdb` files are commonly available from vendors like [MaxMind](https://dev.maxmind.com/geoip/docs/databases), [IPinfo](https://ipinfo.io/developers/database-download), and [IPtoASN](https://iptoasn.com/) which is [Public Domain-licensed](https://opendatacommons.org/licenses/pddl/1-0/).

## Syntax

```text
geoip [DBFILE]
```

or

```text
geoip [DBFILE] {
    [edns-subnet]
    [ecs-fallback resolver-ip|disabled]
    [goedge-city]
}
```

* **DBFILE** the `mmdb` database file path. We recommend updating your `mmdb` database periodically for more accurate results.
  If `goedge-city` is enabled, `DBFILE` becomes optional.
* `edns-subnet`: Optional. Use [EDNS0 subnet](https://en.wikipedia.org/wiki/EDNS_Client_Subnet) (if present) for Geo IP instead of the source IP of the DNS request. This helps identifying the closest source IP address through intermediary DNS resolvers, and it also makes GeoIP testing easy: `dig +subnet=1.2.3.4 @dns-server.example.com www.geo-aware.com`.
* `ecs-fallback`: Optional. ECS unavailable/invalid时的回退策略。
  * `resolver-ip` (default): 使用递归解析器来源 IP 回退；
  * `disabled`: 不回退，直接跳过 geoip 元数据填充。
* `goedge-city`: Optional. 使用 GoEdge 内置 IP 库执行 `IP -> country/province/city/provider` 映射，并输出 `geoip/goedge/*` 元数据。

  **NOTE:** due to security reasons, recursive DNS resolvers may mask a few bits off of the clients' IP address, which can cause inaccuracies in GeoIP resolution.

  There is no defined mask size in the standards, but there are examples: [RFC 7871's example](https://datatracker.ietf.org/doc/html/rfc7871#section-13) conceals the last 72 bits of an IPv6 source address, and NS1 Help Center [mentions](https://help.ns1.com/hc/en-us/articles/360020256573-About-the-EDNS-Client-Subnet-ECS-DNS-extension) that ECS-enabled DNS resolvers send only the first three octets (eg. /24) of the source IPv4 address.

## Examples

The following configuration configures the `City` database, and looks up geolocation based on EDNS0 subnet if present.

```txt
. {
    geoip /opt/geoip2/db/GeoLite2-City.mmdb {
      edns-subnet
      ecs-fallback resolver-ip
    }
    metadata # Note that metadata plugin must be enabled as well.
}
```

仅启用 GoEdge 城市映射：

```txt
. {
    geoip {
      edns-subnet
      ecs-fallback resolver-ip
      goedge-city
    }
    metadata
}
```

The *view* plugin can use *geoip* metadata as selection criteria to provide GSLB functionality.
In this example, clients from the city "Exampleshire" will receive answers for `example.com` from the zone defined in 
`example.com.exampleshire-db`. All other clients will receive answers from the zone defined in `example.com.db`.
Note that the order of the two `example.com` server blocks below is important; the default viewless server block
must be last.

```txt
example.com {
    view exampleshire {
      expr metadata('geoip/city/name') == 'Exampleshire'
    }
    geoip /opt/geoip2/db/GeoLite2-City.mmdb
    metadata
    file example.com.exampleshire-db
}

example.com {
    file example.com.db
}
```

GoEdge 城市元数据也可用于 view 选择（城市优先，默认块即回退）：

```txt
example.com {
    view guangzhou {
      expr metadata('geoip/goedge/city/name') == '广州市'
    }
    geoip {
      edns-subnet
      ecs-fallback resolver-ip
      goedge-city
    }
    metadata
    file example.com.gz-db
}

example.com {
    file example.com.default-db
}
```

## Production Rollout Playbook (ECS City Routing)

建议采用分阶段灰度发布，并绑定明确告警阈值：

1. **Baseline（开关关闭）**: 先观察 24h 基线（延迟、错误率、命中率）。
2. **Canary（小流量）**: 仅对少量域名或线路开启 `goedge-city`（建议 1%~5%）。
3. **Batch（分批放量）**: 逐步扩大到 10% -> 30% -> 60%。
4. **Full（全量）**: 指标稳定后全量开启。

推荐观测指标与阈值（可按业务调整）：

- `coredns_geoip_effective_client_ip_source_total{source="ecs"}`:
  ECS 来源占比相对基线下降 >20% 持续 10 分钟，告警。
- `coredns_geoip_goedge_city_lookup_total{result="hit"}`:
  城市命中率相对基线下降 >15% 持续 10 分钟，告警。
- DNS 解析延迟（p95）:
  相对基线上升 >20ms 或 >20%，告警。
- SERVFAIL 比例:
  超过 0.5% 或高于基线 2 倍，立即回滚。

回滚策略：

1. 立即关闭 `goedge-city`（保留 `geoip` 基础能力）。
2. 如仍异常，关闭 `edns-subnet` 路径，恢复原有来源 IP 路由。
3. 保留问题时段日志与指标，定位 `ecs_missing/ecs_malformed` 与 `city miss` 的主因后再重试灰度。

## Metadata Labels

A limited set of fields will be exported as labels, all values are stored using strings **regardless of their underlying value type**, and therefore you may have to convert it back to its original type, note that numeric values are always represented in base 10.

| Label                                | Type      | Example          | Description
| :----------------------------------- | :-------- | :--------------  | :------------------
| `geoip/city/name`                    | `string`  | `Cambridge`      | Then city name in English language.
| `geoip/country/code`                 | `string`  | `GB`             | Country [ISO 3166-1](https://en.wikipedia.org/wiki/ISO_3166-1) code.
| `geoip/country/name`                 | `string`  | `United Kingdom` | The country name in English language.
| `geoip/country/is_in_european_union` | `bool`    | `false`          | Either `true` or `false`.
| `geoip/continent/code`               | `string`  | `EU`             | See [Continent codes](#ContinentCodes).
| `geoip/continent/name`               | `string`  | `Europe`         | The continent name in English language.
| `geoip/latitude`                     | `float64` | `52.2242`        | Base 10, max available precision.
| `geoip/longitude`                    | `float64` | `0.1315`         | Base 10, max available precision.
| `geoip/timezone`                     | `string`  | `Europe/London`  | The timezone.
| `geoip/postalcode`                   | `string`  | `CB4`            | The postal code.
| `geoip/subdivisions/code`            | `string`  | `ENG,TWH`        | Comma separated [ISO 3166-2](https://en.wikipedia.org/wiki/ISO_3166-2) subdivision(region) codes, e.g. first level (province), second level (state).
| `geoip/asn/number`                   | `uint`    | `396982`         | The autonomous system number.
| `geoip/asn/org`                      | `string`  | `GOOGLE-CLOUD-PLATFORM` | The autonomous system organization.
| `geoip/client/ip`                    | `string`  | `61.142.56.193`  | Effective client IP used for lookup.
| `geoip/client/ip_source`             | `string`  | `ecs`            | Effective client IP source (`ecs` or `fallback`).
| `geoip/client/ip_fallback_reason`    | `string`  | `ecs_missing`    | Fallback reason when source is `fallback`.
| `geoip/goedge/city/id`               | `int64`   | `440100`         | GoEdge city ID.
| `geoip/goedge/city/name`             | `string`  | `广州市`          | GoEdge city name.
| `geoip/goedge/city/hit`              | `bool`    | `true`           | Whether GoEdge city mapping is a hit.

## Continent Codes

| Value | Continent (EN) |
| :---- | :------------- |
| AF    | Africa         |
| AN    | Antarctica     |
| AS    | Asia           |
| EU    | Europe         |
| NA    | North America  |
| OC    | Oceania        |
| SA    | South America  |

## Notable changes

- In CoreDNS v1.13.2, the `geoip` plugin was upgraded to use [`oschwald/geoip2-golang/v2`](https://github.com/oschwald/geoip2-golang/blob/main/MIGRATION.md), the Go library that reads and parses [`.mmdb`](https://maxmind.github.io/MaxMind-DB/) databases. It has a small, but possibly-breaking change, where the `Location.Latitude` and `Location.Longitude` structs changed from value types to pointers (`float64` → `*float64`). In `oschwald/geoip2-golang` v1, missing coordinates returned "0" (which is a [valid location](https://en.wikipedia.org/wiki/Null_Island)), and in v2 they now return an empty string "".
