package schema

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"log/slog"
	"os"
	"sync"
	"time"
)

var reloadGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace:   "graphql_protect",
	Subsystem:   "schema",
	Name:        "reload",
	Help:        "Gauge tracking reloading behavior",
	ConstLabels: nil,
},
	[]string{"state"})

type LoaderConfig struct {
	Type     string `yaml:"type"`     // "local" (default) or "gcp"
	Location string `yaml:"location"` // GCS bucket name when type is "gcp"
}

type Config struct {
	Path       string       `yaml:"path"`
	Loader     LoaderConfig `yaml:"loader"`
	AutoReload struct {
		Enabled  bool          `yaml:"enabled"`
		Interval time.Duration `yaml:"interval"`
	} `yaml:"auto_reload"`
}

func DefaultConfig() Config {
	return Config{
		Path: "./schema.graphql",
		Loader: LoaderConfig{
			Type: "local",
		},
		AutoReload: struct {
			Enabled  bool          `yaml:"enabled"`
			Interval time.Duration `yaml:"interval"`
		}(struct {
			Enabled  bool
			Interval time.Duration
		}{
			Enabled:  true,
			Interval: 30 * time.Second,
		}),
	}
}

type Provider struct {
	cfg           Config
	mu            sync.RWMutex
	schema        *ast.Schema
	done          chan bool
	refreshTicker *time.Ticker
	log           *slog.Logger
	loadFn        func() error
}

func NewSchema(cfg Config, log *slog.Logger) (*Provider, error) {
	refreshTicker := func() *time.Ticker {
		if !cfg.AutoReload.Enabled {
			return nil
		}
		return time.NewTicker(cfg.AutoReload.Interval)
	}()

	p := &Provider{
		cfg: cfg,
		// nil until we load
		schema: nil,
		// buffered in case we don't have reloading enabled
		done:          make(chan bool, 1),
		refreshTicker: refreshTicker,
		log:           log,
	}

	switch cfg.Loader.Type {
	case "gcp":
		gcpLoader, err := newGcpSchemaLoader(cfg.Loader.Location, cfg.Path, log)
		if err != nil {
			return nil, fmt.Errorf("unable to create GCP schema loader: %w", err)
		}
		p.loadFn = func() error {
			contents, err := gcpLoader.fetch()
			if err != nil {
				return err
			}
			return p.load(contents)
		}
	default:
		p.loadFn = p.loadFromFs
	}

	if err := p.loadFn(); err != nil {
		return nil, fmt.Errorf("unable to load schema [%s]: %w", cfg.Path, err)
	}

	// initiate auto reloading
	p.reload()

	return p, nil
}

func (p *Provider) load(contents string) error {
	schema, err := gqlparser.LoadSchema(&ast.Source{
		Name:    "graph/schema.graphqls",
		Input:   contents,
		BuiltIn: false,
	})
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.schema = schema
	p.mu.Unlock()
	return nil
}

func (p *Provider) loadFromFs() error {
	contents, err := os.ReadFile(p.cfg.Path)
	if err != nil {
		return err
	}
	return p.load(string(contents))
}

func (p *Provider) Get() *ast.Schema {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.schema
}

func (p *Provider) reload() {
	if !p.cfg.AutoReload.Enabled {
		return
	}

	go func() {
		for {
			select {
			case <-p.done:
				return
			case <-p.refreshTicker.C:
				err := p.loadFn()
				if err != nil {
					p.log.Warn("Error reloading schema", "err", err)
					reloadGauge.WithLabelValues("failed").Inc()
					continue
				}
				reloadGauge.WithLabelValues("success").Inc()
			}
		}
	}()
}

func (p *Provider) Stop() {
	p.done <- true
}
