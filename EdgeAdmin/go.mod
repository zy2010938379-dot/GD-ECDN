module github.com/TeaOSLab/EdgeAdmin

go 1.24.0

toolchain go1.24.12

replace github.com/TeaOSLab/EdgeCommon => ../EdgeCommon

// replace github.com/TeaOSLab/EdgePlus => ../EdgePlus

require (
	github.com/TeaOSLab/EdgeCommon v0.0.0-00010101000000-000000000000
	// github.com/TeaOSLab/EdgePlus v0.0.0-00010101000000-000000000000
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/iwind/TeaGo v0.0.0-20240508072741-7647e70b7070
	github.com/iwind/gosock v0.0.0-20220505115348-f88412125a62
	github.com/miekg/dns v1.1.61
	github.com/shirou/gopsutil/v3 v3.24.2
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/tealeg/xlsx/v3 v3.2.4
	github.com/xlzd/gotp v0.1.0
	golang.org/x/crypto v0.44.0
	golang.org/x/sys v0.38.0
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require golang.org/x/net v0.47.0

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/frankban/quicktest v1.11.3 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/shabbyrobe/xmlwriter v0.0.0-20200208144257-9fca06d00ffa // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
