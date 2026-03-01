# Integration Guide for Cosmo Backend Monitoring

This guide shows how to integrate the monitoring stack with your Cosmo Backend application.

## 1. Update main.go

Add monitoring initialization to your main.go file:

```go
package main

import (
    "context"
    "os"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/rockship/cosmo-agents-go/internal/middleware"
    "github.com/rockship/cosmo-agents-go/pkg/metrics"
    "github.com/rockship/cosmo-agents-go/pkg/telemetry"
)

func main() {
    // Load environment variables
    _ = godotenv.Load()

    // Initialize monitoring
    setupMonitoring()

    // Create Fiber app
    app := fiber.New(fiber.Config{
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    })

    // Add monitoring middleware (BEFORE routes)
    addMonitoringMiddleware(app)

    // Your existing routes...
    RegisterV1Routes(app, deps)
    RegisterV2Routes(app, deps)
    RegisterV3Routes(app, deps)

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    app.Listen(":" + port)
}

func setupMonitoring() {
    // Initialize OpenTelemetry
    tp, err := telemetry.InitTracer(telemetry.Config{
        ServiceName:    "cosmo-backend",
        ServiceVersion: os.Getenv("SERVICE_VERSION"),
        Environment:    os.Getenv("ENVIRONMENT"),
        EnableTracing:  os.Getenv("ENABLE_TRACING") != "false",
        // Use Tempo (from Docker Compose)
        TempoEndpoint: "http://host.docker.internal:4318",
        SampleRate:     0.1, // 10% sampling
    })

    if err != nil {
        log.Printf("Failed to initialize tracer: %v", err)
    } else {
        // Graceful shutdown
        defer func() {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            tp.Shutdown(ctx)
        }()
    }
}

func addMonitoringMiddleware(app *fiber.App) {
    // Add tracing middleware
    app.Use(middleware.Tracing("cosmo-backend"))

    // Add metrics middleware
    app.Use(middleware.MetricsWithRoute())

    // Add metrics endpoint for Prometheus
    app.Get("/metrics", func(c *fiber.Ctx) error {
        handler := metrics.Handler()
        handler.ServeHTTP(c.Context().ResponseWriter, c.Context().Request)
        return nil
    })

    // Health check endpoint
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "status": "ok",
            "timestamp": time.Now().Unix(),
        })
    })
}
```

## 2. Environment Variables

Create or update your `.env` file:

```bash
# Monitoring Configuration
ENVIRONMENT=production
SERVICE_VERSION=1.0.0
ENABLE_TRACING=true

# Docker-specific
TEMPO_ENDPOINT=http://host.docker.internal:4318
PROMETHEUS_ENDPOINT=http://host.docker.internal:9090
LOKI_ENDPOINT=http://host.docker.internal:3100
```

## 3. Docker Application Setup

Option A: Run your app outside Docker (for development)

```yaml
# Add to your existing docker-compose.yml or create a new one
version: '3.8'

services:
  cosmo-backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENVIRONMENT=development
      - TEMPO_ENDPOINT=http://tempo:4318
    networks:
      - monitoring
    depends_on:
      - prometheus
      - loki
      - tempo

networks:
  monitoring:
    external:
      name: docker_monitoring
```

Option B: Run your app in Docker with monitoring stack

Update the `docker-compose-1.yml` to include your application:

```yaml
services:
  # ... existing monitoring services ...

  cosmo-backend:
    build: .
    container_name: cosmo-backend
    ports:
      - "8080:8080"
    environment:
      - ENVIRONMENT=production
      - SERVICE_VERSION=1.0.0
      - ENABLE_TRACING=true
      - TEMPO_ENDPOINT=http://tempo:4318
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
    volumes:
      - ./logs:/var/log/app
    networks:
      - monitoring
    depends_on:
      - prometheus
      - loki
      - tempo
    restart: unless-stopped
```

## 4. Update Prometheus Configuration

Make sure your `prometheus.yml` includes your application:

```yaml
scrape_configs:
  # ... existing jobs ...

  - job_name: 'cosmo-backend'
    static_configs:
      - targets:
          - 'host.docker.internal:8080'  # If app runs outside Docker
          # - 'cosmo-backend:8080'       # If app runs in Docker
    scrape_interval: 15s
    metrics_path: /metrics
    scrape_timeout: 10s
```

## 5. Custom Metrics in Your Handlers

Here are examples of adding custom metrics to your handlers:

```go
// Example handler with custom metrics
func (h *CampaignHandler) CreateCampaign(c *fiber.Ctx) error {
    ctx := c.UserContext()

    // Add trace attributes
    telemetry.AddSpanAttributes(ctx,
        attribute.String("operation", "create_campaign"),
        attribute.String("user_id", c.Locals("user_id").(string)),
    )

    start := time.Now()

    // Your business logic here...
    campaign, err := h.service.CreateCampaign(ctx, req)

    if err != nil {
        telemetry.RecordError(ctx, err)
        metrics.RecordCampaign("create", "error", time.Since(start))
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    // Record success metrics
    metrics.RecordCampaign("create", "success", time.Since(start))
    metrics.SetActiveCampaigns(h.service.GetActiveCampaignCount())

    telemetry.AddSpanEvent(ctx, "campaign_created",
        attribute.String("campaign_id", campaign.ID),
    )

    return c.JSON(campaign)
}

// Example database operation
func (r *CampaignRepository) Create(ctx context.Context, campaign *Campaign) error {
    ctx, span := telemetry.StartSpan(ctx, "database.create_campaign")
    defer span.End()

    telemetry.AddSpanAttributes(ctx,
        attribute.String("db.table", "campaigns"),
        attribute.String("db.operation", "INSERT"),
    )

    start := time.Now()
    err := r.db.WithContext(ctx).Create(campaign).Error
    duration := time.Since(start)

    metrics.RecordDBQuery("INSERT", "campaigns", duration, err == nil)

    if err != nil {
        telemetry.RecordError(ctx, err)
        span.SetAttributes(attribute.Bool("db.success", false))
    } else {
        span.SetAttributes(attribute.Bool("db.success", true))
    }

    return err
}

// Example cache operation
func (s *CampaignService) GetCampaign(ctx context.Context, id string) (*Campaign, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("campaign:%s", id)
    if cached, found := s.cache.Get(cacheKey); found {
        metrics.RecordCacheHit("redis")
        telemetry.AddSpanEvent(ctx, "cache_hit",
            attribute.String("cache_key", cacheKey),
        )
        return cached, nil
    }

    metrics.RecordCacheMiss("redis")

    // Get from database
    campaign, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, err
    }

    // Store in cache
    s.cache.Set(cacheKey, campaign, 5*time.Minute)

    return campaign, nil
}
```

## 6. Worker/Job Monitoring

For your worker processes:

```go
func (w *EmailWorker) ProcessBatch(ctx context.Context, batch []Email) error {
    ctx, span := telemetry.StartSpan(ctx, "email_batch_process")
    defer span.End()

    telemetry.AddSpanAttributes(ctx,
        attribute.Int("batch_size", len(batch)),
        attribute.String("worker_id", w.id),
    )

    // Increment active jobs
    metrics.SetActiveJobs("email_batch", 1)
    defer metrics.SetActiveJobs("email_batch", 0)

    start := time.Now()
    successCount := 0

    for _, email := range batch {
        itemCtx, itemSpan := telemetry.StartSpan(ctx, "process_email")

        if err := w.sendEmail(itemCtx, email); err != nil {
            telemetry.RecordError(itemCtx, err)
            metrics.RecordEmailSent(email.CampaignID, "error")
        } else {
            successCount++
            metrics.RecordEmailSent(email.CampaignID, "sent")
        }

        itemSpan.End()
    }

    // Record batch metrics
    duration := time.Since(start)
    metrics.RecordJob("email_batch", "success", duration)

    telemetry.AddSpanEvent(ctx, "batch_completed",
        attribute.Int("success_count", successCount),
        attribute.Int("error_count", len(batch)-successCount),
        attribute.String("duration", duration.String()),
    )

    return nil
}
```

## 7. Running Everything

1. Start the monitoring stack:
   ```bash
   cd docker
   docker-compose -f docker-compose-1.yml up -d
   ```

2. Start your application:
   ```bash
   # If running outside Docker
   go run cmd/server/main.go

   # Or build and run
   go build -o cosmo-backend cmd/server/main.go
   ./cosmo-backend
   ```

3. Verify metrics are being scraped:
   - Open http://localhost:9090/targets
   - Check that `cosmo-backend` target is UP

4. Check logs in Grafana:
   - Open http://localhost:3000
   - Navigate to the "Cosmo Backend Logs" dashboard

5. View traces:
   - Open http://localhost:16686 (Jaeger)
   - Or view traces in Grafana if using Tempo

## 8. Next Steps

1. Set up alerts in Prometheus for:
   - High error rates
   - High latency
   - Database connection issues
   - High memory/CPU usage

2. Create custom dashboards for:
   - Business metrics (campaigns created, emails sent)
   - AI usage and costs
   - Database performance
   - Cache hit rates

3. Configure log retention policies based on your needs

4. Set up notification channels (Slack, email) for alerts