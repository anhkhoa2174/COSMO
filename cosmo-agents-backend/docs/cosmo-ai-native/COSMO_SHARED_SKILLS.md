# COSMO — SHARED SKILLS LIBRARY (MVP)

Goal: define reusable skills for agents that plug into the existing Go backend + new segmentation schema (migration 000033). Each skill is thin: call API/DB, return structured JSON, no business logic hidden.

## Data/Endpoints the skills use
- Contacts: `confirmed_facts`, `ai_insights`, `insight_validation`, `scores` (JSONB). API: `/v1/contacts` (POST/PATCH/GET).
- Segments: table `segmentations`, scores table `contact_segment_scores`. API: `/v1/segmentations` (POST/GET), `/v1/segmentations/:segmentation_id/contacts/:contact_id/score` (PUT), `/v1/segmentations/contacts/:contact_id/scores` (GET).
- Interactions: table `interactions` (no handler yet; add later or write via SQL/worker).

## Skill 1: ContactQuerySkill
**Responsibility:** fetch/filter contacts with facts/insights/scores.

**Tools:**
- `get_contact(contact_id: str) -> Contact`: call `GET /v1/contact/{id}`; return contact JSON including `confirmed_facts`, `ai_insights`, `insight_validation`, `scores`.
- `filter_contacts(filters: dict, limit: int = 100) -> List[Contact]`: call `POST /v1/contacts/search` with filters; surface key filters for MVP:
  - `"email"`, `"company"`, `"job_title"` (exact/ilike depending on handler).
  - `"confirmed_facts.<path>"` and `"ai_insights.<path>"` (interpreted server-side as JSONB path—backend filter can be extended to support these keys).
  - `"scores.fit"`, `"scores.priority"`, `"scores.engagement"` (numeric compare).
- `search_contacts_semantic(...)` (optional, needs Redis vectors) — TODO when vector search is enabled.

**Backend glue:** current search handler doesn’t parse JSONB paths. Minimal change: in `normalizeContactFilter` add cases to map `scores.fit` → `scores ->> 'fit'` etc., or implement dedicated search endpoint for JSONB fields.

## Skill 2: SegmentationLogicSkill
**Responsibility:** evaluate contact vs segment criteria, write fit score.

**Tools:**
- `evaluate_contact_fit(contact: dict, segmentation_id: str) -> dict`: fetch segment (`GET /v1/segmentations` or DB), read `criteria` + `icp_definition`, apply rules (filters, scoring weights), return `fit_score`, `score_breakdown`, `passes_filters`.
- `update_segmentation_criteria(segmentation_id: str, criteria_updates: dict) -> Segmentation`: merge criteria and `PUT` back (not yet exposed; can call DB or add PATCH route).
- `save_score(contact_id: str, segmentation_id: str, score_payload: dict)`: call `PUT /v1/segmentations/{seg_id}/contacts/{contact_id}/score` with `fit_score`, `score_breakdown`, `status`, flags. Uses `contact_segment_scores` table.

**Scoring heuristic (MVP):** weights from COSMO_PLAN (title 0.25, company size 0.15, industry 0.15, engagement 0.30, user_defined 0.15). Can run inside skill; backend only stores result.

## Skill 3: FeedbackCaptureSkill
**Responsibility:** capture human feedback, move insights between `ai_insights` ↔ `confirmed_facts`, adjust scores.

**Tools:**
- `capture_score_adjustment(contact_id, field_path, adjusted_score, reason)`: patch contact `scores` (or profile path) via `PATCH /v1/contacts/{id}`; optionally log feedback in a `user_feedback` table (future).
- `capture_insight_validation(contact_id, insight_type, insight_text, validation, confirmed_data=None)`: update `confirmed_facts` or `insight_validation` via `PATCH /v1/contacts/{id}`; remove from `ai_insights` if rejected.
- `capture_custom_fact(contact_id, fact_type, fact_data)`: append to `confirmed_facts` via `PATCH /v1/contacts/{id}`.
- `get_feedback_summary(user_id, days=30)`: optional, aggregates from `insight_validation` and `scores` edits (needs feedback table later).

**Backend glue:** no dedicated feedback endpoints; reuse contact PATCH to mutate JSONB fields. If feedback table needed, add migrations + endpoints later.

## Skill 4: InteractionLogSkill (requires small API addition)
**Responsibility:** log and fetch interactions for engagement scoring.

**Tools (to add):**
- `log_interaction(contact_id, interaction_type, channel, direction, content, ai_analysis)` → insert into `interactions` table. Add route `POST /v1/interactions` (or worker job) to write.
- `get_interaction_history(contact_id, days=90)` → query `interactions`, aggregate basic metrics (counts per type, last occurred_at).

## Skill 5: ScoringSkill (thin wrapper)
**Responsibility:** persist segment scores using existing API.

**Tools:**
- `calculate_and_save_score(contact_id, segmentation_id, score_data)` -> reuse `SegmentationLogicSkill` for calculation, then call `PUT /v1/segmentations/{seg_id}/contacts/{contact_id}/score` to store.

## How to map skills to current Go backend
- Use HTTP adapter for skills (Python agent SDK calls REST). Endpoints already available for contacts/segmentations/scores; interaction endpoints need to be added if required.
- JSONB fields are already exposed in contact handler and returned in responses.
- Segmentations/score endpoints are wired under `/v1/segmentations`.

## Minimal implementation checklist (backend)
- [x] Migration for facts/insights/scores + segmentations/contact_segment_scores + interactions.
- [x] Contact handler DTOs include new JSONB fields.
- [x] Segmentation handler + routes.
- [ ] Extend contact search to filter on `scores.*` and JSONB paths if needed by ContactQuerySkill.
- [ ] Add interaction handler (POST log, GET by contact) to back InteractionLogSkill.
- [ ] Optional: add feedback table + endpoints if FeedbackCaptureSkill needs persistence beyond contact JSONB.

## Example Python skill adapter (HTTP-based, pseudo)
```python
class ContactQuerySkill(Skill):
    name = "contact_query"
    async def get_contact(self, contact_id: str):
        resp = await http.get(f"{API}/v1/contact/{contact_id}", headers=auth())
        return resp.json()["data"]

    async def filter_contacts(self, filters: dict, limit=50):
        payload = {"filter": filters, "limit": limit}
        resp = await http.post(f"{API}/v1/contacts/search", json=payload, headers=auth())
        return resp.json()["data"].get("items", [])
```
Similar thin wrappers can be made for segmentation and feedback skills calling the routes above.
