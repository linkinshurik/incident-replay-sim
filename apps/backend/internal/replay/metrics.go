package replay

import "github.com/prometheus/client_golang/prometheus"

var (
	promReplayRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "replay_runs_total",
			Help: "Total number of replay runs started, labeled by mode.",
		},
		[]string{"mode"},
	)

	promReplayRunsFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "replay_runs_failed_total",
			Help: "Total number of replay runs that failed, labeled by mode and reason.",
		},
		[]string{"mode", "reason"},
	)

	promReplayRunDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "replay_run_duration_seconds",
			Help:    "Histogram of replay run durations in seconds, labeled by mode.",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"mode"},
	)

	promReplayRunsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "replay_active_runs",
			Help: "Current number of active replay runs.",
		},
	)
)

func init() {
	prometheus.MustRegister(promReplayRunsTotal)
	prometheus.MustRegister(promReplayRunsFailedTotal)
	prometheus.MustRegister(promReplayRunDurationSeconds)
	prometheus.MustRegister(promReplayRunsActive)
}
