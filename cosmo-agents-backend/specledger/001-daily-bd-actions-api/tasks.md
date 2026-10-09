# Tasks Index: Daily BD Actions API

Issue Graph Index into the tasks and phases for this feature implementation.
This index does **not contain tasks directly** — those are fully managed in `issues.jsonl`.

## Feature Tracking

* **Original Epic ID**: `SL-1315d7` (v1-v2 implementation, 28 tasks, **all closed**)
* **v3 Epic ID**: `SL-f3c001` (v3 frontend compatibility fixes, 8 tasks)
* **User Stories Source**: `specledger/001-daily-bd-actions-api/spec.md` (v3, updated 2026-03-03)
* **Research Inputs**: `specledger/001-daily-bd-actions-api/research.md`
* **Planning Details**: `specledger/001-daily-bd-actions-api/plan.md` (v3)
* **Data Model**: `specledger/001-daily-bd-actions-api/data-model.md`
* **Contract Definitions**: `specledger/001-daily-bd-actions-api/contracts/`

## Issue Query Hints

```bash
# All open v3 tasks
grep '"version:v3"' specledger/001-daily-bd-actions-api/issues.jsonl | grep '"open"'

# All tasks in a specific phase
grep '"phase:response-shapes"' specledger/001-daily-bd-actions-api/issues.jsonl

# View task details (pipe to jq for readability)
grep 'SL-f3c011' specledger/001-daily-bd-actions-api/issues.jsonl | jq .

# List all v1-v2 closed tasks
grep '"version:v3"' specledger/001-daily-bd-actions-api/issues.jsonl
```

---

# v3 Frontend Compatibility Fixes (Active)

## Context

The initial implementation (v1-v2, 28 tasks T001-T028) is **complete and closed**. This v3 task set addresses **6 specific frontend compatibility gaps** identified during integration testing where backend response shapes don't match frontend TypeScript types.

**No new endpoints. No architecture changes. Only targeted field additions and logic fixes to existing files.**

## v3 Tasks and Phases Structure

```
Epic: SL-f3c001 "v3 Frontend Compatibility Fixes"
├── Phase 1: SL-f3c010 "Response Shape Fixes" (Gaps 1,2,3)
│   ├── T029: SL-f3c011 "Add Color field to CategoryCount" [P]
│   ├── T030: SL-f3c012 "Update mapper and briefing for color" (blocked by T029)
│   ├── T031: SL-f3c013 "Add category metadata in handler" [P]
│   └── T032: SL-f3c014 "Set generated_at and completion_rate" [P]
│
├── Phase 2: SL-f3c020 "Pipeline Summary" (Gap 4)
│   ├── T033: SL-f3c021 "Implement BuildPipelineSummary service" [P]
│   └── T034: SL-f3c022 "Integrate into GetDailyActions handler" (blocked by T033)
│
├── Phase 3: SL-f3c030 "Chat Streaming Rewrite" (Gap 5)
│   └── T035: SL-f3c031 "Rewrite Chat to Vercel AI SDK protocol"
│
└── Phase 4: SL-f3c040 "SSE Last-Event-ID Replay" (Gap 6)
    └── T036: SL-f3c041 "Implement SSE event replay on reconnect"
```

## Convention Summary

| Type    | Description                     | Labels                                          |
| ------- | ------------------------------- | ----------------------------------------------- |
| epic    | v3 compatibility epic           | `spec:001-daily-bd-actions-api`, `version:v3`   |
| feature | Implementation phase            | `phase:<name>`, `version:v3`                    |
| task    | Implementation task             | `component:<area>`, `version:v3`                |

---

## Phase 1: Response Shape Fixes (Gaps 1, 2, 3)

**Feature**: `SL-f3c010` | **Priority**: P1 | **Dependencies**: None

**Goal**: Fix missing fields in GET /daily-actions response so frontend renders correctly.

**Files Modified**:
- `internal/domain/daily_action/daily_action.go` — Add Color to CategoryCount
- `internal/schema/v1/daily_action.go` — Add Color to CategoryCountResponse
- `internal/mapper/daily_action.go` — Map Color in AgentBriefingToResponse
- `internal/service/daily_action/briefing.go` — Include color in generated briefing
- `internal/handler/v1/daily_action/handler.go` — Category metadata, generated_at, completion_rate

### Tasks

- [ ] **T029** `SL-f3c011` [P] Add Color field to domain CategoryCount struct and schema CategoryCountResponse
- [ ] **T030** `SL-f3c012` Update mapper AgentBriefingToResponse to include Color; update briefing generation (depends on T029)
- [ ] **T031** `SL-f3c013` [P] Add category metadata map in GetDailyActions handler (icon, color, description for each category)
- [ ] **T032** `SL-f3c014` [P] Set generated_at from gen.GeneratedAt and compute completion_rate in briefing response

**Parallel Execution**: T029, T031, T032 can run in parallel. T030 depends on T029.

**Checkpoint**: GET /daily-actions response includes category_counts[].color, categories[].icon/color/description, generated_at, completion_rate. Frontend CategoryBadge and category headers render correctly.

---

## Phase 2: Pipeline Summary (Gap 4)

**Feature**: `SL-f3c020` | **Priority**: P1 | **Dependencies**: None (independent of Phase 1)

**Goal**: Compute and return pipeline_summary with 6 fields in the briefing response.

**Files Modified**:
- `internal/service/daily_action/service.go` — New BuildPipelineSummary method
- `internal/handler/v1/daily_action/handler.go` — Call and include pipeline_summary

### Tasks

- [ ] **T033** `SL-f3c021` Implement BuildPipelineSummary() service method (queries contacts by stage, 7-day metrics)
- [ ] **T034** `SL-f3c022` Integrate pipeline_summary into GetDailyActions handler response (depends on T033)

**Parallel Execution**: T033 can run in parallel with Phase 1 tasks. T034 depends on T033.

**Checkpoint**: GET /daily-actions response includes pipeline_summary with total_active_contacts, contacts_by_stage, contacts_by_lifecycle, response_rate_7d, meetings_booked_7d, avg_response_time_hours.

---

## Phase 3: Chat Streaming Rewrite (Gap 5)

**Feature**: `SL-f3c030` | **Priority**: P1 | **Dependencies**: None (independent)

**Goal**: Make POST /chat return Vercel AI SDK data stream protocol so frontend `useChat()` hook works.

**Files Modified**:
- `internal/handler/v1/daily_action/handler.go` — Rewrite Chat() method

### Tasks

- [ ] **T035** `SL-f3c031` Rewrite Chat handler: SetBodyStreamWriter, text/plain content type, f:/0:/e:/d: line format, X-Vercel-AI-Data-Stream header

**Parallel Execution**: Independent — can run in parallel with all other phases.

**Checkpoint**: POST /chat returns streamed response in Vercel AI SDK format. Frontend daily-actions-chat component renders streaming tokens.

---

## Phase 4: SSE Last-Event-ID Replay (Gap 6)

**Feature**: `SL-f3c040` | **Priority**: P2 | **Dependencies**: None (independent)

**Goal**: Replay missed SSE events on client reconnect using Last-Event-ID header.

**Files Modified**:
- `internal/handler/v1/daily_action/handler.go` — SSEStream() method

### Tasks

- [ ] **T036** `SL-f3c041` Read Last-Event-ID header, call FindByUserSince(), replay missed events before live loop

**Parallel Execution**: Independent — can run in parallel with all other phases.

**Checkpoint**: SSE reconnects with Last-Event-ID replay missed events. Frontend receives missed events without data loss.

---

## Dependencies & Execution Order

### Phase Dependencies

All 4 phases are **independent** and can be executed in parallel if staffed.

```
Phase 1 (Response Shapes) ──┐
Phase 2 (Pipeline Summary) ─┼──> All complete = v3 done
Phase 3 (Chat Streaming) ───┤
Phase 4 (SSE Replay) ───────┘
```

### Within-Phase Dependencies

- **Phase 1**: T029 → T030 (sequential), T031 and T032 parallel with T029
- **Phase 2**: T033 → T034 (sequential)
- **Phase 3**: T035 (single task)
- **Phase 4**: T036 (single task)

### Maximum Parallelism

With unlimited staffing, these tasks can run simultaneously:
- T029, T031, T032, T033, T035, T036 (6 tasks in parallel)
- Then T030, T034 (2 tasks, after their blockers)

### Recommended Sequential Order (single agent)

1. T029 → T030 (color field, fastest)
2. T031 (category metadata)
3. T032 (generated_at + completion_rate)
4. T033 → T034 (pipeline summary)
5. T035 (chat streaming — largest single task)
6. T036 (SSE replay)

## Implementation Strategy

### MVP (Phase 1 + Phase 2)

Phase 1 + 2 fix the most visible frontend rendering issues (empty badges, missing timestamps, broken progress, empty pipeline widget). Ship this first.

### Incremental Delivery

1. **Phase 1** — Unblocks frontend rendering of briefing page
2. **Phase 2** — Enables pipeline health widget
3. **Phase 3** — Enables conversational chat feature
4. **Phase 4** — Enables resilient SSE reconnection

---

## Summary

| Metric | Value |
|--------|-------|
| Total v3 tasks | 8 |
| Total v3 phases | 4 |
| Files modified | 5 (existing only, no new files) |
| Tasks per gap | Gap 1: 2, Gap 2: 1, Gap 3: 1, Gap 4: 2, Gap 5: 1, Gap 6: 1 |
| Max parallelism | 6 tasks simultaneously |
| Critical path | T029 → T030, T033 → T034 |
| Estimated MVP | Phase 1 + Phase 2 (6 tasks) |
| Story traceability | US4 (SSE), US6 (Chat), all others cross-cutting |

---

# v1-v2 Original Implementation (All Closed)

The original implementation had 28 tasks (T001-T028) across 10 phases. All are closed. Kept here for reference.

## Original Phases (SL-1315d7 epic, all closed)

| Phase | Feature ID | Title | Tasks |
|-------|-----------|-------|-------|
| 1 | SL-9ab0ed | Setup & Infrastructure | T001-T006 (6 tasks) |
| 2 | SL-35f276 | Foundational | T007-T009 (3 tasks) |
| 3 | SL-d7679b | US1 Async Generation (P1) | T010-T012 (3 tasks) |
| 4 | SL-cbf564 | US2 Read Briefing (P1) | T013-T014 (2 tasks) |
| 5 | SL-0a5127 | US3 Update Action State (P1) | T015-T018 (4 tasks) |
| 6 | SL-b72713 | US4 SSE Events (P2) | T019-T021 (3 tasks) |
| 7 | SL-fbb216 | US5 Category Pagination (P2) | T022 (1 task) |
| 8 | SL-718194 | US6 Chat (P2) | T023-T024 (2 tasks) |
| 9 | SL-829e05 | US7 Daily Summary (P3) | T025 (1 task) |
| 10 | SL-488f8b | Polish | T026-T028 (3 tasks) |

---

> This file is an index only. Implementation data lives in `issues.jsonl`.
> Use `grep` with `jq` to query the issue store.
