package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/block_field_suggestions"
	"github.com/ldebruijn/graphql-protect/internal/business/rules/obfuscate_upstream_errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Timeout   time.Duration `conf:"default:10s" yaml:"timeout"`
	KeepAlive time.Duration `conf:"default:180s" yaml:"keep_alive"`
	Host      string        `conf:"default:http://localhost:8081" yaml:"host"`
	Tracing   TracingConfig `yaml:"tracing"`
}

type TracingConfig struct {
	RedactedHeaders []string `yaml:"redacted_headers"`
}

func NewProxy(cfg Config, blockFieldSuggestions *block_field_suggestions.BlockFieldSuggestionsHandler, obfuscateUpstreamErrors *obfuscate_upstream_errors.ObfuscateUpstreamErrors, logGraphqlErrors bool, log *slog.Logger) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.Header["X-Forwarded-For"] = r.In.Header["X-Forwarded-For"]
			r.Out.Header.Del("Accept-Encoding") // Disabled as compression has no direct benefit for us within our cloud setup, this can be removed if proper parsing for all types of compression is implemented
			r.SetXForwarded()
			r.SetURL(target)
			r.Out.Host = r.In.Host
		},
		Transport:      NewTransport(cfg),
		ModifyResponse: modifyResponse(blockFieldSuggestions, obfuscateUpstreamErrors, logGraphqlErrors, log), // nolint:bodyclose
	}

	return proxy, nil
}

func modifyResponse(blockFieldSuggestions *block_field_suggestions.BlockFieldSuggestionsHandler, obfuscateUpstreamErrors *obfuscate_upstream_errors.ObfuscateUpstreamErrors, logGraphqlErrors bool, log *slog.Logger) func(res *http.Response) error {
	return func(res *http.Response) error {

		// read raw response bytes
		bodyBytes, _ := io.ReadAll(res.Body)
		defer res.Body.Close()

		var response map[string]interface{}
		err := json.Unmarshal(bodyBytes, &response)
		if err != nil {
			// if we cannot decode just return
			// make sure to set body back to original bytes
			res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			return nil
		}

		if logGraphqlErrors {
			log.Info("Error occurred at", "error", response["errors"])
		}

		if blockFieldSuggestions != nil && blockFieldSuggestions.Enabled() {
			response = blockFieldSuggestions.ProcessBody(response)
		}

		if obfuscateUpstreamErrors != nil && obfuscateUpstreamErrors.Enabled() {
			response = obfuscateUpstreamErrors.ProcessBody(response)
		}

		bts, err := json.Marshal(response)
		if err != nil {
			// if we cannot marshall just return
			// make sure to set body back to original bytes
			res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			return nil
		}

		buffer := bytes.NewBuffer(bts)
		res.ContentLength = int64(buffer.Len())
		res.Header.Set("Content-Length", strconv.Itoa(buffer.Len()))
		res.Body = io.NopCloser(buffer)

		return nil
	}
}
