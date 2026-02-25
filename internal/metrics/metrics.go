package metrics

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	BlocksProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "blocks_processed_total",
			Help: "Total number of blocks successfully processed by the indexer",
		},
	)

	CurrentBlockHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "current_block_height",
			Help: "The current block height the indexer has processed up to",
		},
	)

	ChainTipHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "chain_tip_height",
			Help: "The latest block height observed on the blockchain tip",
		},
	)

	RPCErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_errors_total",
			Help: "Total number of RPC errors encountered",
		},
		[]string{"type"}, // e.g., "timeout", "rate_limit", "unknown"
	)

	ReorgDetectedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "reorg_detected_total",
			Help: "Total number of chain re-organizations detected",
		},
	)

	LagEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "lag_events_total",
			Help: "Total number of times indexer lag exceeded threshold",
		},
	)

	BlockProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "block_processing_duration_seconds",
			Help:    "Histogram of block processing durations in seconds",
			Buckets: prometheus.DefBuckets, // Default buckets: 0.005s, 0.01s, 0.025s, ... 10s
		},
	)
)

// InitMetricsServer starts the Prometheus metrics HTTP server on the given address
func InitMetricsServer(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	slog.Info("Starting Prometheus metrics server", "addr", addr)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Error("Metrics server failed", "error", err)
		}
	}()
}
