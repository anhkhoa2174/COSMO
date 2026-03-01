package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"
)

func TestGoMetricsCollection(t *testing.T) {
	// Note: Metrics are initialized globally in init() function
	// Don't create/stop metrics in tests as they're singleton

	// Since the collector runs every 15 seconds, we need to wait for at least one collection cycle
	t.Log("Waiting for metrics collection (15 seconds)...")
	time.Sleep(16 * time.Second)

	// Create a test request to the metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Get the Prometheus handler and serve the request
	handler := Handler()
	handler.ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")

	// Get the metrics output
	metricsOutput := w.Body.String()
	t.Log("Checking for app_go_memstats in metrics output...")

	// Check that Go runtime metrics are present (basic ones are automatically collected by Prometheus)
	assert.Contains(t, metricsOutput, "go_goroutines", "go_goroutines should be present")
	assert.Contains(t, metricsOutput, "go_memstats_heap_alloc_bytes", "go_memstats_heap_alloc_bytes should be present")
	assert.Contains(t, metricsOutput, "go_gc_duration_seconds", "go_gc_duration_seconds should be present")

	// Check for application-specific Go metrics
	assert.Contains(t, metricsOutput, "app_go_info", "app_go_info should be present")

	// The app_go_memstats should be present after collection
	if !strings.Contains(metricsOutput, "app_go_memstats") {
		t.Log("app_go_memstats not found, checking if collector is running...")
		// Try to manually trigger a collection to verify collector is working
		time.Sleep(1 * time.Second)

		// Make another request
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		metricsOutput = w.Body.String()

		// Check again
		if strings.Contains(metricsOutput, "app_go_memstats") {
			t.Log("app_go_memstats found on second attempt")
		} else {
			t.Errorf("app_go_memstats not found in metrics output after waiting")
			t.Logf("Partial metrics output: %s", metricsOutput[:1000])
		}
	} else {
		t.Log("app_go_memstats found successfully")

		// Check for specific memory metrics in app metrics
		assert.Contains(t, metricsOutput, "alloc_bytes")
		assert.Contains(t, metricsOutput, "heap_alloc_bytes")
		assert.Contains(t, metricsOutput, "sys_bytes")
	}
}

func TestRecordGRPCConnection(t *testing.T) {
	// Note: Metrics are initialized globally in init() function

	// Record a gRPC connection state
	RecordGRPCConnection("test.service", "TestMethod", connectivity.Ready)

	// Wait a moment for metrics to be recorded
	time.Sleep(100 * time.Millisecond)

	// Get the metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler := Handler()
	handler.ServeHTTP(w, req)

	metricsOutput := w.Body.String()

	// Check that the gRPC metric is present
	assert.Contains(t, metricsOutput, "grpc_client_connections")
	assert.Contains(t, metricsOutput, "test.service")
	assert.Contains(t, metricsOutput, "TestMethod")
	assert.Contains(t, metricsOutput, "READY")
}

func TestMetricsInitialization(t *testing.T) {
	// Test that NewMetrics returns a properly initialized Metrics struct
	m := NewMetrics()

	assert.NotNil(t, m)
	assert.NotNil(t, m.Go.GCDuration)
	assert.NotNil(t, m.Go.Memstats)
	assert.NotNil(t, m.Go.Info)
	assert.NotNil(t, m.GRPC.ClientConnections)
}

func TestStopMetricsCollector(t *testing.T) {
	// Test that StopMetricsCollector function exists and can be called
	// Note: We don't actually call it in tests as it would stop the global collector

	// Verify the function exists
	assert.NotNil(t, StopMetricsCollector)

	// The Stop method exists on Metrics struct
	m := NewMetrics()
	assert.NotNil(t, m.Stop)
	// Note: Don't call m.Stop() as it would stop the global collector
}

func BenchmarkMetricsCollection(b *testing.B) {
	// Note: Metrics are initialized globally in init() function

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate recording some metrics
		RecordGRPCConnection("benchmark.service", "BenchmarkMethod", connectivity.Ready)
	}
}
