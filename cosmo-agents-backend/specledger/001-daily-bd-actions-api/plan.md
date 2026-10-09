# Implementation Plan: Daily BD Actions API (Backend v3 — Frontend Compatibility)

**Branch**: `001-daily-bd-actions-api` | **Date**: 2026-03-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specledger/001-daily-bd-actions-api/spec.md` (v3, updated 2026-03-03)

**Note**: This is a v3 plan update. The initial implementation (v1-v2, 28 tasks) is complete. This plan addresses gaps identified during frontend integration testing — 6 specific compatibility issues where the backend response shapes don't match the frontend TypeScript types exactly.

## Summary

The backend implementation is 90% complete (all 7 endpoints working, DB migrations applied, async generation pipeline functional). However, frontend integration testing revealed 6 response shape mismatches that break the frontend rendering. This plan targets **only those gaps** — no new endpoints, no architecture changes.

## Technical Context

**Language/Version**: Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`)
**Primary Dependencies**: Fiber v3-rc2 (HTTP), GORM v1.31 (ORM), Asynq v0.25.1 (task queue), OpenAI SDK v1.12 (LLM), Redis v9.14 (cache/pubsub), zerolog v1.34 (logging), Prometheus (metrics)
**Storage**: PostgreSQL (GORM), Redis (Asynq, SSE pub/sub, event replay)
**Testing**: `go test ./...` (unit + integration)
**Target Platform**: Linux server (Docker)
**Project Type**: Web application (backend API)
**Performance Goals**: POST /generate <200ms, GET /daily-actions <500ms, PATCH <500ms, POST /chat TTFT <2s, SSE event latency <30s
**Constraints**: All response shapes must exactly match frontend TypeScript types. No breaking changes to existing endpoints.
**Scale/Scope**: 50 concurrent BD reps, 200 contacts per rep

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Specification-First**: Spec.md v3 complete with prioritized user stories and exact response shapes
- [x] **Test-First**: Contract tests planned — verify response shapes match frontend TypeScript types
- [x] **Code Quality**: `make fmt` + `make lint` (golangci-lint) already configured
- [x] **UX Consistency**: User flows documented in spec.md acceptance scenarios
- [x] **Performance**: Metrics defined in Technical Context
- [x] **Observability**: zerolog + Prometheus already instrumented (T026/T027 complete)
- [x] **Issue Tracking**: Existing epic for 001-daily-bd-actions-api

**Complexity Violations**: None — all changes are targeted fixes to existing code, no new architecture.

## Identified Gaps (v3)

These 6 gaps were found by comparing the frontend TypeScript types (`daily-actions.ts`) with the backend Go response structs (`v1/daily_action.go`) and handler logic (`handler.go`):

### Gap 1: `category_counts[].color` field missing

**Frontend expects**: `{ category, label, count, icon, color }` on each CategoryBadge
**Backend returns**: `{ category, label, count, icon }` — no `color`
**Fix**: Add `Color string` field to `CategoryCountResponse` struct. Update `AgentBriefingToResponse` mapper. Update generation worker to include color in AgentBriefing JSONB.

### Gap 2: `categories[].icon`, `categories[].color`, `categories[].description` not populated

**Frontend expects**: Each ActionCategory has `icon`, `color`, `description` for rendering headers
**Backend returns**: `icon: ""`, `color: ""`, `description: ""` — handler doesn't set them
**Fix**: Add category metadata map in handler `GetDailyActions()` that populates icon (emoji), color, and description for each category ID.

### Gap 3: `generated_at` not set in briefing response

**Frontend expects**: `generated_at: "ISO8601"` when generation_status is "ready"
**Backend returns**: `generated_at: null` always — handler never populates it from the generation record
**Fix**: Set `GeneratedAt` from `gen.GeneratedAt` in the briefing response builder.

### Gap 4: `pipeline_summary` not computed

**Frontend expects**: `pipeline_summary: { total_active_contacts, contacts_by_stage, contacts_by_lifecycle, response_rate_7d, meetings_booked_7d, avg_response_time_hours }`
**Backend returns**: `pipeline_summary: null` — never computed
**Fix**: Add `BuildPipelineSummary()` method to service. Query contacts grouped by outreach_stage and lifecycle_stage. Compute 7-day response rate and meeting count. Cache in generation JSONB.

### Gap 5: Chat endpoint returns plain JSON instead of Vercel AI SDK streaming protocol

**Frontend expects**: `Content-Type: text/plain; charset=utf-8` with `X-Vercel-AI-Data-Stream: v1` header, response body in `{code}:{json}\n` format (`f:`, `0:`, `e:`, `d:` lines)
**Backend returns**: Plain JSON `{ role: "assistant", content: "..." }` in a single response
**Fix**: Rewrite Chat handler to use `SetBodyStreamWriter()` with Vercel AI SDK data stream protocol. Use OpenAI streaming API, re-format each token as `0:{json}\n`. Add start/finish framing lines.

### Gap 6: SSE `Last-Event-ID` replay not implemented

**Frontend expects**: On reconnect with `Last-Event-ID`, missed events are replayed
**Backend**: Handler reads the header but doesn't query for missed events
**Fix**: In SSEStream handler, after registering the event channel, query `SSEEventRepository` for events created after `Last-Event-ID` timestamp and replay them before entering the live event loop.

## Project Structure

### Documentation (this feature)

```text
specledger/001-daily-bd-actions-api/
├── plan.md              # This file (v3)
├── spec.md              # Feature spec (v3, updated 2026-03-03)
├── research.md          # Phase 0 output (v2, mostly stable)
├── data-model.md        # Phase 1 output (v2, needs color field update)
├── quickstart.md        # Phase 1 output (stable)
├── contracts/           # Frontend API contract (synced 2026-03-03)
│   ├── daily-actions-api.yaml
│   └── daily-actions-client.ts
├── checklists/
│   └── requirements.md  # Spec quality checklist (v3)
├── issues.jsonl         # Beads issue tracking
└── tasks.md             # Phase 2 output (to be regenerated by /specledger.tasks)
```

### Source Code (files to modify)

```text
internal/
├── schema/v1/
│   └── daily_action.go          # Gap 1: Add Color to CategoryCountResponse
├── handler/v1/daily_action/
│   └── handler.go               # Gaps 2,3,5,6: Category metadata, generated_at, chat streaming, SSE replay
├── mapper/
│   └── daily_action.go          # Gap 1: Update AgentBriefingToResponse
├── service/daily_action/
│   ├── service.go               # Gap 4: Add BuildPipelineSummary()
│   └── briefing.go              # Gap 1: Include color in briefing generation
└── worker/daily_action/
    └── generation_worker.go     # Gap 1: Include color when building category_counts
```

**Structure Decision**: No new files needed. All 6 gaps are modifications to existing files. This matches the existing codebase structure established in v1-v2 implementation.

## Complexity Tracking

> No violations. All changes are targeted field additions and logic fixes within existing architecture.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
