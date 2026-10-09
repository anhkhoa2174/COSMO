# cosmo-agents-backend Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-01

## Active Technologies
- Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`) + Fiber v3-rc2 (HTTP), GORM v1.31 (ORM), Asynq v0.25.1 (task queue), OpenAI SDK v1.12 (LLM), Temporal SDK v1.39 (workflows), zerolog v1.34 (logging), go-playground/validator v10.28 (validation), Redis v9.14 (cache/pubsub) (001-daily-bd-actions-api)
- PostgreSQL (GORM driver, prepared statements) + Redis (cache, task queue backend, SSE pub/sub) (001-daily-bd-actions-api)
- Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`) + Fiber v3-rc2 (HTTP), GORM v1.31 (ORM), Asynq v0.25.1 (task queue), OpenAI SDK v1.12 (LLM), Redis v9.14 (cache/pubsub), zerolog v1.34 (logging), Prometheus (metrics) (001-daily-bd-actions-api)
- PostgreSQL (GORM), Redis (Asynq, SSE pub/sub, event replay) (001-daily-bd-actions-api)

- Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`) + Fiber v3 (HTTP), GORM (ORM), Asynq (task queue), OpenAI SDK (LLM), Temporal SDK (workflows), zerolog (logging), OpenTelemetry (tracing), Prometheus (metrics) (001-daily-bd-actions-api)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`)

## Code Style

Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`): Follow standard conventions

## Recent Changes
- 001-daily-bd-actions-api: Added Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`) + Fiber v3-rc2 (HTTP), GORM v1.31 (ORM), Asynq v0.25.1 (task queue), OpenAI SDK v1.12 (LLM), Redis v9.14 (cache/pubsub), zerolog v1.34 (logging), Prometheus (metrics)
- 001-daily-bd-actions-api: Added Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`) + Fiber v3-rc2 (HTTP), GORM v1.31 (ORM), Asynq v0.25.1 (task queue), OpenAI SDK v1.12 (LLM), Temporal SDK v1.39 (workflows), zerolog v1.34 (logging), go-playground/validator v10.28 (validation), Redis v9.14 (cache/pubsub)

- 001-daily-bd-actions-api: Added Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`) + Fiber v3 (HTTP), GORM (ORM), Asynq (task queue), OpenAI SDK (LLM), Temporal SDK (workflows), zerolog (logging), OpenTelemetry (tracing), Prometheus (metrics)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
