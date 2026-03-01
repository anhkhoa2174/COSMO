package metrics

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/connectivity"
)

var (
	// HTTP request metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpErrorsTotal     *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Application metrics
	activeConnections prometheus.Gauge
	connectedUsers    prometheus.Gauge

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec
	dbQueriesTotal      *prometheus.CounterVec

	// Cache metrics
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec

	// Worker/Job metrics
	jobsTotal   *prometheus.CounterVec
	jobDuration *prometheus.HistogramVec
	activeJobs  *prometheus.GaugeVec

	// Business metrics
	campaignsActive   prometheus.Gauge
	emailsSent        *prometheus.CounterVec
	contactsProcessed *prometheus.CounterVec

	// AI/LLM metrics
	aiRequestsTotal   *prometheus.CounterVec
	aiRequestDuration *prometheus.HistogramVec
	aiTokensUsed      *prometheus.CounterVec

	// Go runtime metrics
	// Note: go_goroutines and other basic metrics are already collected by Prometheus
	goAppGCDuration *prometheus.HistogramVec
	goAppMemstats   *prometheus.GaugeVec
	goAppInfo       *prometheus.GaugeVec

	// gRPC client metrics
	grpcClientConnections *prometheus.GaugeVec

	// Collector management
	metricsCollectorCancel context.CancelFunc
	initMetricsOnce        sync.Once
)

var (
	collectorCtx context.Context
)

// Initialize all metrics
func init() {
	initMetricsOnce.Do(func() {
		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		)

		httpErrorsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "path", "status_code"},
		)

		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status_code"},
		)

		httpRequestSize = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		)

		httpResponseSize = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "status_code"},
		)

		// Application metrics
		activeConnections = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active connections",
			},
		)

		connectedUsers = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "connected_users",
				Help: "Number of connected users",
			},
		)

		// Database metrics
		dbConnectionsActive = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		)

		dbConnectionsIdle = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
			},
		)

		dbQueryDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"query_type", "table"},
		)

		dbQueriesTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"query_type", "table", "status"},
		)

		// Cache metrics
		cacheHits = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"},
		)

		cacheMisses = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		)

		// Worker/Job metrics
		jobsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_total",
				Help: "Total number of jobs processed",
			},
			[]string{"job_type", "status"},
		)

		jobDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "job_duration_seconds",
				Help:    "Job processing duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},
			[]string{"job_type"},
		)

		activeJobs = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "active_jobs",
				Help: "Number of active jobs",
			},
			[]string{"job_type"},
		)

		// Business metrics
		campaignsActive = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "campaigns_active",
				Help: "Number of active campaigns",
			},
		)

		emailsSent = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "emails_sent_total",
				Help: "Total number of emails sent",
			},
			[]string{"campaign_id", "status"},
		)

		contactsProcessed = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "contacts_processed_total",
				Help: "Total number of contacts processed",
			},
			[]string{"operation", "status"},
		)

		// AI/LLM metrics
		aiRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_requests_total",
				Help: "Total number of AI requests",
			},
			[]string{"provider", "model", "operation"},
		)

		aiRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ai_request_duration_seconds",
				Help:    "AI request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.5, 2, 8),
			},
			[]string{"provider", "model", "operation"},
		)

		aiTokensUsed = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_tokens_used_total",
				Help: "Total number of AI tokens used",
			},
			[]string{"provider", "model", "type"},
		)

		// Go runtime metrics
		// Note: Basic Go metrics like go_goroutines are automatically collected by Prometheus

		goAppGCDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "app_go_gc_duration_seconds",
				Help:    "Application-specific GC duration histogram",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 10, 100},
			},
			[]string{},
		)

		goAppMemstats = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_go_memstats",
				Help: "Application-specific Go memory statistics",
			},
			[]string{"type"},
		)

		goAppInfo = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_go_info",
				Help: "Application-specific Go environment information",
			},
			[]string{"version"},
		)

		grpcClientConnections = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "grpc_client_connections",
				Help: "Current state of gRPC client connections",
			},
			[]string{"grpc_service", "grpc_method", "state"},
		)

		// Register all metrics with default registry
		prometheus.MustRegister(httpRequestsTotal)
		prometheus.MustRegister(httpErrorsTotal)
		prometheus.MustRegister(httpRequestDuration)
		prometheus.MustRegister(httpRequestSize)
		prometheus.MustRegister(httpResponseSize)
		prometheus.MustRegister(activeConnections)
		prometheus.MustRegister(connectedUsers)
		prometheus.MustRegister(dbConnectionsActive)
		prometheus.MustRegister(dbConnectionsIdle)
		prometheus.MustRegister(dbQueryDuration)
		prometheus.MustRegister(dbQueriesTotal)
		prometheus.MustRegister(cacheHits)
		prometheus.MustRegister(cacheMisses)
		prometheus.MustRegister(jobsTotal)
		prometheus.MustRegister(jobDuration)
		prometheus.MustRegister(activeJobs)
		prometheus.MustRegister(campaignsActive)
		prometheus.MustRegister(emailsSent)
		prometheus.MustRegister(contactsProcessed)
		prometheus.MustRegister(aiRequestsTotal)
		prometheus.MustRegister(aiRequestDuration)
		prometheus.MustRegister(aiTokensUsed)
		prometheus.MustRegister(goAppGCDuration)
		prometheus.MustRegister(goAppMemstats)
		prometheus.MustRegister(goAppInfo)
		prometheus.MustRegister(grpcClientConnections)

		// Start Go metrics collector
		collectorCtx, metricsCollectorCancel = context.WithCancel(context.Background())
		go startGoMetricsCollector(collectorCtx)
	})
}

// Metrics groups related metrics for easier management
type Metrics struct {
	HTTP struct {
		RequestsTotal     *prometheus.CounterVec
		RequestDuration   *prometheus.HistogramVec
		RequestSize       *prometheus.HistogramVec
		ResponseSize      *prometheus.HistogramVec
		ActiveConnections prometheus.Gauge
		ConnectedUsers    prometheus.Gauge
	}
	DB struct {
		ConnectionsActive prometheus.Gauge
		ConnectionsIdle   prometheus.Gauge
		QueryDuration     *prometheus.HistogramVec
		QueriesTotal      *prometheus.CounterVec
	}
	Cache struct {
		Hits   *prometheus.CounterVec
		Misses *prometheus.CounterVec
	}
	Worker struct {
		JobsTotal   *prometheus.CounterVec
		JobDuration *prometheus.HistogramVec
		ActiveJobs  *prometheus.GaugeVec
	}
	Business struct {
		CampaignsActive   prometheus.Gauge
		EmailsSent        *prometheus.CounterVec
		ContactsProcessed *prometheus.CounterVec
	}
	AI struct {
		RequestsTotal   *prometheus.CounterVec
		RequestDuration *prometheus.HistogramVec
		TokensUsed      *prometheus.CounterVec
	}
	Go struct {
		GCDuration *prometheus.HistogramVec
		Memstats   *prometheus.GaugeVec
		Info       *prometheus.GaugeVec
	}
	GRPC struct {
		ClientConnections *prometheus.GaugeVec
	}
}

// Stop gracefully stops the metrics collector
func (m *Metrics) Stop() {
	StopMetricsCollector()
}

//Metrics
//abc

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	m := &Metrics{
		HTTP: struct {
			RequestsTotal     *prometheus.CounterVec
			RequestDuration   *prometheus.HistogramVec
			RequestSize       *prometheus.HistogramVec
			ResponseSize      *prometheus.HistogramVec
			ActiveConnections prometheus.Gauge
			ConnectedUsers    prometheus.Gauge
		}{
			RequestsTotal:     httpRequestsTotal,
			RequestDuration:   httpRequestDuration,
			RequestSize:       httpRequestSize,
			ResponseSize:      httpResponseSize,
			ActiveConnections: activeConnections,
			ConnectedUsers:    connectedUsers,
		},
		DB: struct {
			ConnectionsActive prometheus.Gauge
			ConnectionsIdle   prometheus.Gauge
			QueryDuration     *prometheus.HistogramVec
			QueriesTotal      *prometheus.CounterVec
		}{
			ConnectionsActive: dbConnectionsActive,
			ConnectionsIdle:   dbConnectionsIdle,
			QueryDuration:     dbQueryDuration,
			QueriesTotal:      dbQueriesTotal,
		},
		Cache: struct {
			Hits   *prometheus.CounterVec
			Misses *prometheus.CounterVec
		}{
			Hits:   cacheHits,
			Misses: cacheMisses,
		},
		Worker: struct {
			JobsTotal   *prometheus.CounterVec
			JobDuration *prometheus.HistogramVec
			ActiveJobs  *prometheus.GaugeVec
		}{
			JobsTotal:   jobsTotal,
			JobDuration: jobDuration,
			ActiveJobs:  activeJobs,
		},
		Business: struct {
			CampaignsActive   prometheus.Gauge
			EmailsSent        *prometheus.CounterVec
			ContactsProcessed *prometheus.CounterVec
		}{
			CampaignsActive:   campaignsActive,
			EmailsSent:        emailsSent,
			ContactsProcessed: contactsProcessed,
		},
		AI: struct {
			RequestsTotal   *prometheus.CounterVec
			RequestDuration *prometheus.HistogramVec
			TokensUsed      *prometheus.CounterVec
		}{
			RequestsTotal:   aiRequestsTotal,
			RequestDuration: aiRequestDuration,
			TokensUsed:      aiTokensUsed,
		},
		Go: struct {
			GCDuration *prometheus.HistogramVec
			Memstats   *prometheus.GaugeVec
			Info       *prometheus.GaugeVec
		}{
			GCDuration: goAppGCDuration,
			Memstats:   goAppMemstats,
			Info:       goAppInfo,
		},
		GRPC: struct {
			ClientConnections *prometheus.GaugeVec
		}{
			ClientConnections: grpcClientConnections,
		},
	}

	// Add method to stop collector
	return m
}

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// ObserveHTTPRequest records HTTP request metrics
func ObserveHTTPRequest(method, path, statusCode string, duration time.Duration, requestSize, responseSize int64) {
	httpRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	httpRequestDuration.WithLabelValues(method, path, statusCode).Observe(duration.Seconds())

	// Increment error counter for 4xx and 5xx status codes
	if statusCode[0] == '4' || statusCode[0] == '5' {
		httpErrorsTotal.WithLabelValues(method, path, statusCode).Inc()
	}

	if requestSize > 0 {
		httpRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		httpResponseSize.WithLabelValues(method, path, statusCode).Observe(float64(responseSize))
	}
}

// IncrementActiveConnections increments the active connections counter
func IncrementActiveConnections() {
	activeConnections.Inc()
}

// DecrementActiveConnections decrements the active connections counter
func DecrementActiveConnections() {
	activeConnections.Dec()
}

// SetActiveConnections sets the active connections to a specific value
func SetActiveConnections(n float64) {
	activeConnections.Set(n)
}

// IncrementConnectedUsers increments the connected users counter
func IncrementConnectedUsers() {
	connectedUsers.Inc()
}

// DecrementConnectedUsers decrements the connected users counter
func DecrementConnectedUsers() {
	connectedUsers.Dec()
}

// SetConnectedUsers sets the number of connected users
func SetConnectedUsers(n float64) {
	connectedUsers.Set(n)
}

// RecordDBQuery records database query metrics
func RecordDBQuery(queryType, table string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	dbQueryDuration.WithLabelValues(queryType, table).Observe(duration.Seconds())
	dbQueriesTotal.WithLabelValues(queryType, table, status).Inc()
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheType string) {
	cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheType string) {
	cacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordJob records job processing metrics
func RecordJob(jobType, status string, duration time.Duration) {
	jobsTotal.WithLabelValues(jobType, status).Inc()
	jobDuration.WithLabelValues(jobType).Observe(duration.Seconds())
}

// SetActiveJobs sets the number of active jobs for a job type
func SetActiveJobs(jobType string, count float64) {
	activeJobs.WithLabelValues(jobType).Set(count)
}

// RecordEmailSent records an email sent event
func RecordEmailSent(campaignID, status string) {
	emailsSent.WithLabelValues(campaignID, status).Inc()
}

// RecordContactProcessed records a contact processing event
func RecordContactProcessed(operation, status string) {
	contactsProcessed.WithLabelValues(operation, status).Inc()
}

// RecordAIRequest records an AI request
func RecordAIRequest(provider, model, operation string, duration time.Duration) {
	aiRequestsTotal.WithLabelValues(provider, model, operation).Inc()
	aiRequestDuration.WithLabelValues(provider, model, operation).Observe(duration.Seconds())
}

// RecordTokensUsed records AI token usage
func RecordTokensUsed(provider, model, tokenType string, count float64) {
	aiTokensUsed.WithLabelValues(provider, model, tokenType).Add(count)
}

// startGoMetricsCollector starts a goroutine that collects Go runtime metrics
// It respects context cancellation for graceful shutdown
func startGoMetricsCollector(ctx context.Context) {
	// Set Go version info
	goAppInfo.WithLabelValues(runtime.Version()).Set(1)

	var lastNumGC uint32
	runtime.ReadMemStats(&runtime.MemStats{}) // Just to get initial state
	lastNumGC = 0

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, exit gracefully
			return
		case <-ticker.C:
			// Note: go_goroutines is automatically collected by Prometheus

			// Read memory stats
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Update memory statistics
			goAppMemstats.WithLabelValues("alloc_bytes").Set(float64(m.Alloc))
			goAppMemstats.WithLabelValues("alloc_bytes_total").Set(float64(m.TotalAlloc))
			goAppMemstats.WithLabelValues("sys_bytes").Set(float64(m.Sys))
			goAppMemstats.WithLabelValues("heap_alloc_bytes").Set(float64(m.HeapAlloc))
			goAppMemstats.WithLabelValues("heap_sys_bytes").Set(float64(m.HeapSys))
			goAppMemstats.WithLabelValues("heap_idle_bytes").Set(float64(m.HeapIdle))
			goAppMemstats.WithLabelValues("heap_inuse_bytes").Set(float64(m.HeapInuse))
			goAppMemstats.WithLabelValues("heap_released_bytes").Set(float64(m.HeapReleased))
			goAppMemstats.WithLabelValues("heap_objects").Set(float64(m.HeapObjects))
			goAppMemstats.WithLabelValues("stack_inuse_bytes").Set(float64(m.StackInuse))
			goAppMemstats.WithLabelValues("stack_sys_bytes").Set(float64(m.StackSys))
			goAppMemstats.WithLabelValues("gc_cpu_fraction").Set(m.GCCPUFraction)
			goAppMemstats.WithLabelValues("num_gc").Set(float64(m.NumGC))
			goAppMemstats.WithLabelValues("num_forced_gc").Set(float64(m.NumForcedGC))
			goAppMemstats.WithLabelValues("gc_pause_total_ns").Set(float64(m.PauseTotalNs))
			goAppMemstats.WithLabelValues("num_go_routines").Set(float64(runtime.NumGoroutine()))

			// Observe GC pause duration if GC has run
			// We check if NumGC has increased since last check
			if lastNumGC != m.NumGC && m.NumGC > 0 {
				// The PauseNs array is circular, with index 256 as most recent
				// Index for the most recent GC pause
				recentGCIndex := int(m.NumGC+255) % 256
				if recentGCIndex < 256 {
					pauseNs := m.PauseNs[recentGCIndex]
					goAppGCDuration.WithLabelValues().Observe(float64(pauseNs) / 1e9) // Convert nanoseconds to seconds
				}
				lastNumGC = m.NumGC
			}
		}
	}
}

// StopMetricsCollector gracefully stops the metrics collector
func StopMetricsCollector() {
	if metricsCollectorCancel != nil {
		metricsCollectorCancel()
		metricsCollectorCancel = nil
	}
}

// RecordGRPCConnection records the state of a gRPC connection
func RecordGRPCConnection(grpcService, grpcMethod string, state connectivity.State) {
	stateStr := "UNKNOWN"
	switch state {
	case connectivity.Idle:
		stateStr = "IDLE"
	case connectivity.Connecting:
		stateStr = "CONNECTING"
	case connectivity.Ready:
		stateStr = "READY"
	case connectivity.TransientFailure:
		stateStr = "TRANSIENT_FAILURE"
	case connectivity.Shutdown:
		stateStr = "SHUTDOWN"
	}

	grpcClientConnections.WithLabelValues(grpcService, grpcMethod, stateStr).Set(1)
}
