package persisted_operations

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

type Payload struct {
	OperationName string      `json:"operationName"`
	Variables     interface{} `json:"variables"`
	Query         string      `json:"query"`
	Extensions    Extensions  `json:"extensions"`
}
type Extensions struct {
	PersistedQuery PersistedQuery `json:"persistedQuery"`
}
type PersistedQuery struct {
	Sha256Hash string `json:"sha256Hash"`
}

type Error struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type PersistedOperationsLoader interface {
	Load(ctx context.Context) (map[string]string, error)
}

type Config struct {
	Enabled bool `conf:"default:false"`
	Store   struct {
		Bucket string `conf:"gs://something/foo"`
		Dir    string `conf:""`
	}
	FailUnknownRequests      bool `conf:"default:true"`
	AllowUnPersistedRequests bool `conf:"default:true"`
}

type PersistedOperationsHandler struct {
	log *slog.Logger
	cfg Config
	// this has the opportunity to grow indefinitely, might wat to replace with a fixed-cap cache
	// or something like a LRU with a TTL
	cache  map[string]string
	loader PersistedOperationsLoader
}

func NewPersistedOperations(log *slog.Logger, cfg Config, loader PersistedOperationsLoader) (*PersistedOperationsHandler, error) {
	if cfg.Store.Dir == "" && cfg.Store.Bucket == "" {
		log.Warn("No store specified to load persisted operations from", "store", cfg.Store)
	}

	cache, err := loader.Load(context.Background())
	if err != nil {
		return nil, err
	}

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
		if !p.cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		// Replace the body with a new reader after reading from the original
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		var payload Payload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			p.log.Warn("error decoding payload", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		if p.cfg.AllowUnPersistedRequests && payload.Query != "" {
			next.ServeHTTP(w, r)
			return
		}

		hash := payload.Extensions.PersistedQuery.Sha256Hash
		if hash == "" {
			p.log.Warn("no hash present ", "err", err)
			res, _ := json.Marshal(buildError("PersistedQueryNotFound"))
			http.Error(w, string(res), 200)
			return
		}

		query, ok := p.cache[hash]
		if !ok {
			// load hash & put into cache
			p.log.Warn("Unknown hash, persisted operation not found ", "err", err)
			res, _ := json.Marshal(buildError("PersistedOperationNotFound"))
			http.Error(w, string(res), 200)
			return
		}

		payload.Query = query

		bts, err := json.Marshal(payload)
		if err != nil {
			// handle
			next.ServeHTTP(w, r)
			return
		}

		// overwrite request body with new payload
		r.Body = io.NopCloser(bytes.NewBuffer(bts))

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func buildError(message string) Error {
	return Error{
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
