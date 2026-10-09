package daily_action

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// GenerationDuration tracks daily action generation pipeline duration.
	GenerationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "daily_actions_generation_duration_seconds",
		Help:    "Duration of daily action generation pipeline",
		Buckets: prometheus.ExponentialBuckets(0.5, 2, 10),
	})

	// GenerationActionCount tracks the number of actions generated per run.
	GenerationActionCount = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "daily_actions_generation_action_count",
		Help:    "Number of actions generated per generation",
		Buckets: []float64{5, 10, 25, 50, 100, 200, 500},
	})

	// TransitionsTotal counts state transitions by type.
	TransitionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "daily_actions_transitions_total",
		Help: "Total state transitions by type",
	}, []string{"transition"})

	// SSEConnectionsActive tracks active SSE connections.
	SSEConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "daily_actions_sse_connections_active",
		Help: "Number of active SSE connections",
	})

	// HTTPRequestDuration tracks HTTP request duration by endpoint.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_daily_actions_request_duration_seconds",
		Help:    "Duration of daily actions HTTP requests by endpoint",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint"})

	// ChatTTFT tracks chat time-to-first-token.
	ChatTTFT = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "daily_actions_chat_ttft_seconds",
		Help:    "Time to first token for chat responses",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 8),
	})
)
