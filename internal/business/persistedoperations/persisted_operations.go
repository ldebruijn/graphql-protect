package persistedoperations // nolint:revive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/ldebruijn/graphql-protect/internal/business/validation"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

var (
	persistedOpsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "persisted_operations",
		Name:      "result_count",
		Help:      "The results of the persisted operations rule",
	},
		[]string{"state", "result"},
	)
	loadingResultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "persisted_operations",
		Name:      "load_result_count",
		Help:      "Counter tracking loading behavior and results",
	},
		[]string{"type", "result"},
	)
	uniqueHashesInMemGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "graphql_protect",
		Subsystem: "persisted_operations",
		Name:      "unique_hashes_in_memory_count",
		Help:      "number of unique hashes in memory",
	}, []string{},
	)
)

type ErrorPayload struct {
	Errors gqlerror.List `json:"errors"`
}

type ErrorMessage struct {
	Message string `json:"message"`
}

type Config struct {
	Enabled         bool         `conf:"default:false" yaml:"enabled"`
	RejectOnFailure bool         `conf:"default:true" yaml:"reject_on_failure"`
	Loader          LoaderConfig `yaml:"loader"`
}

type LoaderConfig struct {
	Type     string `conf:"default:local" yaml:"type"`
	Location string `conf:"default:./store" yaml:"location"`
	// Configuration for auto-reloading persisted operations
	Reload struct {
		Enabled  bool          `conf:"default:true" yaml:"enabled"`
		Interval time.Duration `conf:"default:5m" yaml:"interval"`
		Timeout  time.Duration `conf:"default:10s" yaml:"timeout"`
	}
}

var ErrNoLoaderSupplied = errors.New("no remoteLoader supplied")
var ErrNoHashFound = errors.New("no hash found")
var ErrPersistedQueryNotFound = errors.New("PersistedQueryNotFound")
var ErrPersistedOperationNotFound = errors.New("PersistedOperationNotFound")
var ErrReloadIntervalTooShort = errors.New("load interval cannot be less than 10 seconds")

type Handler struct {
	log *slog.Logger
	cfg Config
	// this has the opportunity to grow indefinitely, might wat to replace with a fixed-cap cache
	// or something like an LRU with a TTL
	cache         map[string]PersistedOperation
	refreshTicker *time.Ticker
	refreshLock   sync.Mutex

	loader Loader
	done   chan bool
	lock   sync.RWMutex
}

func init() {
	prometheus.MustRegister(persistedOpsCounter, loadingResultCounter, uniqueHashesInMemGauge)
}

func NewPersistedOperations(log *slog.Logger, cfg Config, loader Loader) (*Handler, error) {
	if loader == nil {
		return nil, ErrNoLoaderSupplied
	}

	if cfg.Loader.Reload.Enabled && cfg.Loader.Reload.Interval < 10*time.Second {
		return nil, ErrReloadIntervalTooShort
	}

	refreshTicker := func() *time.Ticker {
		if !cfg.Loader.Reload.Enabled {
			return nil
		}
		return time.NewTicker(cfg.Loader.Reload.Interval)
	}()
	// buffered in case we don't have reloading enabled
	done := make(chan bool, 1)

	poh := &Handler{
		log:           log,
		cfg:           cfg,
		cache:         map[string]PersistedOperation{},
		loader:        loader,
		refreshTicker: refreshTicker,
		done:          done,
		lock:          sync.RWMutex{},
		refreshLock:   sync.Mutex{},
	}

	err := poh.load()
	if err != nil {
		return nil, err
	}

	poh.reloadProcessor()

	return poh, nil
}

// SwapHashForQuery runs of the persisted operations handler
// it uses the configuration supplied to decide its behavior
func (p *Handler) SwapHashForQuery(next http.Handler) http.Handler { // nolint:funlen,cyclop
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !p.cfg.Enabled || r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		var errs gqlerror.List

		payload, err := gql.ParseRequestPayload(r)
		if err != nil {
			p.log.Warn("error decoding payload", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		for i, data := range payload {
			if !p.cfg.RejectOnFailure && data.Query != "" {
				persistedOpsCounter.WithLabelValues("unknown", "allowed").Inc()
				continue
			}

			hash, err := hashFromPayload(data)
			if err != nil {
				persistedOpsCounter.WithLabelValues("error", "rejected").Inc()
				errs = append(errs, gqlerror.Wrap(ErrPersistedQueryNotFound))
				continue
			}

			p.lock.RLock()
			operation, ok := p.cache[hash]
			p.lock.RUnlock()

			if !ok {
				// hash not found, fail
				persistedOpsCounter.WithLabelValues("unknown", "rejected").Inc()
				errs = append(errs, gqlerror.Wrap(ErrPersistedOperationNotFound))
				continue
			}

			// update the original data
			payload[i].Query = operation.Operation
			payload[i].Extensions.PersistedQuery = nil
			payload[i].OperationName = operation.Name

			persistedOpsCounter.WithLabelValues("known", "allowed").Inc()
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

func (p *Handler) Validate(validate func(operation string) gqlerror.List) []validation.Error {
	var errs []validation.Error
	for hash, operation := range p.cache {
		err := validate(operation.Operation)

		if len(err) > 0 {
			valErr := validation.Wrap(err)
			valErr.Hash = hash
			valErr.Operation = operation.Operation
			errs = append(errs, valErr)
		}
	}

	return errs
}

func (p *Handler) load() error {
	newState, err := p.loader.Load(context.Background())
	if err != nil {
		loadingResultCounter.WithLabelValues(p.loader.Type(), "failure").Inc()
		return err
	}

	p.lock.Lock()
	p.cache = newState
	p.lock.Unlock()

	p.log.Info(fmt.Sprintf("Total number of unique operation hashes: %d", len(newState)))
	uniqueHashesInMemGauge.WithLabelValues().Set(float64(len(newState)))

	loadingResultCounter.WithLabelValues(p.loader.Type(), "success").Inc()

	return nil
}

func (p *Handler) reloadProcessor() {
	if !p.cfg.Loader.Reload.Enabled {
		return
	}

	go func() {
		for {
			select {
			case <-p.done:
				return
			case <-p.refreshTicker.C:
				if !p.refreshLock.TryLock() {
					p.log.Warn("Refresh ticker still running while next tick")
					continue
				}
				err := p.load()
				if err != nil {
					continue
				}
				p.refreshLock.Unlock()
			}
		}
	}()
}

func (p *Handler) Shutdown() {
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
