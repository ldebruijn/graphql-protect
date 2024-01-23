package persisted_operations // nolint:revive

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

import (
	"context"
)

var (
	persistedOpsHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "graphql_protect",
		Subsystem: "persisted_operations",
		Name:      "counter",
		Help:      "The results of the persisted operations rule",
	},
		[]string{"state", "result"},
	)
	reloadCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "graphql_protect",
		Subsystem:   "persisted_operations",
		Name:        "reload",
		Help:        "Counter tracking reloading behavior and results",
		ConstLabels: nil,
	},
		[]string{"system", "result"})
)

type ErrorPayload struct {
	Errors gqlerror.List `json:"errors"`
}

type ErrorMessage struct {
	Message string `json:"message"`
}

type Config struct {
	Enabled bool `conf:"default:true" yaml:"enabled"`
	// The location on which persisted operations are stored
	Store string `conf:"default:./store" yaml:"store"`
	// Configuration for auto-reloading persisted operations
	Reload struct {
		Enabled  bool          `conf:"default:true" yaml:"enabled"`
		Interval time.Duration `conf:"default:5m" yaml:"interval"`
		Timeout  time.Duration `conf:"default:10s" yaml:"timeout"`
	}
	// Remote strategies for fetching persisted operations
	Remote struct {
		GcpBucket string `conf:"default:your_bucket_name" yaml:"gcp_bucket"`
	}
	RejectOnFailure bool `conf:"default:true" yaml:"reject_on_failure"`
}

var ErrNoLoaderSupplied = errors.New("no remoteLoader supplied")
var ErrNoHashFound = errors.New("no hash found")
var ErrPersistedQueryNotFound = errors.New("PersistedQueryNotFound")
var ErrPersistedOperationNotFound = errors.New("PersistedOperationNotFound")
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
	prometheus.MustRegister(persistedOpsHistogram, reloadCounter)
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
func (p *PersistedOperationsHandler) Execute(next http.Handler) http.Handler { // nolint:funlen,cyclop
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !p.cfg.Enabled || r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		var errs gqlerror.List

		payload, err := gql.ParseRequestPayload(r)
		if err != nil {
			p.log.Warn("error decoding payload", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		for i, data := range payload {
			if !p.cfg.RejectOnFailure && data.Query != "" {
				persistedOpsHistogram.WithLabelValues("unknown", "allowed").Observe(time.Since(start).Seconds())
				continue
			}

			hash, err := hashFromPayload(data)
			if err != nil {
				persistedOpsHistogram.WithLabelValues("error", "rejected").Observe(time.Since(start).Seconds())
				errs = append(errs, gqlerror.Wrap(ErrPersistedQueryNotFound))
				continue
			}

			p.lock.RLock()
			query, ok := p.cache[hash]
			p.lock.RUnlock()

			if !ok {
				// hash not found, fail
				persistedOpsHistogram.WithLabelValues("unknown", "rejected").Observe(time.Since(start).Seconds())
				errs = append(errs, gqlerror.Wrap(ErrPersistedOperationNotFound))
				continue
			}

			// update the original data
			payload[i].Query = query
			payload[i].Extensions.PersistedQuery = nil

			persistedOpsHistogram.WithLabelValues("known", "allowed").Observe(time.Since(start).Seconds())
		}

		if len(errs) > 0 {
			// if any error occurred we fail
			res, _ := json.Marshal(ErrorPayload{
				Errors: errs,
			})
			http.Error(w, string(res), 200)
			return
		}

		var bts []byte
		// forward batched request
		if len(payload) > 1 {
			bts, err = json.Marshal(payload)
			if err != nil {
				// handle
				next.ServeHTTP(w, r)
				return
			}
		} else if len(payload) == 1 {
			// forward regular request
			bts, err = json.Marshal(payload[0])
			if err != nil {
				// handle
				next.ServeHTTP(w, r)
				return
			}
		}

		// overwrite request body with new payload
		r.Body = io.NopCloser(bytes.NewBuffer(bts))
		r.ContentLength = int64(len(bts))

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

func hashFromPayload(payload gql.RequestData) (string, error) {
	if payload.Extensions.PersistedQuery == nil {
		return "", ErrNoHashFound
	}

	hash := payload.Extensions.PersistedQuery.Sha256Hash
	if hash == "" {
		return "", ErrNoHashFound
	}

	return hash, nil
}
