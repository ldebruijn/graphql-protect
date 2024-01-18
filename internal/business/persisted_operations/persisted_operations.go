package persisted_operations // nolint:revive

import (
	"encoding/json"
	"errors"
	"github.com/ldebruijn/go-graphql-armor/internal/business/gql"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

import (
	"bytes"
	"context"
)

var (
	persistedOpsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go_graphql_armor",
		Subsystem: "persisted_operations",
		Name:      "counter",
		Help:      "The results of the persisted operations rule",
	},
		[]string{"state", "result"},
	)
	reloadCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "go_graphql_armor",
		Subsystem:   "persisted_operations",
		Name:        "reload",
		Help:        "Counter tracking reloading behavior and results",
		ConstLabels: nil,
	},
		[]string{"system", "result"})
)

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
		GcpBucket string `conf:"your_bucket_name" yaml:"gcp_bucket"`
	}
	FailUnknownOperations bool `conf:"default:false" yaml:"fail_unknown_operations"`
}

var ErrNoLoaderSupplied = errors.New("no remoteLoader supplied")
var ErrNoHashFound = errors.New("no hash found")
var ErrReloadIntervalTooShort = errors.New("reload interval cannot be less than 10 seconds")

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

func init() {
	prometheus.MustRegister(persistedOpsCounter, reloadCounter)
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

	if cfg.Reload.Enabled && cfg.Reload.Interval < 10*time.Second {
		return nil, ErrReloadIntervalTooShort
	}

	refreshTicker := func() *time.Ticker {
		if !cfg.Reload.Enabled {
			return nil
		}
		return time.NewTicker(cfg.Reload.Interval)
	}()
	// buffered in case we dont have reloading enabled
	done := make(chan bool, 1)

	poh := &PersistedOperationsHandler{
		log:           log,
		cfg:           cfg,
		cache:         map[string]string{},
		remoteLoader:  remoteLoader,
		dirLoader:     loader,
		refreshTicker: refreshTicker,
		done:          done,
		lock:          sync.RWMutex{},
	}

	if cfg.Enabled {
		poh.reloadFromRemote()
		err := poh.reloadFromLocalDir()
		if err != nil {
			return nil, err
		}

		// start reloader
		poh.reload()
	}

	return poh, nil
}

// Execute runs of the persisted operations handler
// it uses the configuration supplied to decide its behavior
func (p *PersistedOperationsHandler) Execute(next http.Handler) http.Handler { // nolint:funlen
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !p.cfg.Enabled || r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		payload, err := gql.ParseRequestPayload(r)
		if err != nil {
			p.log.Warn("error decoding payload", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		if !p.cfg.FailUnknownOperations && payload.Query != "" {
			persistedOpsCounter.WithLabelValues("unknown", "allowed").Inc()
			next.ServeHTTP(w, r)
			return
		}

		hash, err := hashFromPayload(payload)
		if err != nil {
			persistedOpsCounter.WithLabelValues("unknown", "rejected").Inc()
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
			persistedOpsCounter.WithLabelValues("unknown", "rejected").Inc()
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
			persistedOpsCounter.WithLabelValues("errored", "allowed").Inc()
			next.ServeHTTP(w, r)
			return
		}

		// overwrite request body with new payload
		r.Body = io.NopCloser(bytes.NewBuffer(bts))
		r.ContentLength = int64(len(bts))

		persistedOpsCounter.WithLabelValues("known", "allowed").Inc()

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (p *PersistedOperationsHandler) reloadFromLocalDir() error {
	cache, err := p.dirLoader.Load(context.Background())
	if err != nil {
		reloadCounter.WithLabelValues("local", "failure").Inc()
		return err
	}
	p.lock.Lock()
	p.cache = cache
	p.lock.Unlock()

	p.log.Info("Loaded persisted operations", "amount", len(cache))
	reloadCounter.WithLabelValues("local", "success").Inc()

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
			case <-p.refreshTicker.C:
				p.reloadFromRemote()
				err := p.reloadFromLocalDir()
				if err != nil {
					p.log.Warn("Error loading from local dir", "err", err)
					reloadCounter.WithLabelValues("ticker", "failure").Inc()
					continue
				}
				reloadCounter.WithLabelValues("ticker", "success").Inc()
			}
		}
	}()
}

func (p *PersistedOperationsHandler) reloadFromRemote() {
	if p.remoteLoader == nil {
		return
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, p.cfg.Reload.Timeout)
	defer cancel()

	err := p.remoteLoader.Load(ctx)
	if err != nil {
		reloadCounter.WithLabelValues("remote", "failure").Inc()
		return
	}

	reloadCounter.WithLabelValues("remote", "success").Inc()
}

func (p *PersistedOperationsHandler) Shutdown() {
	p.done <- true
}

func hashFromPayload(payload gql.RequestPayload) (string, error) {
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
