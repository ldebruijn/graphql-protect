package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"regexp"
)

func init() {
	// Register metrics from GoCollector collecting statistics from the Go Runtime.
	// This enabled default, recommended metrics with the additional, recommended metric for
	// goroutine scheduling latencies histogram that is currently bit too expensive for the default option.
	//
	// See the related GopherConUK talk to learn more: https://www.youtube.com/watch?v=18dyI_8VFa0

	// Unregister the default GoCollector.
	prometheus.Unregister(collectors.NewGoCollector())

	// Register the default GoCollector with a custom config.
	prometheus.MustRegister(
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(
				collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/sched/latencies:seconds")},
			),
		),
	)
}
