package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/ldebruijn/go-graphql-armor/internal/business/block_field_suggestions"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Config struct {
	Timeout   time.Duration `conf:"default:10s" yaml:"timeout"`
	KeepAlive time.Duration `conf:"default:180s" yaml:"keepAlive"`
	Host      string        `conf:"default:http://localhost:8081" yaml:"host"`
	Path      string        `conf:"default:/graphql" yaml:"path"`
}

func NewProxy(cfg Config, blockFieldSuggestions *block_field_suggestions.BlockFieldSuggestionsHandler) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(cfg.Host)
	target.Path = cfg.Path
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
	proxy.ModifyResponse = func(res *http.Response) error {
		if !blockFieldSuggestions.Enabled() {
			return nil
		}

		decoder := json.NewDecoder(res.Body)

		var response map[string]interface{}
		err := decoder.Decode(&response)
		if err != nil {
			// if we cannot decode just return
			return nil
		}

		modified := blockFieldSuggestions.ProcessBody(response)
		bts, err := json.Marshal(modified)
		if err != nil {
			// if we cannot marshall just return
			return nil
		}

		buffer := bytes.NewBuffer(bts)
		res.Body = io.NopCloser(buffer)
		res.ContentLength = int64(buffer.Len())

		return nil
	}

	return proxy, nil
}
