module github.com/ldebruijn/graphql-protect

go 1.23.0

replace github.com/vektah/gqlparser/v2 v2.5.16 => github.com/ldebruijn/gqlparser/v2 v2.0.0-20240913154106-70bd43e5a479

require (
	cloud.google.com/go/storage v1.44.0
	github.com/jedib0t/go-pretty/v6 v6.6.0
	github.com/prometheus/client_golang v1.20.4
	github.com/stretchr/testify v1.9.0
	github.com/vektah/gqlparser/v2 v2.5.16
	go.opentelemetry.io/contrib/exporters/autoexport v0.55.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.55.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.55.0
	go.opentelemetry.io/contrib/propagators/autoprop v0.55.0
	go.opentelemetry.io/otel v1.31.0
	go.opentelemetry.io/otel/sdk v1.31.0
	google.golang.org/api v0.200.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cel.dev/expr v0.16.1 // indirect
	cloud.google.com/go v0.115.1 // indirect
	cloud.google.com/go/auth v0.9.8 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.4 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	cloud.google.com/go/iam v1.2.1 // indirect
	cloud.google.com/go/monitoring v1.21.1 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.24.1 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.48.1 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.48.1 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20240905190251-b4127c9b8d78 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/envoyproxy/go-control-plane v0.13.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.59.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/bridges/prometheus v0.55.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.29.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.54.0 // indirect
	go.opentelemetry.io/contrib/propagators/aws v1.30.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.30.0 // indirect
	go.opentelemetry.io/contrib/propagators/jaeger v1.30.0 // indirect
	go.opentelemetry.io/contrib/propagators/ot v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.6.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.6.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.52.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.6.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.30.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.30.0 // indirect
	go.opentelemetry.io/otel/log v0.6.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.6.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.30.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/oauth2 v0.23.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20241007155032-5fefd90f89a9 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240930140551-af27646dc61f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241007155032-5fefd90f89a9 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/grpc/stats/opentelemetry v0.0.0-20240907200651-3ffb98b2c93a // indirect
	google.golang.org/protobuf v1.35.1 // indirect
)
