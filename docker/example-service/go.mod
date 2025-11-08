module example-service

go 1.21

require (
	eve.evalgo.org/tracing v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
	github.com/labstack/echo/v4 v4.11.4
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.18.0
)

replace eve.evalgo.org/tracing => ../../tracing
