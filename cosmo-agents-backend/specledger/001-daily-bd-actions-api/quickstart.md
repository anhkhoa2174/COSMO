# Quickstart: Daily BD Actions API

**Branch**: `001-daily-bd-actions-api` | **Date**: 2026-03-02

## Prerequisites

- Go 1.25+ installed
- PostgreSQL running (with existing `cosmo_agents` database)
- Redis running (for Asynq task queue, SSE pub/sub, event replay)
- `.env` configured with `DATABASE_URL`, `REDIS_URL`, `OPENAI_API_KEY`

## Dev Setup

```bash
# 1. Switch to feature branch
git checkout 001-daily-bd-actions-api

# 2. Install dependencies
go mod tidy

# 3. Run database migration
make migrate-up  # Applies 000048_create_daily_action_tables

# 4. Start server
make run-server

# 5. Start worker (separate terminal)
make run-worker
```

## Files to Create (Implementation Checklist)

### Domain Layer
- [ ] `internal/domain/daily_action/daily_action.go` — DailyAction, DailyActionGeneration, ActionSnooze, ActionCompletionLog, SSEEvent, ChatMessage models
- [ ] `internal/domain/daily_action/enums.go` — ActionType, ActionStatus, CategoryID, GenerationStatus enums

### Repository Layer
- [ ] `internal/repository/daily_action/generation_repository.go` — CRUD for DailyActionGeneration
- [ ] `internal/repository/daily_action/action_repository.go` — CRUD + category queries for DailyAction
- [ ] `internal/repository/daily_action/snooze_repository.go` — Snooze records
- [ ] `internal/repository/daily_action/completion_log_repository.go` — Append-only log
- [ ] `internal/repository/daily_action/sse_event_repository.go` — SSE event persistence
- [ ] `internal/repository/daily_action/chat_message_repository.go` — Chat history

### Service Layer
- [ ] `internal/service/daily_action/service.go` — Main service (generation orchestration, action execution)
- [ ] `internal/service/daily_action/priority.go` — Priority scoring algorithm
- [ ] `internal/service/daily_action/briefing.go` — Agent briefing + pipeline summary generation
- [ ] `internal/service/daily_action/chat.go` — Conversational agent (intent classification, tool routing)
- [ ] `internal/service/daily_action/sse_manager.go` — SSE connection registry + Redis pub/sub

### Handler Layer
- [ ] `internal/handler/v1/daily_action/handler.go` — HTTP handler with all endpoints
- [ ] `internal/schema/v1/daily_action.go` — Request/response DTOs

### Worker Layer
- [ ] `internal/worker/daily_action/generation_worker.go` — Async generation task handler

### Mapper Layer
- [ ] `internal/mapper/daily_action.go` — Domain ↔ schema mappers

### Infrastructure
- [ ] `migrations/000048_create_daily_action_tables.up.sql`
- [ ] `migrations/000048_create_daily_action_tables.down.sql`
- [ ] Update `cmd/server/dependencies.go` — Wire new repos, services, handlers
- [ ] Update `cmd/server/routes_v1.go` — Register /v1/daily-actions/* routes
- [ ] Update `cmd/worker/main.go` — Register generation worker task

## API Endpoints

| Method | Path | Handler | Spec FR |
|--------|------|---------|---------|
| POST | `/v1/daily-actions/generate` | Generate | FR-001..FR-007 |
| GET | `/v1/daily-actions` | GetDailyActions | FR-008, FR-009 |
| PATCH | `/v1/daily-actions/{action_id}` | UpdateAction | FR-011..FR-015 |
| GET | `/v1/daily-actions/events` | SSEStream | FR-016..FR-020 |
| GET | `/v1/daily-actions/summary` | GetSummary | FR-024, FR-025 |
| GET | `/v1/daily-actions/categories/{category_id}/actions` | LoadMoreActions | FR-010 |
| POST | `/v1/daily-actions/chat` | Chat | FR-021..FR-023 |

## Performance Targets

| Endpoint | Target | Notes |
|----------|--------|-------|
| POST /generate | <200ms response | Async — enqueues Asynq task |
| Full generation | <10s | For 200 contacts |
| GET /daily-actions | <500ms | Read from DB/cache |
| PATCH /{id} | <500ms | Including state machine + logging |
| GET /events (SSE) | <30s event latency | Via Redis pub/sub |
| POST /chat | <2s TTFT | Streaming via Vercel AI SDK protocol |
| GET /summary | <500ms | Computed from completion logs |
| GET /categories/{id}/actions | <200ms | Paginated DB query |

## Key Architecture Decisions

1. **Actions are persisted** — DailyAction rows have UUIDs, survive across GET requests, referenced by PATCH
2. **Generation is async** — POST /generate enqueues Asynq task, returns 202, pushes SSE `generation_complete` when done
3. **SSE via Fiber streaming** — `c.RequestCtx().SetBodyStreamWriter()` + Redis pub/sub for cross-instance
4. **Chat uses Vercel AI SDK data protocol** — Custom `{code}:{json}\n` line format, NOT standard SSE
5. **Priority 1=highest** — Weighted linear model: recency 30%, cadence 25%, meeting 25%, completeness 12%, lifecycle 8%
6. **Snooze is query-time** — No background worker; actions reappear when `snooze_until < NOW()`
7. **Contract-first** — `contracts/daily-actions-api.yaml` copied from frontend repo is source of truth
