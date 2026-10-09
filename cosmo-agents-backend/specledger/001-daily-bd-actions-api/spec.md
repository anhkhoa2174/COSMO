# Feature Specification: Daily BD Actions API (Backend)

**Feature Branch**: `001-daily-bd-actions-api`
**Created**: 2026-02-27
**Updated**: 2026-03-03
**Status**: In Progress
**Input**: Build backend API that is 100% compatible with the frontend Daily BD Action Suggestion module. Every response shape, field name, enum value, and streaming protocol MUST match the frontend TypeScript types and OpenAPI contract exactly.

**Frontend Contract Source**: `cosmo-agents-frontend/specledger/001-daily-bd-actions/contracts/daily-actions-api.yaml`
**Frontend Types Source**: `cosmo-agents-frontend/src/types/daily-actions.ts`
**Frontend Client Source**: `cosmo-agents-frontend/src/network/client/daily-actions.ts`

## Compatibility Contract

The backend MUST return JSON responses that exactly match the following frontend TypeScript interfaces. Any mismatch (missing field, wrong type, wrong enum value, wrong field name) will break the frontend.

### Generation Status Values

The `generation_status` field MUST be one of exactly these strings:
- `"not_generated"` — no generation has run today
- `"started"` — generation task has been enqueued (returned by POST /generate)
- `"generating"` — worker is actively processing
- `"ready"` — generation complete, results available
- `"stale"` — new events occurred since last generation

The frontend renders different UI states based on these exact values:
- `"not_generated"` → "Generate Briefing" button
- `"started"` or `"generating"` → loading spinner
- `"ready"` → full briefing with chat
- `"stale"` → briefing with refresh indicator

### Action Types

Exactly: `"outreach"`, `"followup"`, `"respond"`, `"meeting_prep"`, `"enrich"`

### Action Statuses

Exactly: `"suggested"`, `"in_progress"`, `"completed"`, `"skipped"`, `"snoozed"`, `"deferred"`

### Category IDs

Exactly: `"replied"`, `"followup"`, `"new_outreach"`, `"meeting_prep"`, `"enrichment"`

### Action Transitions

Exactly: `"mark_sent"`, `"skip"`, `"snooze"`, `"snooze_custom"`, `"mark_completed"`, `"reopen"`

### Contact Sources

Exactly: `"LinkedIn"`, `"Apollo"`, `"Manual"`, `"HubSpot"`

### Intent Assessment Values

Exactly: `"interested"`, `"requesting_info"`, `"scheduling_meeting"`, `"declining"`, `"unclear"`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Async Daily Actions Generation (Priority: P1)

When the frontend opens the Daily Actions page, it calls `POST /v1/daily-actions/generate` to trigger the AI pipeline. The backend MUST:

- Accept the request and return immediately with a `generation_id` and status (`"started"`, `"already_in_progress"`, or `"cached"`)
- Run the generation asynchronously (not blocking the HTTP response)
- Fan out to: (1) find all active contacts needing actions, (2) determine conversation state for each, (3) run priority scoring, (4) generate AI reasoning text and agent briefing with memory references, (5) generate draft messages for outreach/follow-up actions, (6) generate meeting prep briefings for upcoming meetings, (7) identify contacts with missing data for enrichment
- Persist the computed actions with unique UUIDs so they can be referenced by `PATCH /v1/daily-actions/{action_id}`
- Group actions into ordered categories: `replied`, `followup`, `new_outreach`, `meeting_prep`, `enrichment`
- Push a `generation_complete` SSE event when finished
- If a generation is already running for this user, return `"already_in_progress"` (429)
- If today's actions are already computed and fresh, return `"cached"` without re-running

**Request body** (all optional):
```
{
  "language": "vi" | "en",     // default "vi"
  "force_refresh": boolean     // default false
}
```

**Response body** (202):
```
{
  "generation_id": "uuid" | null,
  "generation_status": "started" | "already_in_progress" | "cached"
}
```

**Why this priority**: This is the foundation. Without async generation, no actions exist for the frontend to display.

**Independent Test**: Can be tested by seeding contacts in various pipeline stages, calling POST /generate, verifying a 202 response with generation_id, then polling GET /v1/daily-actions until generation_status becomes "ready" and verifying the response contains correctly categorized actions.

**Acceptance Scenarios**:

1. **Given** a BD rep has contacts in various pipeline stages, **When** POST /generate is called, **Then** the response is 202 with `generation_status: "started"`.
2. **Given** generation is in progress, **When** POST /generate is called again, **Then** the response is 429 with `generation_status: "already_in_progress"`.
3. **Given** today's actions were already generated and no new events occurred, **When** POST /generate is called, **Then** the response is 202 with `generation_status: "cached"` and `generation_id` of the existing generation.
4. **Given** generation completes, **When** the frontend listens on the SSE stream, **Then** a `generation_complete` event is pushed with the `generation_id` and `action_count`.
5. **Given** a BD rep has no active contacts, **When** POST /generate is called, **Then** generation completes with zero actions and generation_status becomes "ready" with empty categories.
6. **Given** `force_refresh: true` is sent, **When** a previous generation exists, **Then** the backend MUST regenerate, replacing the previous generation.

---

### User Story 2 - Read Daily Actions Briefing (Priority: P1)

The frontend calls `GET /v1/daily-actions` to fetch the computed briefing. The backend MUST return the full `DailyActionsBriefing` response.

**Query parameters**:
- `language` (optional, default "vi"): `"vi"` or `"en"`
- `include_completed` (optional, default "true"): whether to include completed/skipped/snoozed actions

**Response body** — MUST exactly match this shape:
```
{
  "generation_id": "uuid" | null,
  "generation_status": "not_generated" | "started" | "generating" | "ready" | "stale",
  "generated_at": "ISO8601 datetime" | null,
  "date": "YYYY-MM-DD",
  "language": "vi" | "en",
  "agent_briefing": {
    "greeting": "string",
    "strategic_reasoning": "string",
    "memory_references": [
      {
        "contact_id": "uuid",
        "contact_name": "string",
        "event_summary": "string",
        "event_timestamp": "ISO8601",
        "relevance": "string"
      }
    ],
    "category_counts": [
      {
        "category": "replied" | "followup" | "new_outreach" | "meeting_prep" | "enrichment",
        "label": "string",
        "count": number,
        "icon": "string",
        "color": "string"
      }
    ]
  },
  "categories": [
    {
      "id": "replied" | "followup" | "new_outreach" | "meeting_prep" | "enrichment",
      "label": "string",
      "icon": "string",
      "color": "string",
      "description": "string",
      "actions": [ DailyAction[] ],
      "total_count": number,
      "has_more": boolean,
      "next_offset": number | null
    }
  ],
  "pipeline_summary": {
    "total_active_contacts": number,
    "contacts_by_stage": { "COLD": number, "NO_REPLY": number, ... },
    "contacts_by_lifecycle": { "new": number, "contacted": number, ... },
    "response_rate_7d": number,
    "meetings_booked_7d": number,
    "avg_response_time_hours": number
  },
  "progress": {
    "total": number,
    "completed": number,
    "skipped": number,
    "snoozed": number,
    "remaining": number,
    "completion_rate": number
  }
}
```

**Critical fields the frontend reads** (MUST be populated when generation_status is "ready"):
- `agent_briefing.greeting` — rendered as the first text block in chat
- `agent_briefing.strategic_reasoning` — rendered in the Agent Reasoning block with bold/highlight styling
- `agent_briefing.memory_references[]` — rendered as expandable memory reference cards
- `agent_briefing.category_counts[]` — rendered as clickable summary badges (requires `category`, `label`, `count`, `icon`, AND `color`)
- `categories[].icon` — emoji string used in category header
- `categories[].color` — color name used for category header styling
- `categories[].description` — shown as category subtitle
- `pipeline_summary` — used for pipeline health indicators
- `progress.completed` and `progress.total` — rendered in the persistent progress pill: "X / Y completed"

**DailyAction shape** (each action in a category):
```
{
  "id": "uuid",
  "type": "outreach" | "followup" | "respond" | "meeting_prep" | "enrich",
  "contact": {
    "id": "uuid",
    "name": "string",
    "email": "string" | null,
    "company": "string" | null,
    "job_title": "string" | null,
    "linkedin_url": "string" | null,
    "source": "LinkedIn" | "Apollo" | "Manual" | "HubSpot",
    "status": "ready" | "pending",
    "outreach_stage": "COLD" | "NO_REPLY" | "REPLIED" | "POST_MEETING" | "DROPPED",
    "lifecycle_stage": "string" | null,
    "avatar_url": "string" | null
  },
  "priority": number,
  "priority_factors": [ { "factor": "string", "value": number, "description": "string" } ] | null,
  "reasoning": "string",
  "status": "suggested" | "in_progress" | "completed" | "skipped" | "snoozed" | "deferred",
  "status_changed_at": "ISO8601" | null,
  "snooze_until": "ISO8601" | null,
  "outreach_data": { ... } | null,
  "meeting_data": { ... } | null,
  "enrichment_data": { ... } | null,
  "respond_data": { ... } | null,
  "created_at": "ISO8601",
  "updated_at": "ISO8601"
}
```

**Type-specific data shapes** — exactly one is populated per action:

**outreach_data** (for type "outreach" or "followup"):
```
{
  "draft_message": "string",
  "scenario": "string",
  "context_level": "string",
  "company_context": "string" | null,
  "followup_number": number | null,
  "days_since_last_interaction": number,
  "previous_messages_count": number,
  "last_sent_date": "YYYY-MM-DD" | null,
  "is_final_followup": boolean,
  "outreach_state": {
    "conversation_state": "string",
    "next_step": "string",
    "followup_count": number,
    "max_followups": number
  }
}
```

**respond_data** (for type "respond"):
```
{
  "reply_preview": "string",
  "reply_timestamp": "ISO8601",
  "reply_channel": "LinkedIn" | "Email" | "Call" | "Meeting",
  "intent_assessment": "interested" | "requesting_info" | "scheduling_meeting" | "declining" | "unclear",
  "intent_reasoning": "string",
  "recommended_action": "string",
  "draft_response": "string" | null,
  "conversation_context": {
    "total_interactions": number,
    "days_in_conversation": number,
    "last_outgoing_message_preview": "string" | null,
    "key_topics_discussed": ["string"]
  }
}
```

**meeting_data** (for type "meeting_prep"):
```
{
  "meeting_id": "string",
  "meeting_title": "string",
  "meeting_time": "ISO8601",
  "meeting_duration_minutes": number,
  "meeting_channel": "string",
  "hours_until_meeting": number,
  "briefing": {
    "prospect_profile_summary": "string",
    "conversation_summary": {
      "touchpoint_count": number,
      "duration_days": number,
      "tone_assessment": "string",
      "key_topics": ["string"]
    },
    "pain_points": [
      { "pain_point": "string", "confidence": number, "evidence": ["string"] }
    ],
    "suggested_agenda": [
      { "topic": "string", "duration_minutes": number, "notes": "string" | null }
    ],
    "discovery_questions": ["string"],
    "recommended_next_steps": ["string"],
    "risk_flags": ["string"]
  }
}
```

**enrichment_data** (for type "enrich"):
```
{
  "missing_fields": ["string"],
  "quality_impact": "string",
  "contact_status": "ready" | "pending",
  "suggested_sources": [
    { "field": "string", "source": "string", "url": "string" | null }
  ]
}
```

**Why this priority**: This is the main read endpoint that powers the entire frontend UI.

**Acceptance Scenarios**:

1. **Given** no generation has run today, **When** GET /daily-actions is called, **Then** `generation_status` is `"not_generated"` with empty categories array and null agent_briefing.
2. **Given** generation is complete, **When** GET /daily-actions is called, **Then** `generation_status` is `"ready"` with populated agent_briefing (including greeting, strategic_reasoning, memory_references, category_counts with color field), categories (with icon, color, description), pipeline_summary, and progress.
3. **Given** a category has 10 actions, **When** GET /daily-actions is called, **Then** the category shows 5 actions with `has_more: true` and `next_offset: 5`.
4. **Given** `include_completed=false`, **When** GET /daily-actions is called, **Then** completed/skipped/snoozed actions are excluded from the response.
5. **Given** a prospect replied after generation, **When** GET /daily-actions is called, **Then** `generation_status` is `"stale"`.
6. **Given** generation is in progress (enqueued or running), **When** GET /daily-actions is called, **Then** `generation_status` is `"started"` or `"generating"` with empty categories.

---

### User Story 3 - Update Action State (Priority: P1)

When a BD rep clicks Mark Sent, Skip, Snooze, or other buttons on an action card, the frontend calls `PATCH /v1/daily-actions/{action_id}`.

**Request body**:
```
{
  "transition": "mark_sent" | "skip" | "snooze" | "snooze_custom" | "mark_completed" | "reopen",
  "content": "string" | null,
  "channel": "LinkedIn" | "Email" | "Call" | "Meeting" | null,
  "skip_reason": "string" | null,
  "snooze_until": "ISO8601" | null,
  "feedback_action": "used_draft" | "modified_draft" | "wrote_own" | "skipped" | null
}
```

**Response body** (200):
```
{
  "action": DailyAction,
  "contact_state_change": {
    "previous_state": "string",
    "new_state": "string",
    "new_next_step": "string"
  } | null,
  "progress": {
    "total": number,
    "completed": number,
    "skipped": number,
    "snoozed": number,
    "remaining": number,
    "completion_rate": number
  }
}
```

The backend MUST:

- For `mark_sent`: log the interaction via the outreach service, advance the contact's pipeline state (e.g., COLD → NO_REPLY), record feedback, return the `contact_state_change` with previous/new state
- For `skip`: record the skip with optional reason, no contact state change
- For `snooze`: store a snooze record with reappearance time (default 5pm user's timezone), exclude from active actions
- For `snooze_custom`: same as snooze but with a custom `snooze_until` time
- For `mark_completed`: generic completion for meeting_prep and enrichment actions
- For `reopen`: undo a skip/snooze, return the action to "suggested" status
- Return 404 if action_id not found, 409 if state conflict (e.g., already completed)
- Publish an `action_updated` SSE event after successful transition

**Why this priority**: Action execution is the core interaction loop.

**Acceptance Scenarios**:

1. **Given** an active outreach action for a COLD contact, **When** PATCH with `transition: "mark_sent"` is called, **Then** the contact advances to NO_REPLY, interaction is logged, feedback is recorded, and `contact_state_change` shows the transition.
2. **Given** an active action, **When** PATCH with `transition: "skip"` is called, **Then** the action status becomes "skipped" and progress.remaining decreases by 1.
3. **Given** an active action, **When** PATCH with `transition: "snooze"` is called, **Then** the action status becomes "snoozed" with `snooze_until` set to 5pm user's timezone and the action is excluded from active lists.
4. **Given** a snoozed action, **When** PATCH with `transition: "reopen"` is called, **Then** the action returns to "suggested" status and reappears in the active list.
5. **Given** an already-completed action, **When** PATCH with `transition: "mark_sent"` is called, **Then** the response is 409 (conflict).

---

### User Story 4 - SSE Real-Time Event Stream (Priority: P2)

The frontend opens `GET /v1/daily-actions/events` as a Server-Sent Events stream on page load using `new EventSource(url)`. The frontend parses each `event.data` as JSON.

The backend MUST:
- Set response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
- Send events in standard SSE format: `id: {event_id}\nevent: {event_type}\ndata: {json}\n\n`
- Send heartbeat comments (`: heartbeat\n\n`) every 15 seconds to keep the connection alive
- Send `retry: 3000\n\n` on connect to configure auto-reconnect interval

**Event types and payloads**:

**prospect_replied**:
```
{
  "event_type": "prospect_replied",
  "event_id": "string",
  "timestamp": "ISO8601",
  "contact": ActionContact,
  "reply_preview": "string",
  "reply_timestamp": "ISO8601",
  "reply_channel": "string",
  "intent_assessment": "interested" | "requesting_info" | "scheduling_meeting" | "declining" | "unclear",
  "agent_reasoning": "string",
  "recommended_action": "string",
  "draft_response": "string" | null,
  "action_id": "uuid"
}
```

**meeting_approaching**:
```
{
  "event_type": "meeting_approaching",
  "event_id": "string",
  "timestamp": "ISO8601",
  "meeting_id": "string",
  "meeting_title": "string",
  "meeting_time": "ISO8601",
  "contact": ActionContact,
  "hours_until": number,
  "has_prep": boolean,
  "agent_message": "string",
  "action_id": "uuid"
}
```

**followup_due**:
```
{
  "event_type": "followup_due",
  "event_id": "string",
  "timestamp": "ISO8601",
  "contact": ActionContact,
  "followup_number": number,
  "days_since": number,
  "is_final": boolean,
  "agent_message": "string",
  "action_id": "uuid"
}
```

**snooze_expired**:
```
{
  "event_type": "snooze_expired",
  "event_id": "string",
  "timestamp": "ISO8601",
  "actions": DailyAction[],
  "agent_message": "string"
}
```

**generation_complete**:
```
{
  "event_type": "generation_complete",
  "event_id": "string",
  "timestamp": "ISO8601",
  "generation_id": "uuid",
  "action_count": number
}
```

**action_updated**:
```
{
  "event_type": "action_updated",
  "event_id": "string",
  "timestamp": "ISO8601",
  "action_id": "uuid",
  "new_status": "suggested" | "in_progress" | "completed" | "skipped" | "snoozed" | "deferred",
  "updated_by": "string"
}
```

The frontend uses these events to:
- `generation_complete`, `action_updated` → invalidate the `['daily-actions']` React Query cache
- `prospect_replied`, `meeting_approaching`, `followup_due`, `snooze_expired` → render inline alert messages in chat AND invalidate cache

The stream MUST support `Last-Event-ID` header for reconnection resume — when a client reconnects with a `Last-Event-ID`, the backend MUST replay all events that were published after that ID.

**Why this priority**: Real-time updates make the briefing a "living document".

**Acceptance Scenarios**:

1. **Given** a prospect replies to outreach while the BD rep has an open SSE connection, **When** the notification is processed, **Then** a `prospect_replied` event is pushed with intent assessment and recommended action.
2. **Given** a meeting is 2 hours away, **When** the scheduled check runs, **Then** a `meeting_approaching` event is pushed with meeting details and prep status.
3. **Given** an action was snoozed until 5pm, **When** 5pm arrives, **Then** a `snooze_expired` event is pushed with the resurfaced actions as full DailyAction objects.
4. **Given** the SSE connection drops and reconnects with `Last-Event-ID`, **When** events were missed, **Then** the missed events are replayed in order.
5. **Given** an action is updated via PATCH, **Then** an `action_updated` event is pushed to all other SSE connections for that user.

---

### User Story 5 - Category Pagination (Priority: P2)

When a category has more than 5 actions, the frontend calls `GET /v1/daily-actions/categories/{category_id}/actions?offset=N&limit=5`.

**Path parameters**:
- `category_id`: one of `replied`, `followup`, `new_outreach`, `meeting_prep`, `enrichment`

**Query parameters**:
- `offset` (required): 0-based offset
- `limit` (optional, default 5, max 20): batch size

**Response body** (200):
```
{
  "actions": DailyAction[],
  "has_more": boolean,
  "next_offset": number | null
}
```

**Why this priority**: Scales the UI to 200+ contacts without performance degradation.

**Acceptance Scenarios**:

1. **Given** 12 follow-up actions exist, **When** GET /categories/followup/actions?offset=5&limit=5 is called, **Then** actions 6-10 are returned with `has_more: true` and `next_offset: 10`.
2. **Given** 12 follow-up actions exist, **When** GET /categories/followup/actions?offset=10&limit=5 is called, **Then** actions 11-12 are returned with `has_more: false`.

---

### User Story 6 - Conversational Chat with BD Agent (Priority: P2)

The frontend routes chat messages through a Next.js API route (`/api/ai/daily-actions`) which proxies to the backend `POST /v1/daily-actions/chat`. The frontend uses the Vercel AI SDK's `useChat` hook which expects the response to follow the Vercel AI SDK data stream protocol.

**Request body**:
```
{
  "messages": [
    { "role": "user" | "assistant" | "system", "content": "string" }
  ],
  "stream": boolean | null
}
```

**Response** — MUST be `Content-Type: text/event-stream` following the Vercel AI SDK data stream protocol:
- Text chunks: `0:"chunk text"\n`
- Finish reason: `e:{"finishReason":"stop","usage":{"promptTokens":N,"completionTokens":N}}\n`
- Done signal: `d:{"finishReason":"stop"}\n`

The backend MUST:
- Accept the Vercel AI SDK message format
- Stream the response using the Vercel AI SDK data protocol (not plain JSON)
- Support: action commands (mark sent, regenerate draft, update contact), pipeline queries (reply rate, contact filters), strategy advice (approach recommendations), analytics (summaries, performance metrics), and contact lookups
- Reference relevant pipeline data and conversation context in responses
- When entity is ambiguous (e.g., multiple contacts named "Bao"), return disambiguation options

**Why this priority**: Conversational capability transforms the chat from one-directional to interactive.

**Acceptance Scenarios**:

1. **Given** a BD rep sends "Reply rate tuan nay bao nhieu?", **When** the chat endpoint processes it, **Then** a streamed response includes the actual reply rate metric with trend context.
2. **Given** a BD rep pastes a prospect's reply, **When** processed, **Then** the agent classifies the intent and suggests a response strategy with optional draft.
3. **Given** a BD rep sends "regenerate draft for Bao" and 2 contacts named "Bao" exist, **When** processed, **Then** the response asks for disambiguation with contact details.
4. **Given** the frontend sends a message, **When** the backend responds, **Then** the response uses the Vercel AI SDK streaming protocol so `useChat` can parse it correctly.

---

### User Story 7 - Daily Summary (Priority: P3)

The frontend calls `GET /v1/daily-actions/summary` for end-of-day wrap-up.

**Query parameters**:
- `language` (optional): `"vi"` or `"en"`

**Response body** (200):
```
{
  "date": "YYYY-MM-DD",
  "agent_summary": "string",
  "progress": {
    "total": number,
    "completed": number,
    "skipped": number,
    "snoozed": number,
    "remaining": number,
    "completion_rate": number
  },
  "breakdown": {
    "outreach_sent": number,
    "followups_sent": number,
    "replies_handled": number,
    "meetings_prepped": number,
    "contacts_enriched": number
  },
  "outcomes": {
    "responses_received_today": number,
    "meetings_booked_today": number,
    "contacts_dropped": number
  },
  "carry_over": [
    {
      "action_id": "uuid",
      "type": "string",
      "contact_name": "string",
      "reason": "string"
    }
  ]
}
```

**Why this priority**: Completion tracking provides accountability metrics.

**Acceptance Scenarios**:

1. **Given** 10 of 15 actions completed today, **When** GET /summary is called, **Then** progress shows 10 completed, 5 remaining, with breakdown by type.
2. **Given** 3 actions were snoozed past their reappearance time, **When** GET /summary is called, **Then** carry_over includes those 3 with reason "snoozed".

---

### Edge Cases

- What happens when a contact has conflicting signals (replied positively but marked "dropped")? The backend MUST surface the conflict and not auto-resolve — include a flag so the frontend can prompt the BD rep.
- What happens when the BD rep has no contacts? The backend MUST return generation_status "ready" with empty categories so the frontend can show the onboarding message: "Your pipeline is currently empty."
- What happens when draft generation fails (LLM timeout)? The action MUST still be included without a draft; `outreach_data.draft_message` is empty string and the frontend handles the missing draft.
- What happens when timezone data is missing for snooze? Default to UTC.
- What happens when a contact's state changes between generation and action execution? The PATCH endpoint MUST re-validate and return 409 with current vs expected state if they diverge.
- What happens when multiple prospect responses arrive within 60 seconds? The backend sends individual SSE events — the frontend handles display grouping.
- What happens when the SSE connection drops? The backend MUST support Last-Event-ID for replay of missed events. Events MUST be persisted for at least 24 hours.
- What happens when force_refresh=true is sent? The backend MUST regenerate even if today's actions exist, marking the previous generation as replaced.
- What happens when categories[] would be empty for a "ready" generation? Return `categories: []` (empty array, not null). The frontend checks `categories.length === 0` to show the empty pipeline message.

## Requirements *(mandatory)*

### Functional Requirements

**Async Generation Pipeline**

- **FR-001**: `POST /v1/daily-actions/generate` MUST return immediately (202) with a `generation_id` and trigger asynchronous generation via the background task queue.
- **FR-002**: Generation MUST produce persisted `DailyAction` records with unique UUIDs that survive across GET requests and can be referenced by PATCH.
- **FR-003**: Generation MUST categorize actions into 5 ordered categories: `replied`, `followup`, `new_outreach`, `meeting_prep`, `enrichment`. Each category MUST include `id`, `label`, `icon` (emoji), `color` (CSS color name), and `description`.
- **FR-004**: The priority scoring algorithm MUST weight: (1) response recency, (2) follow-up cadence deadline, (3) meeting proximity within 48h, (4) data completeness, (5) lifecycle stage importance. Score range 1-100 (1 = highest priority).
- **FR-005**: Generation MUST produce an `AgentBriefing` with AI-generated greeting, strategic_reasoning narrative, memory_references array, and category_counts array. Each category_count MUST include `category`, `label`, `count`, `icon`, and `color`.
- **FR-006**: If generation is already in progress, MUST return 429 with `generation_status: "already_in_progress"`. If results are cached and fresh, MUST return 202 with `generation_status: "cached"` and the existing `generation_id`.
- **FR-007**: Generation MUST detect when results become stale (new events since last generation) and reflect this in `generation_status: "stale"` on subsequent GET requests.

**Read Briefing**

- **FR-008**: `GET /v1/daily-actions` MUST return the full `DailyActionsBriefing` structure from cache/DB (no AI computation on read). MUST include `generated_at` timestamp when generation_status is "ready".
- **FR-009**: Each category MUST return at most 5 actions initially, with `has_more` and `next_offset` for pagination. Categories with zero actions MUST be omitted.
- **FR-010**: `GET /v1/daily-actions/categories/{category_id}/actions` MUST support offset-based pagination (max limit 20) for loading more actions within a category.
- **FR-010a**: The briefing response MUST include a populated `pipeline_summary` object with `total_active_contacts`, `contacts_by_stage`, `contacts_by_lifecycle`, `response_rate_7d`, `meetings_booked_7d`, and `avg_response_time_hours` when generation_status is "ready".
- **FR-010b**: The `progress` object MUST always include `completion_rate` as a percentage (0-100).

**Action State Updates**

- **FR-011**: `PATCH /v1/daily-actions/{action_id}` MUST support transitions: `mark_sent`, `skip`, `snooze`, `snooze_custom`, `mark_completed`, `reopen`.
- **FR-012**: `mark_sent` MUST: log the interaction, advance the contact's outreach state via the existing state machine, record feedback (used_draft/modified_draft/wrote_own/skipped), and return `contact_state_change`.
- **FR-013**: `snooze` MUST store a reappearance time (default 5pm in user's timezone, UTC if unknown) and exclude the action from active lists until that time.
- **FR-014**: `reopen` MUST undo a skip or snooze, returning the action to "suggested" status and clearing any snooze record.
- **FR-015**: PATCH MUST return 409 if the action's current status prevents the requested transition.
- **FR-015a**: After successful transition, PATCH MUST publish an `action_updated` SSE event to all active SSE connections for the user.

**SSE Real-Time Events**

- **FR-016**: `GET /v1/daily-actions/events` MUST open a Server-Sent Events stream for the authenticated user with proper SSE headers.
- **FR-017**: The backend MUST push `prospect_replied` events when incoming responses are detected, including intent classification and recommended action.
- **FR-018**: The backend MUST push `meeting_approaching` events 2 hours before scheduled meetings, `followup_due` when cadence deadlines are reached, and `snooze_expired` at snooze reappearance times.
- **FR-019**: The backend MUST push `generation_complete` when async generation finishes and `action_updated` when an action is modified.
- **FR-020**: SSE MUST support `Last-Event-ID` header for reconnection and replay of missed events. Events MUST be persisted with monotonically increasing IDs for replay.
- **FR-020a**: SSE MUST send heartbeat comments every 15 seconds to prevent proxy/CDN timeouts.
- **FR-020b**: SSE MUST send `retry: 3000` on initial connection to configure the client's auto-reconnect interval.

**Conversational Chat**

- **FR-021**: `POST /v1/daily-actions/chat` MUST accept a `messages[]` array (Vercel AI SDK format) and return a streamed response following the Vercel AI SDK data stream protocol (`Content-Type: text/event-stream` with `0:`, `e:`, `d:` prefixed lines).
- **FR-022**: The chat endpoint MUST classify user intent (action command, pipeline query, strategy query, analytics request) and route to appropriate handlers.
- **FR-023**: When entities are ambiguous, the response MUST include disambiguation options with identifying contact details.

**Daily Summary**

- **FR-024**: `GET /v1/daily-actions/summary` MUST return completion stats, breakdown by action type, outcomes, and carry-over items.
- **FR-025**: The summary MUST include an AI-generated `agent_summary` narrative.

**Language & Timezone**

- **FR-026**: All AI-generated content (reasoning, drafts, briefings, summaries) MUST support Vietnamese and English based on the `language` parameter.
- **FR-027**: Snooze reappearance time MUST default to 5pm in the BD rep's timezone. If timezone is unknown, use UTC.

### Key Entities

- **DailyAction**: A persisted action suggestion with UUID, type (outreach/followup/respond/meeting_prep/enrich), embedded contact snapshot, priority score (1-100, 1=highest), reasoning text, status lifecycle (suggested → in_progress → completed/skipped/snoozed/deferred), type-specific data (outreach_data/respond_data/meeting_data/enrichment_data), and timestamps (created_at, updated_at, status_changed_at).
- **DailyActionGeneration**: Tracks each generation run with generation_id, user_id, status (started/generating/ready/stale), generated_at timestamp, action_count, language, and agent_briefing JSONB (greeting, strategic_reasoning, memory_references, category_counts).
- **ActionSnooze**: Persisted snooze record with user_id, action_id, snoozed_until timestamp, and cleared_at.
- **ActionCompletionLog**: Append-only log of action executions (mark_sent, skip, snooze, mark_completed, reopen) with action_type, contact info, content, channel, skip_reason, feedback, date, and timestamp for daily summary computation.
- **SSEEvent**: Persisted event record with event_id (monotonically increasing), event_type, user_id, payload JSON, and timestamp. Used for Last-Event-ID replay support. Events older than 24 hours may be pruned.
- **ChatMessage**: Conversational message record with user_id, role, content, optional generation_id reference, classified intent, and timestamp.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The generation trigger responds instantly — BD reps see acknowledgment within 1 second of clicking "Generate Briefing". Full generation completes within 30 seconds for up to 200 active contacts.
- **SC-002**: The briefing loads instantly — BD reps see the full briefing within 1 second of the page loading (when generation is already complete).
- **SC-003**: Action updates are instant — BD reps see the card transition and progress update within 1 second of clicking an action button.
- **SC-004**: Real-time alerts arrive promptly — prospect reply alerts appear in the chat within 30 seconds of detection.
- **SC-005**: Chat responses begin immediately — BD reps see the first word of the agent's response within 2 seconds of sending a message.
- **SC-006**: Daily summaries load instantly — the summary is available within 1 second.
- **SC-007**: The system supports 50 concurrent BD reps without any degradation in response times.
- **SC-008**: Category pagination is seamless — additional actions load within 500ms.
- **SC-009**: Snooze reminders are reliable — 100% of snoozed actions reappear at the configured time.
- **SC-010**: No events are lost on reconnect — when the SSE connection drops and resumes, all missed events are replayed.

### Previous work

The following existing backend infrastructure is directly leveraged:

- **Outreach Service** (`internal/service/outreach/`): `SuggestContacts()`, `GenerateDraftWithLanguage()`, `DetermineConversationState()`, `UpdateOutreach()`, `RecordAction()`, `GenerateMeetingPrep()` — all reused for action generation and execution.
- **Outreach API** (`/v1/outreach/*`): Existing endpoints for suggestions, drafts, state updates, meetings, interactions, and feedback. The new daily-actions API orchestrates these services.
- **Intent Classifier** (`internal/service/intent/`): Email intent classification used for `prospect_replied` events.
- **Intelligence Service** (`internal/service/intelligence/`): Contact enrichment and AI insights used for reasoning text and meeting prep.
- **Gmail/Outlook Notification Pipeline** (`internal/service/gmail/`, `internal/worker/gmailnotification/`): Existing push notification handling used for real-time response detection → `prospect_replied` SSE events.
- **Contact Orchestrator** (`internal/worker/orchestrator/`): Chains enrichment, scoring, and playbook evaluation.
- **Asynq Task Queue** (`pkg/worker/`): Existing background task infrastructure used for async generation.

## Dependencies & Assumptions

### Dependencies

- The frontend contract at `contracts/daily-actions-api.yaml` and `contracts/daily-actions-client.ts` (copied from `cosmo-agents-frontend`) is the source of truth for all request/response shapes.
- Relies on the existing outreach state machine and `DetermineConversationState()` for action categorization.
- Relies on the existing draft generation pipeline (`GenerateDraftWithLanguage`) for message generation.
- Relies on the existing Gmail/Outlook notification pipeline for prospect response detection.
- Relies on the existing intent classification service for response intent analysis.
- Relies on the existing meeting management system for meeting prep data.
- Relies on OpenAI API for agent briefing, reasoning text, and chat responses.

### Assumptions

- Actions are **persisted** (not computed on-demand) so they have stable UUIDs for frontend PATCH calls. They are regenerated daily via POST /generate.
- The async generation model uses a background task queue. The frontend triggers POST /generate on page open, then reads results via GET (polling every 30 seconds) and SSE (for instant notification).
- SSE uses Fiber's streaming response. Each user has one active SSE connection at a time.
- The chat endpoint streams responses using the Vercel AI SDK data protocol (text/event-stream with `0:`, `e:`, `d:` prefixed lines). The frontend proxies through a Next.js API route `/api/ai/daily-actions` which forwards to the backend.
- Follow-up cadence defaults (Day 4-5 for first, Day 9-12 for second) match the existing outreach service config.
- Priority score is 1-100 where 1 = highest priority. This matches the frontend contract.
- Language support covers Vietnamese and English for all generated content. Default is Vietnamese (`"vi"`).
- The module is designed for individual BD reps viewing their own pipeline. Team-level aggregation is out of scope.
- End of business day for snooze reappearance is 5pm in the BD rep's local timezone, configurable per organization.
- The frontend renders category headers with emoji icons and color names — the backend MUST provide these in both `category_counts` and `categories[]` responses.
