package batch

import (
	"errors"
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "max_batch",
		Name:      "results",
		Help:      "The results of the max batch rule",
	},
		[]string{"result"},
	)
	ErrMaxBatchSizeTooSmall = errors.New("maximum allowed batch size cannot be smaller than 1. Protection auto-disabled")
)

type Config struct {
	Enabled         bool `conf:"default:true" yaml:"enabled"`
	Max             int  `conf:"default:5" yaml:"max"`
	RejectOnFailure bool `conf:"default:true" yaml:"reject_on_failure"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		Max:             5,
		RejectOnFailure: true,
	}
}

func init() {
	prometheus.MustRegister(resultCounter)
}

type MaxBatchRule struct {
	cfg Config
}

func NewMaxBatch(cfg Config) (*MaxBatchRule, error) {
	if cfg.Max < 1 {
		return &MaxBatchRule{
			cfg: Config{
				Enabled: false,
			},
		}, ErrMaxBatchSizeTooSmall
	}

	return &MaxBatchRule{
		cfg: cfg,
	}, nil
}

func (t *MaxBatchRule) Validate(payload []gql.RequestData) error {
	if !t.cfg.Enabled {
		return nil
	}

	if len(payload) > t.cfg.Max {
		if t.cfg.RejectOnFailure {
			resultCounter.WithLabelValues("rejected").Inc()
			return fmt.Errorf("operation has exceeded maximum batch size. found [%d], max [%d]", len(payload), t.cfg.Max)
		}
		resultCounter.WithLabelValues("failed").Inc()
	}
	resultCounter.WithLabelValues("allowed").Inc()
	return nil
}
