package schema

import (
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log/slog"
	"os"
	"time"
)

var reloadGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace:   "go_graphql_armor",
	Subsystem:   "schema",
	Name:        "reload",
	Help:        "Gauge tracking reloading behavior",
	ConstLabels: nil,
},
	[]string{"state"})

type Config struct {
	Path       string `conf:"default:./schema.graphql" yaml:"path"`
	AutoReload struct {
		Enabled  bool          `conf:"default:true" yaml:"enabled"`
		Interval time.Duration `conf:"default:30s" yaml:"interval"`
	}
}

type Provider struct {
	cfg           Config
	schema        *graphql.Schema
	done          chan bool
	refreshTicker *time.Ticker
	log           *slog.Logger
}

func NewSchema(cfg Config, log *slog.Logger) (*Provider, error) {
	refreshTicker := func() *time.Ticker {
		if !cfg.AutoReload.Enabled {
			return nil
		}
		return time.NewTicker(cfg.AutoReload.Interval)
	}()

	p := Provider{
		cfg: cfg,
		// nil until we load
		schema: nil,
		// buffered in case we don't have reloading enabled
		done:          make(chan bool, 1),
		refreshTicker: refreshTicker,
		log:           log,
	}

	err := p.loadFromFs()
	if err != nil {
		return nil, fmt.Errorf("unable to load schema from disk [%s]: %w", p.cfg.Path, err)
	}

	return &p, nil
}

type query struct{}

func (p *Provider) load(target io.Reader) error {
	var to []byte
	_, err := target.Read(to)
	if err != nil {
		return err
	}

	schema := graphql.MustParseSchema(string(to), &query{})

	p.schema = schema
	return nil
}

func (p *Provider) loadFromFs() error {
	open, err := os.Open(p.cfg.Path)
	if err != nil {
		p.log.Warn("error opening file", "err", err)
	}
	return p.load(open)
}

func (p *Provider) Get() *graphql.Schema {
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
				err := p.loadFromFs()
				if err != nil {
					p.log.Warn("Error loading from local dir", "err", err)
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
