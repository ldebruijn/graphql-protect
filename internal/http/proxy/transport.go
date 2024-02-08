package proxy

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net"
	"net/http"
	"net/http/httptrace"
)

func NewTransport(cfg Config) http.RoundTripper {
	return otelhttp.NewTransport(
		&http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   cfg.Timeout,
				KeepAlive: cfg.KeepAlive,
			}).DialContext,
		},
		otelhttp.WithSpanNameFormatter(spanNameFormatter),
		otelhttp.WithClientTrace(newClientTrace(cfg.Tracing)))
}

func spanNameFormatter(_ string, _ *http.Request) string {
	return "Proxy to target GraphQL Server"
}

func newClientTrace(conf TracingConfig) func(ctx context.Context) *httptrace.ClientTrace {
	return func(ctx context.Context) *httptrace.ClientTrace {
		return otelhttptrace.NewClientTrace(ctx, otelhttptrace.WithRedactedHeaders(conf.RedactedHeaders...))
	}
}
