module github.com/TeaOSLab/EdgeUser

go 1.22

require (
	github.com/TeaOSLab/EdgeCommon v0.0.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang/protobuf v1.5.3
	github.com/google/uuid v1.3.0
	github.com/redis/go-redis/v9 v9.0.5
	google.golang.org/grpc v1.56.0
	google.golang.org/protobuf v1.31.0
)

replace github.com/TeaOSLab/EdgeCommon => ../EdgeCommon