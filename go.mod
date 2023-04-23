module github.com/UNO-SOFT/otel

go 1.19

require (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.40.0
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/metric v0.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.14.0
	go.opentelemetry.io/otel/sdk/metric v0.37.0
	go.opentelemetry.io/otel/trace v1.14.0
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.54.0
)

require (
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.37.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.14.0
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
