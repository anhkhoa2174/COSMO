package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registry     *prometheus.Registry
	registryOnce sync.Once
	metricsOnce  sync.Once
)

// GetRegistry returns the Prometheus registry (creates one if it doesn't exist)
func GetRegistry() *prometheus.Registry {
	registryOnce.Do(func() {
		registry = prometheus.NewRegistry()
		// Register default Go metrics
		registry.MustRegister(prometheus.NewGoCollector())
	})
	return registry
}

// initMetrics initializes the metrics with our custom registry
func initMetrics() {
	metricsOnce.Do(func() {
		reg := GetRegistry()

		// Register all metrics with our custom registry
		reg.MustRegister(httpRequestsTotal)
		reg.MustRegister(httpErrorsTotal)
		reg.MustRegister(httpRequestDuration)
		reg.MustRegister(httpRequestSize)
		reg.MustRegister(httpResponseSize)
		reg.MustRegister(activeConnections)
		reg.MustRegister(connectedUsers)
		reg.MustRegister(dbConnectionsActive)
		reg.MustRegister(dbConnectionsIdle)
		reg.MustRegister(dbQueryDuration)
		reg.MustRegister(dbQueriesTotal)
		reg.MustRegister(cacheHits)
		reg.MustRegister(cacheMisses)
		reg.MustRegister(jobsTotal)
		reg.MustRegister(jobDuration)
		reg.MustRegister(activeJobs)
		reg.MustRegister(campaignsActive)
		reg.MustRegister(emailsSent)
		reg.MustRegister(contactsProcessed)
		reg.MustRegister(aiRequestsTotal)
		reg.MustRegister(aiRequestDuration)
		reg.MustRegister(aiTokensUsed)
	})
}
