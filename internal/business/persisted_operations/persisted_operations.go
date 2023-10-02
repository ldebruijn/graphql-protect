package persisted_operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
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

type Config struct {
	Enabled bool `conf:"default:false" yaml:"enabled"`
	// The location on which persisted operations are stored
	Store string `conf:"./store" yaml:"store"`
	// Configuration for auto-reloading persisted operations
	Reload struct {
		Enabled  bool          `conf:"default:false" yaml:"enabled"`
		Interval time.Duration `conf:"default:5m" yaml:"interval"`
		Timeout  time.Duration `conf:"default:10s" yaml:"timeout"`
	}
	// Remote strategies for fetching persisted operations
	Remote struct {
		GcpBucket string `conf:"gs://something/foo" yaml:"gcp_bucket"`
	}
	FailUnknownOperations bool `conf:"default:false" yaml:"fail_unknown_operations"`
}

var ErrNoLoaderSupplied = errors.New("no remoteLoader supplied")
var ErrNoHashFound = errors.New("no hash found")

type PersistedOperationsHandler struct {
	log *slog.Logger
	cfg Config
	// this has the opportunity to grow indefinitely, might wat to replace with a fixed-cap cache
	// or something like an LRU with a TTL
	cache map[string]string
	// Strategy for loading persisted operations from a remote location
	remoteLoader  RemoteLoader
	refreshTicker *time.Ticker

	dirLoader LocalLoader
	done      chan bool
	lock      sync.RWMutex
}

func NewPersistedOperations(log *slog.Logger, cfg Config, loader LocalLoader, remoteLoader RemoteLoader) (*PersistedOperationsHandler, error) {
	if loader == nil {
		return nil, ErrNoLoaderSupplied
	}

	if remoteLoader != nil {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, cfg.Reload.Timeout)
		defer cancel()
		err := remoteLoader.Load(ctx)
		if err != nil {
			return nil, err
		}
	}

	if cfg.Reload.Interval < 10*time.Second {
		cfg.Reload.Interval = 10 * time.Second
		log.Warn("Reload interval cannot be less than every 10 seconds, manually overwrote to 10 seconds")
	}

	refreshTicker := time.NewTicker(cfg.Reload.Interval)
	done := make(chan bool)

	cache, err := loader.Load(context.Background())
	if err != nil {
		return nil, err
	}

	log.Info("Loaded persisted operations", "amount", len(cache))

	poh := &PersistedOperationsHandler{
		log:           log,
		cfg:           cfg,
		cache:         cache,
		remoteLoader:  remoteLoader,
		dirLoader:     loader,
		refreshTicker: refreshTicker,
		done:          done,
		lock:          sync.RWMutex{},
	}

	// start reloader
	poh.reload()

	return poh, nil
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

		if !p.cfg.FailUnknownOperations && payload.Query != "" {
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

		p.lock.RLock()
		query, ok := p.cache[hash]
		p.lock.RUnlock()

		if !ok {
			// hash not found, fail
			p.log.Warn("Unknown hash, persisted operation not found ", "hash", hash)
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

func (p *PersistedOperationsHandler) reloadFromLocalDir() error {
	dirLoader := NewLocalDirLoader(p.cfg)
	if dirLoader == nil {
		return errors.New("dir loader is nil")
	}

	cache, err := dirLoader.Load(context.Background())
	if err != nil {
		return err
	}
	p.lock.Lock()
	p.cache = cache
	p.lock.Unlock()

	p.log.Info("Loaded persisted operations", "amount", len(cache))

	return nil
}

func (p *PersistedOperationsHandler) reload() {
	if !p.cfg.Reload.Enabled {
		return
	}

	go func() {
		for {
			select {
			case <-p.done:
				return
			case _ = <-p.refreshTicker.C:
				p.reloadFromRemote()
			}
		}
	}()
}

func (p *PersistedOperationsHandler) reloadFromRemote() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, p.cfg.Reload.Timeout)
	err := p.remoteLoader.Load(ctx)
	cancel()
	if err != nil {
		return
	}

	err = p.reloadFromLocalDir()
	if err != nil {
		p.log.Error("Error loading from local dir", "err", err)
		return
	}
}

func (p *PersistedOperationsHandler) Shutdown() {
	p.done <- true
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
