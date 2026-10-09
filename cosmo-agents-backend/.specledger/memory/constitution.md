<!--
  Sync Impact Report
  ==================
  Version change: 0.0.0 → 1.0.0 (initial ratification)
  Modified principles: N/A (first version)
  Added sections:
    - 6 Core Principles (I–VI)
    - Technology Stack & Constraints
    - Development Workflow
    - Agent Preferences
    - Governance
  Removed sections: None
  Templates requiring updates:
    - .specledger/templates/plan-template.md ✅ compatible (Constitution Check aligns)
    - .specledger/templates/spec-template.md ✅ compatible (user stories + requirements)
    - .specledger/templates/tasks-template.md ✅ compatible (phase structure aligns)
  Follow-up TODOs: None
-->

# Cosmo Agents Backend Constitution

## Core Principles

### I. Clean Architecture (NON-NEGOTIABLE)

All code MUST follow the established layered architecture with strict
dependency direction: `domain → repository → service → handler → cmd`.

- **Domain layer** (`internal/domain/`): Pure business entities and
  interfaces. MUST NOT import infrastructure or framework packages.
- **Repository layer** (`internal/repository/`): Data access via the
  generic `BaseRepository[T]` interface and `GormRepository[T]`
  implementation. New repositories MUST embed the generic base.
- **Service layer** (`internal/service/`): Business logic orchestration.
  Services receive repositories and clients via constructor injection.
- **Handler layer** (`internal/handler/`): HTTP request handling only.
  Handlers MUST delegate business logic to services, never implement
  it directly. Use `ResponseHelper` for standardized responses.
- **Package layer** (`pkg/`): Reusable infrastructure (database, cache,
  auth, integrations). MUST NOT import `internal/` packages.
- Dependencies MUST be injected via struct constructors in
  `cmd/server/dependencies.go`. No global state or service locators.

### II. Multi-Tenant Data Isolation (NON-NEGOTIABLE)

Every domain entity MUST include a `UserID uuid.UUID` field. All
repository queries MUST scope data by the authenticated user's ID
unless explicitly documented as cross-tenant (e.g., admin operations).

- All database queries MUST filter by `UserID` (or `OrganizationID`
  where applicable) to prevent data leakage between tenants.
- Handler methods MUST extract user context via `AuthHelper` methods
  (`GetUserID`, `GetUserAndOrganization`, `RequireAuthentication`).
- New domain models MUST embed `base.Base`, `base.TimestampMixin`,
  and `base.SoftDeleteMixin`.

### III. Async-First Processing

Long-running or resource-intensive operations MUST be processed
asynchronously via background workers, not in HTTP request handlers.

- **Asynq task queue** (`pkg/worker/`): Use for discrete background
  tasks (email sending, AI generation, indexing, enrichment).
  Tasks MUST be registered in `cmd/worker/main.go` with appropriate
  queue priority (critical: 60%, default: 30%, low: 10%).
- **Temporal workflows** (`internal/temporal/`): Use for complex,
  multi-step processes requiring retry semantics and state tracking
  (batch enrichment, full contact analysis).
- Worker payloads MUST be JSON-serializable structs defined in
  `internal/worker/dto/`.
- Workers MUST use context timeouts appropriate to the task
  (2 minutes for short tasks, 10 minutes for sync operations).

### IV. Versioned REST API

All HTTP endpoints MUST be versioned (`/v1/`, `/v2/`, `/v3/`) and
follow established conventions.

- New features SHOULD target the latest API version.
- Existing endpoints MUST NOT introduce breaking changes; create
  a new version instead.
- All responses MUST use the standardized `APIResponse` wrapper
  (`{status, message, data, error}`).
- Request validation MUST use `go-playground/validator` struct tags.
- API schemas MUST be defined in `internal/schema/v{N}/`.
- Swagger annotations MUST be maintained for all public endpoints.

### V. Observability

All services and workers MUST be observable through structured
logging, metrics, and distributed tracing.

- **Logging**: Use `zerolog` via `pkg/logger/`. Include request IDs
  and job IDs in all log entries. Use appropriate log levels
  (Info for operations, Error for failures, Debug for diagnostics).
- **Metrics**: Expose Prometheus metrics via `pkg/metrics/` for
  HTTP, database, cache, worker, business, and AI operations.
- **Tracing**: Use OpenTelemetry via `pkg/telemetry/` for
  distributed trace propagation across service boundaries.
- Workers MUST log task type, payload summary, and outcome
  (success/failure with error details).

### VI. Soft Delete & Data Safety

Data deletion MUST use soft delete (`is_deleted = true`) unless
there is an explicit, documented reason for hard deletion.

- All repository `Delete()` calls perform soft delete by default.
- `HardDelete()` MUST only be used for data cleanup operations
  (e.g., GDPR compliance, expired embeddings) and MUST be
  explicitly justified in code comments.
- The `SoftDeleteMixin` on domain models ensures all queries
  automatically exclude soft-deleted records.

## Technology Stack & Constraints

- **Language**: Go 1.25+ (module: `github.com/rockship/cosmo-agents-go`)
- **HTTP Framework**: Fiber v3 with middleware pipeline
- **ORM**: GORM with PostgreSQL driver (prepared statements enabled)
- **Task Queue**: Asynq with Redis backend
- **Workflow Engine**: Temporal SDK for long-running processes
- **AI/LLM**: OpenAI API (GPT-4o-mini) via `sashabaranov/go-openai`
- **Vector Store**: Redis Stack (RediSearch, HNSW, cosine similarity)
- **Cache**: Redis with in-memory fallback (`pkg/cache/`)
- **Object Storage**: AWS S3 for knowledge base files
- **Auth**: JWT (HS256/RS256) with personal API key support
- **Integrations**: Gmail API, Microsoft Graph, HubSpot CRM, Facebook Lead Ads
- **Migrations**: golang-migrate with sequential numbered SQL files
- **Observability**: zerolog + Prometheus + OpenTelemetry (Jaeger/Tempo)
- **Build**: `make build` produces `bin/server` and `bin/worker`

## Development Workflow

- Feature branches MUST target `dev` for integration, then merge
  to `main` for production releases.
- Database changes MUST use numbered migration files in `migrations/`
  created via `make migrate-create name=<description>`.
- Code MUST pass `make fmt` (gofmt + goimports) and `make lint`
  (golangci-lint) before committing.
- Tests MUST pass via `make test` before merging.
- Swagger docs MUST be regenerated (`make swagger`) when API
  endpoints are added or modified.
- Environment configuration uses `.env` files loaded via
  `godotenv` + `viper`. Secrets MUST NOT be committed.

## Agent Preferences

- **Preferred Agent**: Claude Code

## Governance

This constitution defines the authoritative development standards
for the Cosmo Agents Backend project. All contributions MUST comply
with these principles.

- Constitution supersedes ad-hoc practices. When a principle
  conflicts with a proposed change, the principle wins unless
  the constitution is formally amended.
- Amendments require: (1) documented rationale, (2) team review,
  (3) version bump following semver (MAJOR for principle
  removals/redefinitions, MINOR for additions, PATCH for
  clarifications).
- Complexity beyond what principles prescribe MUST be justified
  in the Complexity Tracking table of the relevant plan.md.

**Version**: 1.0.0 | **Ratified**: 2026-02-27 | **Last Amended**: 2026-02-27
