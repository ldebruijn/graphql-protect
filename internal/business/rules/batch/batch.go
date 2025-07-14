package batch

import (
	"errors"
	"fmt"
	"github.com/ldebruijn/graphql-protect/internal/business/gql"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

var (
	resultCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "graphql_protect",
		Subsystem: "max_batch",
		Name:      "results",
		Help:      "The results of the max batch rule, including the size of the batch. The actual size is only tracked for allowed operations, to prevent excessive metric generation on malicious input",
	},
		[]string{"result", "size"},
	)
	ErrMaxBatchSizeTooSmall = errors.New("maximum allowed batch size cannot be smaller than 1. Protection auto-disabled")
)

type Config struct {
	Enabled         bool `yaml:"enabled"`
	Max             int  `yaml:"max"`
	RejectOnFailure bool `yaml:"reject_on_failure"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		Max:             1,
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
			resultCounter.WithLabelValues("violation-rejected", "exceeded").Inc()
			return fmt.Errorf("operation has exceeded maximum batch size. found [%d], max [%d]", len(payload), t.cfg.Max)
		} else {
			resultCounter.WithLabelValues("violation-allowed", "exceeded").Inc()
		}
		return nil
	}

	size := strconv.Itoa(len(payload))

	resultCounter.WithLabelValues("allowed", size).Inc()
	return nil
}
