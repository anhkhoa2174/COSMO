# Data Model: Daily BD Actions API

**Branch**: `001-daily-bd-actions-api` | **Date**: 2026-03-02
**Frontend Contract**: `contracts/daily-actions-api.yaml`

## Overview

Six new domain models support the Daily BD Actions feature. All models embed `base.Base` (UUID PK), `base.TimestampMixin`, and `base.SoftDeleteMixin` per constitution. All include `UserID` for multi-tenant isolation.

## Entity 1: DailyActionGeneration

Tracks each generation run. Created by `POST /v1/daily-actions/generate`, referenced by `GET /v1/daily-actions`.

### Go Domain Model

```go
package daily_action

import (
    "time"
    "github.com/google/uuid"
    "github.com/rockship/cosmo-agents-go/internal/domain/base"
)

type GenerationStatus string

const (
    GenerationStatusStarted    GenerationStatus = "started"
    GenerationStatusGenerating GenerationStatus = "generating"
    GenerationStatusReady      GenerationStatus = "ready"
    GenerationStatusStale      GenerationStatus = "stale"
)

type DailyActionGeneration struct {
    base.Base
    base.TimestampMixin
    base.SoftDeleteMixin

    UserID       uuid.UUID        `gorm:"type:uuid;not null;index:idx_gen_user_date" json:"user_id"`
    Date         string           `gorm:"type:date;not null;index:idx_gen_user_date" json:"date"` // YYYY-MM-DD
    Status       GenerationStatus `gorm:"type:varchar(20);not null;default:'started'" json:"status"`
    Language     string           `gorm:"type:varchar(10);default:'vi'" json:"language"`
    ActionCount  int              `gorm:"default:0" json:"action_count"`
    GeneratedAt  *time.Time       `json:"generated_at"`
    ReplacedByID *uuid.UUID       `gorm:"type:uuid" json:"replaced_by_id"` // Points to newer generation on force_refresh

    // AI-generated briefing content (stored as JSONB)
    AgentBriefing base.JSONB `gorm:"type:jsonb;default:'{}'" json:"agent_briefing"`

    // Pipeline summary snapshot
    PipelineSummary base.JSONB `gorm:"type:jsonb;default:'{}'" json:"pipeline_summary"`
}

func (DailyActionGeneration) TableName() string {
    return "daily_action_generations"
}
```

### AgentBriefing JSONB Structure
```json
{
  "greeting": "Good morning, Anh! ...",
  "strategic_reasoning": "Today you have 3 hot replies...",
  "memory_references": [
    {
      "contact_id": "uuid",
      "contact_name": "Bao Nguyen",
      "event_summary": "Replied positively yesterday",
      "event_timestamp": "2026-03-01T14:30:00Z",
      "relevance": "High-priority follow-up opportunity"
    }
  ],
  "category_counts": [
    {"category": "replied", "label": "Replied", "count": 3, "icon": "💬", "color": "green"},
    {"category": "followup", "label": "Follow-ups", "count": 5, "icon": "🔄", "color": "orange"},
    {"category": "new_outreach", "label": "New Outreach", "count": 8, "icon": "🚀", "color": "blue"},
    {"category": "meeting_prep", "label": "Meeting Prep", "count": 1, "icon": "📋", "color": "purple"},
    {"category": "enrichment", "label": "Enrichment", "count": 2, "icon": "🔍", "color": "red"}
  ]
}
```

### Category Metadata (icon, color, description)

Each category in the `categories[]` response MUST include these fields for frontend rendering:

| Category ID | Icon | Color | Description |
|-------------|------|-------|-------------|
| `replied` | 💬 | green | Prospects who replied — need immediate action |
| `followup` | ⏰ | orange | Follow-up cadence deadlines reached |
| `new_outreach` | 📤 | blue | Fresh contacts ready for first outreach |
| `meeting_prep` | 📅 | purple | Upcoming meetings requiring preparation |
| `enrichment` | 🔍 | red | Contacts with missing data blocking outreach |
```

## Entity 2: DailyAction

A persisted action suggestion with UUID. Created during generation, updated via `PATCH /v1/daily-actions/{action_id}`.

### Go Domain Model

```go
type ActionType string

const (
    ActionTypeOutreach    ActionType = "outreach"
    ActionTypeFollowup    ActionType = "followup"
    ActionTypeRespond     ActionType = "respond"
    ActionTypeMeetingPrep ActionType = "meeting_prep"
    ActionTypeEnrich      ActionType = "enrich"
)

type ActionStatus string

const (
    ActionStatusSuggested  ActionStatus = "suggested"
    ActionStatusInProgress ActionStatus = "in_progress"
    ActionStatusCompleted  ActionStatus = "completed"
    ActionStatusSkipped    ActionStatus = "skipped"
    ActionStatusSnoozed    ActionStatus = "snoozed"
    ActionStatusDeferred   ActionStatus = "deferred"
)

type CategoryID string

const (
    CategoryReplied      CategoryID = "replied"
    CategoryFollowup     CategoryID = "followup"
    CategoryNewOutreach  CategoryID = "new_outreach"
    CategoryMeetingPrep  CategoryID = "meeting_prep"
    CategoryEnrichment   CategoryID = "enrichment"
)

type DailyAction struct {
    base.Base
    base.TimestampMixin
    base.SoftDeleteMixin

    UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_action_user_gen" json:"user_id"`
    GenerationID uuid.UUID  `gorm:"type:uuid;not null;index:idx_action_user_gen" json:"generation_id"`
    ContactID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_action_contact" json:"contact_id"`
    Type         ActionType `gorm:"type:varchar(20);not null;index:idx_action_category" json:"type"`
    CategoryID   CategoryID `gorm:"type:varchar(20);not null;index:idx_action_category" json:"category_id"`

    // Priority (1=highest, 100=lowest)
    Priority        int        `gorm:"not null;index:idx_action_priority" json:"priority"`
    PriorityFactors base.JSONB `gorm:"type:jsonb;default:'[]'" json:"priority_factors"`

    // AI-generated reasoning
    Reasoning string `gorm:"type:text" json:"reasoning"`

    // Status lifecycle
    Status          ActionStatus `gorm:"type:varchar(20);not null;default:'suggested'" json:"status"`
    StatusChangedAt *time.Time   `json:"status_changed_at"`
    SnoozeUntil     *time.Time   `json:"snooze_until"`

    // Type-specific data (JSONB — only one is populated per action)
    OutreachData    base.JSONB `gorm:"type:jsonb" json:"outreach_data,omitempty"`
    MeetingData     base.JSONB `gorm:"type:jsonb" json:"meeting_data,omitempty"`
    EnrichmentData  base.JSONB `gorm:"type:jsonb" json:"enrichment_data,omitempty"`
    RespondData     base.JSONB `gorm:"type:jsonb" json:"respond_data,omitempty"`

    // Embedded contact snapshot (JSONB — denormalized for fast reads)
    ContactSnapshot base.JSONB `gorm:"type:jsonb;not null" json:"contact"`
}

func (DailyAction) TableName() string {
    return "daily_actions"
}
```

### State Machine: Action Status Transitions

```
suggested → in_progress (mark_sent begins)
suggested → skipped     (skip)
suggested → snoozed     (snooze, snooze_custom)
suggested → completed   (mark_completed — for meeting_prep, enrich)

in_progress → completed (mark_sent completes)

skipped → suggested     (reopen)
snoozed → suggested     (reopen, or snooze_until expires)
completed → suggested   (reopen)
```

### Valid Transitions Table

| From Status | Allowed Transitions |
|-------------|-------------------|
| suggested | mark_sent, skip, snooze, snooze_custom, mark_completed |
| in_progress | mark_completed |
| skipped | reopen |
| snoozed | reopen |
| completed | reopen |
| deferred | reopen |

### Type-Specific Data Structures

**OutreachActionData** (outreach, followup types):
```json
{
  "draft_message": "Hi Bao, ...",
  "scenario": "no_reply_followup",
  "context_level": "HIGH",
  "company_context": "TechCorp is expanding...",
  "followup_number": 1,
  "days_since_last_interaction": 5,
  "previous_messages_count": 2,
  "last_sent_date": "2026-02-25",
  "is_final_followup": false,
  "outreach_state": {
    "conversation_state": "NO_REPLY",
    "next_step": "FOLLOW_UP_1",
    "followup_count": 0,
    "max_followups": 2
  }
}
```

**RespondActionData** (respond type):
```json
{
  "reply_preview": "Thanks for reaching out...",
  "reply_timestamp": "2026-03-01T14:30:00Z",
  "reply_channel": "Email",
  "intent_assessment": "interested",
  "intent_reasoning": "Prospect asked about pricing...",
  "recommended_action": "Schedule a demo call",
  "draft_response": "Great to hear your interest...",
  "conversation_context": {
    "total_interactions": 5,
    "days_in_conversation": 12,
    "last_outgoing_message_preview": "Just checking in...",
    "key_topics_discussed": ["pricing", "timeline"]
  }
}
```

**MeetingActionData** (meeting_prep type):
```json
{
  "meeting_id": "uuid",
  "meeting_title": "Demo with TechCorp",
  "meeting_time": "2026-03-02T14:00:00Z",
  "meeting_duration_minutes": 30,
  "meeting_channel": "Zoom",
  "hours_until_meeting": 4.5,
  "briefing": {
    "prospect_profile_summary": "...",
    "conversation_summary": { "touchpoint_count": 5, "duration_days": 12, "tone_assessment": "positive", "key_topics": ["pricing"] },
    "pain_points": [{ "pain_point": "...", "confidence": 0.8, "evidence": ["..."] }],
    "suggested_agenda": [{ "topic": "...", "duration_minutes": 10 }],
    "discovery_questions": ["..."],
    "recommended_next_steps": ["..."],
    "risk_flags": ["..."]
  }
}
```

**EnrichmentActionData** (enrich type):
```json
{
  "missing_fields": ["email", "job_title"],
  "quality_impact": "Cannot send personalized outreach without job title",
  "contact_status": "pending",
  "suggested_sources": [
    { "field": "email", "source": "Apollo", "url": "https://..." },
    { "field": "job_title", "source": "LinkedIn", "url": "https://..." }
  ]
}
```

### Contact Snapshot Structure (denormalized in each action)
```json
{
  "id": "uuid",
  "name": "Bao Nguyen",
  "email": "bao@techcorp.com",
  "company": "TechCorp",
  "job_title": "CTO",
  "linkedin_url": "https://linkedin.com/in/bao",
  "source": "Apollo",
  "status": "ready",
  "outreach_stage": "NO_REPLY",
  "lifecycle_stage": "PRE_SALES",
  "avatar_url": null
}
```

## Entity 3: ActionSnooze

Tracks snooze records. Created on `snooze`/`snooze_custom` transitions.

```go
type ActionSnooze struct {
    base.Base
    base.TimestampMixin

    UserID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_snooze_user" json:"user_id"`
    ActionID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_snooze_action" json:"action_id"`
    SnoozeUntil time.Time `gorm:"not null" json:"snooze_until"`
    ClearedAt  *time.Time `json:"cleared_at"` // Set on reopen (semantic soft-clear)
}

func (ActionSnooze) TableName() string {
    return "action_snoozes"
}
```

## Entity 4: ActionCompletionLog

Append-only audit log of action state changes. Used for daily summary computation.

```go
type ActionCompletionLog struct {
    base.Base
    base.TimestampMixin

    UserID       uuid.UUID    `gorm:"type:uuid;not null;index:idx_log_user_date" json:"user_id"`
    ActionID     uuid.UUID    `gorm:"type:uuid;not null" json:"action_id"`
    ActionType   ActionType   `gorm:"type:varchar(20);not null" json:"action_type"`
    Transition   string       `gorm:"type:varchar(20);not null" json:"transition"` // mark_sent, skip, snooze, reopen, mark_completed
    ContactID    uuid.UUID    `gorm:"type:uuid;not null" json:"contact_id"`
    ContactName  string       `gorm:"type:varchar(255)" json:"contact_name"`
    Content      *string      `gorm:"type:text" json:"content,omitempty"` // Message content for mark_sent
    Channel      *string      `gorm:"type:varchar(50)" json:"channel,omitempty"`
    SkipReason   *string      `gorm:"type:text" json:"skip_reason,omitempty"`
    Feedback     *string      `gorm:"type:varchar(50)" json:"feedback,omitempty"` // used_draft, modified_draft, wrote_own
    Date         string       `gorm:"type:date;not null;index:idx_log_user_date" json:"date"` // YYYY-MM-DD for summary queries
}

func (ActionCompletionLog) TableName() string {
    return "action_completion_logs"
}
```

## Entity 5: SSEEvent

Transient event record for Last-Event-ID replay support.

```go
type SSEEvent struct {
    base.Base
    base.TimestampMixin

    UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_sse_user_time" json:"user_id"`
    EventType string    `gorm:"type:varchar(30);not null" json:"event_type"`
    Payload   base.JSONB `gorm:"type:jsonb;not null" json:"payload"`
}

func (SSEEvent) TableName() string {
    return "sse_events"
}
```

**Note**: SSE events are primarily stored in Redis sorted sets (5-min TTL) for hot replay. The PostgreSQL table serves as fallback and audit trail. Events older than 24 hours can be cleaned up.

## Entity 6: ChatMessage

Conversational request/response pairs for the BD Agent chat.

```go
type ChatMessage struct {
    base.Base
    base.TimestampMixin

    UserID         uuid.UUID `gorm:"type:uuid;not null;index:idx_chat_user" json:"user_id"`
    Role           string    `gorm:"type:varchar(20);not null" json:"role"` // user, assistant, system
    Content        string    `gorm:"type:text;not null" json:"content"`
    ClassifiedIntent *string `gorm:"type:varchar(50)" json:"classified_intent,omitempty"` // action_command, pipeline_query, strategy_query, analytics_request
    GenerationID   *uuid.UUID `gorm:"type:uuid" json:"generation_id,omitempty"` // Links to daily action context
}

func (ChatMessage) TableName() string {
    return "chat_messages"
}
```

## Relationships

```
DailyActionGeneration 1 ──── N DailyAction (generation_id FK)
DailyAction 1 ──── N ActionSnooze (action_id FK)
DailyAction 1 ──── N ActionCompletionLog (action_id FK)
User 1 ──── N DailyActionGeneration (user_id FK)
User 1 ──── N DailyAction (user_id FK)
User 1 ──── N ChatMessage (user_id FK)
User 1 ──── N SSEEvent (user_id FK)
Contact 1 ──── N DailyAction (contact_id FK)
```

## SQL Migration: 000048_create_daily_action_tables.up.sql

```sql
-- Daily Action Generations
CREATE TABLE IF NOT EXISTS daily_action_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'started',
    language VARCHAR(10) DEFAULT 'vi',
    action_count INTEGER DEFAULT 0,
    generated_at TIMESTAMPTZ,
    replaced_by_id UUID,
    agent_briefing JSONB DEFAULT '{}',
    pipeline_summary JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE UNIQUE INDEX idx_gen_user_date ON daily_action_generations(user_id, date) WHERE is_deleted = FALSE AND replaced_by_id IS NULL;

-- Daily Actions
CREATE TABLE IF NOT EXISTS daily_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    generation_id UUID NOT NULL REFERENCES daily_action_generations(id),
    contact_id UUID NOT NULL REFERENCES contacts(id),
    type VARCHAR(20) NOT NULL,
    category_id VARCHAR(20) NOT NULL,
    priority INTEGER NOT NULL,
    priority_factors JSONB DEFAULT '[]',
    reasoning TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'suggested',
    status_changed_at TIMESTAMPTZ,
    snooze_until TIMESTAMPTZ,
    outreach_data JSONB,
    meeting_data JSONB,
    enrichment_data JSONB,
    respond_data JSONB,
    contact JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_action_user_gen ON daily_actions(user_id, generation_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_action_category ON daily_actions(category_id, priority) WHERE is_deleted = FALSE;
CREATE INDEX idx_action_contact ON daily_actions(contact_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_action_priority ON daily_actions(user_id, priority) WHERE is_deleted = FALSE AND status = 'suggested';

-- Action Snoozes
CREATE TABLE IF NOT EXISTS action_snoozes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    action_id UUID NOT NULL REFERENCES daily_actions(id),
    snooze_until TIMESTAMPTZ NOT NULL,
    cleared_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_snooze_user ON action_snoozes(user_id) WHERE cleared_at IS NULL;
CREATE INDEX idx_snooze_action ON action_snoozes(action_id) WHERE cleared_at IS NULL;

-- Action Completion Logs (append-only)
CREATE TABLE IF NOT EXISTS action_completion_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    action_id UUID NOT NULL REFERENCES daily_actions(id),
    action_type VARCHAR(20) NOT NULL,
    transition VARCHAR(20) NOT NULL,
    contact_id UUID NOT NULL,
    contact_name VARCHAR(255),
    content TEXT,
    channel VARCHAR(50),
    skip_reason TEXT,
    feedback VARCHAR(50),
    date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_log_user_date ON action_completion_logs(user_id, date);

-- SSE Events (transient, for replay support)
CREATE TABLE IF NOT EXISTS sse_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_type VARCHAR(30) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sse_user_time ON sse_events(user_id, created_at);

-- Chat Messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    classified_intent VARCHAR(50),
    generation_id UUID REFERENCES daily_action_generations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chat_user ON chat_messages(user_id, created_at);
```

## SQL Migration: 000048_create_daily_action_tables.down.sql

```sql
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS sse_events;
DROP TABLE IF EXISTS action_completion_logs;
DROP TABLE IF EXISTS action_snoozes;
DROP TABLE IF EXISTS daily_actions;
DROP TABLE IF EXISTS daily_action_generations;
```
