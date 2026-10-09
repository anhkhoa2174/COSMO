# cosmo-agents-frontend Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-11

## Active Technologies
- Backend API (REST) — frontend is stateless; server state via TanStack Query cache (001-daily-bd-actions)
- TypeScript 5.9.3, Next.js 14.2.25, React 18.3.1, Vercel AI SDK (`ai` 4.3.19, `@ai-sdk/react`), `@ai-sdk/anthropic`, `cosmo-agents-sdk` (local path dep), TanStack React Query 5.90, Jotai 2.15, Ky 1.14, shadcn/ui (Radix UI), Lucide React, react-markdown 9.1, date-fns 4.1, sonner 2.0, Zod 3.25 (001-daily-bd-actions)
- No new storage — reads from existing COSMO backend API via CosmoApiClient (001-daily-bd-actions)
- TypeScript 5.9.3 (frontend), Go 1.21+ (backend) + Vercel AI SDK (`ai` 4.3.19, `@ai-sdk/anthropic`), cosmo-agents-sdk, TanStack Query — frontend. Fiber, GORM, Asynq — backend. (006-agentic-daily-actions)
- PostgreSQL (1 new table: `outcome_metrics`) (006-agentic-daily-actions)

## Project Structure

```text
src/
tests/
```

## Commands

npm test && npm run lint

## Code Style

TypeScript 5.9.3, Next.js 14.2.25, React 18.3.1: Follow standard conventions

## Recent Changes
- 006-agentic-daily-actions: Added TypeScript 5.9.3 (frontend), Go 1.21+ (backend) + Vercel AI SDK (`ai` 4.3.19, `@ai-sdk/anthropic`), cosmo-agents-sdk, TanStack Query — frontend. Fiber, GORM, Asynq — backend.
- 001-daily-bd-actions: Unified spec — single vertical chat flow with daily briefing + conversational queries. Desktop-first responsive, skeleton loading, toast errors, 60s polling, data isolation, keyboard shortcuts (Enter/Escape/Ctrl+K). FR-001–FR-049.

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
