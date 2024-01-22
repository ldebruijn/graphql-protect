package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/ldebruijn/graphql-protect/internal/business/block_field_suggestions"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Timeout   time.Duration `conf:"default:10s" yaml:"timeout"`
	KeepAlive time.Duration `conf:"default:180s" yaml:"keepAlive"`
	Host      string        `conf:"default:http://localhost:8081" yaml:"host"`
}

func NewProxy(cfg Config, blockFieldSuggestions *block_field_suggestions.BlockFieldSuggestionsHandler) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.Timeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,
	}
	proxy.ModifyResponse = modifyResponse(blockFieldSuggestions) // nolint:bodyclose

	return proxy, nil
}

func modifyResponse(blockFieldSuggestions *block_field_suggestions.BlockFieldSuggestionsHandler) func(res *http.Response) error {
	return func(res *http.Response) error {
		if !blockFieldSuggestions.Enabled() {
			return nil
		}

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

		modified := blockFieldSuggestions.ProcessBody(response)
		bts, err := json.Marshal(modified)
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
