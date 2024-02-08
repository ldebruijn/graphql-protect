# Tracing 

Graphql Protect now supports OpenTelemetry-based tracing, enhancing observability and monitoring capabilities. 
Although the instrumentation is currently limited, it enables the creation of new spans that can be exported to 
any OTLP-compatible exporter.

## Exporting Traces

Tracing data exporting relies on [autoexport](https://pkg.go.dev/go.opentelemetry.io/contrib/exporters/autoexport#NewSpanExporter). 
Configuration is done via environment variables `OTEL_TRACES_EXPORTER` and `OTEL_EXPORTER_OTLP_PROTOCOL`, which 
determine how trace data is exported. For example, setting `OTEL_EXPORTER_OTLP_PROTOCOL` to `grpc` enables gRPC protocol
for exporting data.

## Header Propagation

The system uses [autoprop](https://pkg.go.dev/go.opentelemetry.io/contrib/propagators/autoprop) for header propagation, 
configured through the `OTEL_PROPAGATORS` environment variable. Supported propagators include tracecontext, baggage, b3,
and others. This configuration determines how trace context is maintained across different service calls.

## Redacting headers
OpenTelemetry might collect HTTP headers that contain sensitive information or PII which are usually not desirable to be
logged in the traces. By default, OpenTelemetry redacts the following headers: `Authorization, WWW-Authenticate, Proxy-Authenticate
Proxy-Authorization, Cookie, Set-Cookie`.
If desired, additional headers can be redacted in `config.yml`.
```yaml
target:
    redacted_headers:
        - Some-Private-Header
        - Some-Other-Private-Header
```

### Kubernetes Configuration Example

Below is an example configuration for Kubernetes, replace v0.11.0 with the version of Graphql Protect you are using.


```yaml
...
spec:
  template:
    spec:
      containers:
      - name: graphql-protect
        image: eu.gcr.io/bolcom-stg-shop-api-a1c/graphql-protect:latest # Replace with the appropriate version
        env:
          - name: OTEL_EXPORTER_OTLP_PROTOCOL
            value: grpc
          - name: OTEL_PROPAGATORS
            value: b3multi,tracecontext,baggage
...
```
