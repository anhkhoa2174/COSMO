# Phase 0 Research: Daily BD Actions API

**Branch**: `001-daily-bd-actions-api` | **Date**: 2026-03-02

## Prior Work

The following existing backend infrastructure is directly leveraged (no new infrastructure dependencies):

| System | Location | Reuse Level |
|--------|----------|-------------|
| Outreach State Machine | `internal/service/outreach/outreach_service.go` | Direct — `DetermineConversationState()`, state transitions |
| Draft Generation | `internal/service/outreach/` | Direct — `GenerateDraftForHandlerWithLanguage()` |
| Meeting Prep | `internal/service/outreach/` | Direct — `GenerateMeetingPrep()` |
| Contact Suggestion | `internal/service/outreach/` | Wrapped — `SuggestContacts()` provides base list |
| Intent Classification | `internal/service/intent/` | Direct — email reply intent classification |
| Gmail/Outlook Pipeline | `internal/service/gmail/`, `internal/worker/gmailnotification/` | Indirect — triggers `prospect_replied` SSE events |
| Asynq Task Queue | `pkg/worker/` | Direct — async generation task |
| WebSocket Pub/Sub | `internal/handler/v1/pubsub/websocket_manager.go` | Pattern — SSEManager follows same Redis pub/sub design |
| Relationship Scorer | `internal/agents/relationship_scorer_agent.go` | Pattern — weighted additive model (0-100) |
| Contact Enrichment | `internal/domain/contact/contact.go` | Direct — `MissingFields`, `Status`, `CalculateStatus()` |

## Research Topic 1: SSE with Fiber v3

### Decision: Use `c.RequestCtx().SetBodyStreamWriter()` with Redis pub/sub

### Rationale
Fiber v3 (v3.0.0-rc.2) has no built-in SSE middleware. However, the underlying `fasthttp` exposes `SetBodyStreamWriter(func(w *bufio.Writer))` which keeps the connection open until the callback returns. This is the correct primitive for long-lived SSE streaming.

### Implementation Pattern
- **SSEManager**: New struct modeled after existing `WebSocketManager` at `internal/handler/v1/pubsub/websocket_manager.go`. Manages per-user SSE connections with `map[uuid.UUID][]chan *SSEEvent` (supports multiple tabs).
- **Redis Pub/Sub**: Per-user channels (`sse:daily-actions:{user_id}`) for cross-instance fan-out. Same pattern as existing WebSocket system.
- **Last-Event-ID Replay**: Redis sorted set per user (`sse:events:{user_id}`) with 5-minute TTL. Events stored with timestamp score. On reconnect, `ZRANGEBYSCORE` replays missed events.
- **Disconnect Detection**: `w.Flush()` returns error when client disconnects. Combined with 15-second heartbeat via SSE comment lines (`: heartbeat\n\n`).
- **Headers**: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`.

### Key Gotchas
1. Extract all values from `fiber.Ctx` BEFORE entering `SetBodyStreamWriter` callback (Ctx is recycled after handler returns).
2. One goroutine per SSE connection is consumed in the fasthttp worker pool. At 50 concurrent users this is negligible.
3. Send `retry: 3000\n\n` on initial connect to control browser reconnection interval.
4. Fiber auth middleware works fine — runs before handler, sets `c.Locals()`.

### Alternatives Considered
- **WebSocket**: Already exists in codebase, but SSE is better for unidirectional server-to-client events. SSE has built-in browser auto-reconnect (`EventSource` API) and native Last-Event-ID support.
- **Long Polling**: Simpler but higher latency and server load.

## Research Topic 2: Vercel AI SDK Data Stream Protocol

### Decision: Implement custom line-based protocol (`{code}:{json}\n`) for chat endpoint

### Rationale
The Vercel AI SDK v4 (installed at `ai@4.3.19` in the frontend) uses a custom data stream protocol — NOT standard SSE format. The frontend's `useChat` hook with `streamProtocol: "data"` (default) reads the response body as a raw byte stream, splitting on `\n` and parsing `{code}:{json}` lines.

### Wire Format
Each line: `{single_char_code}:{json_encoded_value}\n`

| Code | Name | JSON Value | When to Send |
|------|------|------------|-------------|
| `f` | start_step | `{"messageId":"<uuid>"}` | Once at start |
| `0` | text | `"<escaped text>"` | Each LLM token |
| `2` | data | `[{...}]` (array) | Optional custom data |
| `3` | error | `"error message"` | On error |
| `e` | finish_step | `{"finishReason":"stop","isContinued":false}` | End of step |
| `d` | finish_message | `{"finishReason":"stop"}` | End of message |

### Required Response Headers
```
Content-Type: text/plain; charset=utf-8
X-Vercel-AI-Data-Stream: v1
Cache-Control: no-cache
Connection: keep-alive
```

**Critical**: Content-Type is `text/plain; charset=utf-8`, NOT `text/event-stream`. The frontend contract YAML says `text/event-stream` but the actual Vercel AI SDK parser expects plain text with its own line protocol.

### Minimum Viable Streaming Sequence
```
f:{"messageId":"msg-abc123"}
0:"Hello"
0:", how can I help?"
e:{"finishReason":"stop","isContinued":false}
d:{"finishReason":"stop"}
```

### Go Integration
- Use `github.com/sashabaranov/go-openai` (already in go.mod v1.41) `CreateChatCompletionStream()`.
- For each `stream.Recv()` chunk, re-format delta content as `0:{json.Marshal(text)}\n`.
- Flush `*bufio.Writer` after each line for real-time delivery.
- Text values MUST be `json.Marshal`'d (proper escaping of quotes, newlines).

### Alternatives Considered
- **Standard SSE format**: Would require the frontend to use `streamProtocol: "text"` mode instead of default `"data"` mode. Since the frontend contract is the source of truth and uses `useChat` default, we must implement the data stream protocol.

## Research Topic 3: Priority Scoring Algorithm

### Decision: Weighted Linear Additive Model (1-100, 1=highest)

### Rationale
The codebase already has a relationship strength scorer (`internal/agents/relationship_scorer_agent.go`) using weighted additive model (0-100). The fit scoring system (`internal/skills/scoring.go`) uses `FitResult{FitScore int, ScoreBreakdown map[string]int}`. Both patterns validate the approach.

### Weight Distribution

| Factor | Weight | Data Source | Computation |
|--------|--------|-------------|-------------|
| Response Recency | 30% | `OutreachState.LastInteractionAt`, `.ConversationState` | REPLIED + <1h = 100, <4h = 80, <24h = 60, <72h = 40, else = 20 |
| Cadence Deadline | 25% | `OutreachState.DaysSinceLastInteraction`, `.FollowupCount` | In FU window = 100, slightly overdue = 70, well overdue = 40, not due = 10 |
| Meeting Proximity | 25% | `Meeting.Time`, `.Status` | <2h = 100, <4h = 90, <8h = 75, <24h = 50, <48h = 30, else = 5 |
| Data Completeness | 12% | `Contact.MissingFields` | 4+ missing = 100, 3 = 50, 2 = 30, 1 = 10, 0 = 0 |
| Lifecycle Stage | 8% | `Contact.BusinessStage` | PRE_SALES = 100, SALES = 50, POST_SALES = 10 |

### Formula
```
urgency = (w1*recency + w2*cadence + w3*meeting + w4*completeness + w5*lifecycle) / 100
priority_score = max(1, min(100, round(100 - urgency)))
```

Score 1 = highest urgency → highest priority. Score 100 = lowest.

### Follow-Up Cadence Config (from existing service)
- Follow-Up #1: Day 4-5 after initial send (configurable via `Config.FollowUp1MinDays`)
- Follow-Up #2: Day 4-5 after FU#1 (Day 9-12 total)
- Max follow-ups: 2 (before drop)
- No-reply auto-transition: 8 hours
- Re-engage threshold: 60 days

### Priority Factor Breakdown Response
Each action includes `priority_factors[]` array matching the frontend `PriorityFactor` type:
```json
[
  {"factor": "response_recency", "value": 80, "description": "Replied 3 hours ago"},
  {"factor": "cadence_deadline", "value": 100, "description": "Follow-up #1 due today (day 5)"},
  {"factor": "meeting_proximity", "value": 0, "description": "No upcoming meeting"},
  {"factor": "data_completeness", "value": 10, "description": "1 field missing (email)"},
  {"factor": "lifecycle_stage", "value": 100, "description": "Pre-sales pipeline"}
]
```

## Research Topic 4: Existing Outreach Service Reuse

### Decision: Wrap existing service methods — no duplication

### Key Methods to Reuse Directly
| Method | Purpose in Daily Actions |
|--------|------------------------|
| `SuggestContacts(ctx, userID, orgID, isAdmin, "mixed", 200)` | Get all contacts needing actions |
| `DetermineConversationState(ctx, contact, userID)` | Classify each contact → determine action type |
| `GenerateDraftForHandlerWithLanguage(ctx, userID, contactID, lang, userName)` | Generate draft for outreach/followup actions |
| `GenerateMeetingPrep(ctx, userID, meetingID, lang)` | Generate meeting briefing |
| `UpdateOutreachForHandler(ctx, userID, contactID, event, content, channel, sentiment)` | Execute `mark_sent` transition |
| `RecordAction(ctx, outreachFeedbackInput)` | Record feedback (used_draft/modified_draft/wrote_own) |

### Action Type Mapping (Contact State → Daily Action Type)
| ConversationState | NextStep | → Action Type | → Category |
|------------------|----------|---------------|------------|
| REPLIED | any | `respond` | `replied` |
| NO_REPLY | FOLLOW_UP_1/2 | `followup` | `followup` |
| COLD | SEND | `outreach` | `new_outreach` |
| any | PREPARE_MEETING | `meeting_prep` | `meeting_prep` |
| any (status=pending) | any | `enrich` | `enrichment` |

### Domain Model Inventory (Existing)
- **ConversationState**: COLD, NO_REPLY, REPLIED, POST_MEETING, DROPPED
- **NextStepAction**: SEND, FOLLOW_UP_1, FOLLOW_UP_2, SET_MEETING, FOLLOW_UP_MEETING_1/2, PREPARE_MEETING, WAIT, FOLLOW_UP, DROP
- **Scenario**: role_based, industry_based, no_reply_followup, post_reply, meeting_confirmation, post_meeting, re_engage
- **OutreachIntent**: INTRO, FOLLOW_UP, RE_ENGAGE, POST_MEETING
- **InteractionChannel**: LinkedIn, Email, Call, Meeting, Note
- **FeedbackAction**: used_draft, modified_draft, wrote_own, skipped

### What's New (not in existing service)
1. **DailyAction persistence** — new domain model with UUID for frontend PATCH
2. **DailyActionGeneration** — tracking generation runs (status, timing)
3. **ActionSnooze** — snooze records with reappearance time
4. **ActionCompletionLog** — append-only audit log
5. **SSEEvent** — transient event storage for Last-Event-ID replay
6. **ChatMessage** — conversation history for the agent chat
7. **AgentBriefing generation** — AI narrative (greeting + strategy + memory refs)
8. **Priority scoring** — new weighted scoring algorithm (see Topic 3)
9. **Category pagination** — offset-based within categories
10. **SSE streaming** — real-time event delivery (see Topic 1)

## Research Topic 5: Snooze System

### Decision: PostgreSQL table with query-time filtering (no background worker)

### Rationale
Snooze is a query-time concept. When fetching active actions, exclude those where `snooze_until > NOW()`. When `NOW() > snooze_until`, the action naturally reappears. No background worker needed for the core behavior.

### Snooze Expiry SSE Events
For the `snooze_expired` SSE event (FR-018), use a lightweight Asynq scheduled task:
- When snooze is created, schedule an Asynq task at `snooze_until` time
- Task publishes `snooze_expired` SSE event via Redis pub/sub to the user's SSE channel
- If user is not connected, event is persisted in Redis sorted set for replay on reconnect

### Default Time: 5pm in user's timezone (UTC if unknown), configurable per org.

## Research Topic 6: Alert Batching

### Decision: No batching needed — SSE events are sent individually

### Rationale
The spec says "batch them into a single SSE event or send them individually with close timestamps — the frontend handles grouping." Since the frontend handles grouping and SSE is a push model (not batched like email notifications), sending individual events is simpler and more responsive. Each `prospect_replied` event is sent as it's detected.
