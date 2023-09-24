package persisted_operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type RequestPayload struct {
	//OperationName string      `json:"operationName"`
	Variables  interface{} `json:"variables"`
	Query      string      `json:"query"`
	Extensions Extensions  `json:"extensions"`
}
type Extensions struct {
	PersistedQuery *PersistedQuery `json:"persistedQuery"`
}
type PersistedQuery struct {
	Sha256Hash string `json:"sha256Hash"`
}

type ErrorPayload struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type PersistedOperationsLoader interface {
	Load(ctx context.Context) (map[string]string, error)
}

type Config struct {
	Enabled bool `conf:"default:false" yaml:"enabled"`
	Store   struct {
		GcpBucket string `conf:"gs://something/foo" yaml:"gcp_bucket"`
		Dir       string `conf:"" yaml:"dir"`
	}
	FailUnknownOperations      bool `conf:"default:true" yaml:"fail_unknown_operations"`
	AllowUnPersistedOperations bool `conf:"default:false" yaml:"allow_unpersisted_operations"`
}

var ErrNoLoaderSupplied = errors.New("no loader supplied")
var ErrNoHashFound = errors.New("no hash found")

type PersistedOperationsHandler struct {
	log *slog.Logger
	cfg Config
	// this has the opportunity to grow indefinitely, might wat to replace with a fixed-cap cache
	// or something like an LRU with a TTL
	cache map[string]string
	// not sure if keeping a reference to this is required, might be nice for refreshing during runtime
	loader PersistedOperationsLoader
}

func NewPersistedOperations(log *slog.Logger, cfg Config, loader PersistedOperationsLoader) (*PersistedOperationsHandler, error) {
	if loader == nil {
		return nil, ErrNoLoaderSupplied
	}

	cache, err := loader.Load(context.Background())
	if err != nil {
		return nil, err
	}

	log.Info("Loaded persisted operations", "amount", len(cache))

	return &PersistedOperationsHandler{
		log:    log,
		cfg:    cfg,
		cache:  cache,
		loader: loader,
	}, nil
}

// Execute runs of the persisted operations handler
// it uses the configuration supplied to decide its behavior
func (p *PersistedOperationsHandler) Execute(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !p.cfg.Enabled || r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		// Replace the body with a new reader after reading from the original
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		var payload RequestPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			p.log.Warn("error decoding payload", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		if p.cfg.AllowUnPersistedOperations && payload.Query != "" {
			next.ServeHTTP(w, r)
			return
		}

		hash, err := hashFromPayload(payload)
		if err != nil {
			p.log.Warn("no hash found ", "err", err)
			res, _ := json.Marshal(buildErrorResponse("PersistedQueryNotFound"))
			http.Error(w, string(res), 200)
			return
		}

		query, ok := p.cache[hash]
		if !ok {
			// hash not found, fail
			p.log.Warn("Unknown hash, persisted operation not found ", "err", err)
			res, _ := json.Marshal(buildErrorResponse("PersistedOperationNotFound"))
			http.Error(w, string(res), 200)
			return
		}

		payload.Query = query
		payload.Extensions.PersistedQuery = nil

		bts, err := json.Marshal(payload)
		if err != nil {
			// handle
			next.ServeHTTP(w, r)
			return
		}

		// overwrite request body with new payload
		r.Body = io.NopCloser(bytes.NewBuffer(bts))
		r.ContentLength = int64(len(bts))

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func hashFromPayload(payload RequestPayload) (string, error) {
	if payload.Extensions.PersistedQuery == nil {
		return "", ErrNoHashFound
	}

	hash := payload.Extensions.PersistedQuery.Sha256Hash
	if hash == "" {
		return "", ErrNoHashFound
	}

	return hash, nil
}

func buildErrorResponse(message string) ErrorPayload {
	return ErrorPayload{
		Errors: []struct {
			Message string `json:"message"`
		}([]struct {
			Message string
		}{
			{
				Message: message,
			},
		}),
	}
}
