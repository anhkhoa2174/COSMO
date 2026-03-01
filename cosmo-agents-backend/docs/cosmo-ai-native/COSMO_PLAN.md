# COSMO – AI-ASSISTED CRM FOR HIGH-QUALITY CUSTOMER DISCOVERY

Specification distilled from the PDF pack (01–62). Focus: build a CRM that answers 3 core questions for every contact — Identity & Context, Customer Fit, Engagement & Priority — with clear separation of facts vs. AI insights, human-in-the-loop, and dynamic segmentation/playbooks.

---

## 1. Business Flow & Architecture
- **System objective:** CRM for decision support, not just storage. Always answer:
  1) Identity & Context (who is this contact?),
  2) Customer Fit (do they match ICP?),
  3) Engagement & Priority (should we act now?).
- **Phases:**
  - **Phase 1 – Data Foundation & Signal Quality (must-have):** make data correct/clean/meaningful; separate fact vs insight; build base for accurate segmentation/campaign.
  - **Phase 2 – AI-Assisted Intelligence:** automate segmentation, prioritization, and recommended actions once signals are good.
- **Components:**
  - **Component A – Contact Intelligence Core (Phase 1)**
    - **Smart ingestion & cleaning:** sources LinkedIn, Apollo, email lists, socials, CSV/manual; dedup priority linkedin_url → email → full_name+company_domain; merge into canonical contact; normalization of job title/industry/seniority; intake classification READY (complete), PENDING (missing data), ARCHIVED (out of ICP).
    - **Confirmed Facts vs AI Insights:** facts are user-verified (pain, goals, requirements, constraints) and always prioritized; insights are AI hypotheses with confidence/evidence.
    - **Interaction log & relationship signals:** quick log type/time/quality/sentiment/follow-up; system derives days since last, frequency, relationship strength (0–100).
    - **Status & priority signals:** lifecycle stage (new → contacted → engaged → meeting → customer → closed_lost), fit score, engagement score, priority score; V1 uses rules/heuristics.
  - **Component B – AI Reasoning & Assist Layer (Phase 2)**
    - **AI-assisted enrichment:** gap detector compares profile vs ICP; triggers suggestion cards for missing decision_power/use_case/budget_owner; human-in-loop to confirm.
    - **Automatic/dynamic segmentation:** combine fit + engagement + confirmed facts + business context; multiple segments per contact; virtual segments when thresholds met.
    - **Meeting intelligence:** pre-meeting brief from last 3 interactions + confirmed facts + LinkedIn + current segments; generates talking points, discovery questions, risk flags; post-meeting updates engagement/segments/next actions.
    - **Continuous learning loop:** log decisions/outcomes, compare AI suggestions vs reality, adjust scoring/prompt; no RL in V1.

---

## 2. Tech Solution (V1/V2)
- **Stack guidance:** Postgres (JSONB for new contact fields), Redis vector store (RediSearch on Supabase), extend sales_rep with JSON. Keep flexibility; avoid heavy migrations by leaning on JSONB.
- **Component A – Data foundation:**
  - **Smart Ingestion Engine:** NLP normalize titles (Spacy/GPT-4o-mini); dedup multi-step (linkedin_url exact → email → fuzzy full_name+domain); merge to canonical record to avoid fragmentation.
  - **Schema split (facts vs insights):**
    - `confirmed_facts`: only overwritten by user action; includes fact_type, fact_value, source_interaction_id, verified_at.
    - `ai_insights`: overwritten by AI agents; hypothesis, confidence_score (0–1), evidence_list.
    - **Priority logic:** queries favor confirmed_facts; fallback to ai_insights when empty.
- **Component B – Reasoning & assist:**
  - **Gap detector:** worker compares contact profile vs ICP definition; if missing decision_power/use_case/budget, emit event to AI task; surface suggestion card in UI.
  - **Dynamic segmentation:** Fit_Score = f(title, industry, company_size); Engagement_Score = f(email_open, linkedin_accept, response_time). Auto-tag into virtual segment when total score passes threshold; contact can belong to many segments.
  - **Meeting intelligence:** Pre-meeting RAG fetch last 3 interactions + confirmed_facts + LinkedIn; generate talking points, discovery questions on gaps, risk flags; post-meeting update engagement/segments/next actions.

---

## 3. Data Models (aligned with PDF)
- **contacts** (extended JSONB):
  - Base: email/first/last/title/company, lifecycle_stage, relationship_strength (0–100), responsiveness (0–100), last_interaction, interaction_count, meetings_had, social links.
  - JSONB: `ai_insights` (suspected_pain_points/goals/buying_signals/decision_style/communication_style/anticipated_objections/etc.), `confirmed_facts` (pain_points/goals/requirements/budget_timeline/decision_process/objections_raised/preferences/personal_context/competitors/success_metrics), `insight_validation` (confirmed/rejected/pending), `custom_fields`.
  - Indexes: email, company, lifecycle_stage, last_interaction, GIN(ai_insights), GIN(confirmed_facts).
- **sales_reps**: social presence, personal brand metrics, network stats, preferences JSONB (channels/hours/templates), performance JSONB (win_rate, avg_deal_size, etc.), scores with constraints.
- **segmentations**: name, priority (1–10), `criteria` JSONB (filters, scoring_rules, exclusions), `icp_definition` JSONB (ideal_title_patterns/size/industries/tech_stack/key_attributes), is_active, optional playbook_id/created_by/team_id; GIN(criteria).
- **contact_segment_scores**: contact_id, segmentation_id, fit_score (0–10 or 0–100 depending on impl), score_breakdown JSONB (title/company_size/industry/tech_stack/engagement/buying_signals), score_type (auto/manual), status (qualified/active/paused/excluded), enrollment flags; unique(contact, segmentation); indexes on fit_score/status.
- **playbooks / stages / actions**: playbook_type (nurture/outreach/re-engagement/upsell), config JSONB (stages, messaging_strategy, timing_rules, channel_sequence), stage/action types (send_email, linkedin_touch, wait_for_reply, evaluate_signal), stage templates, metrics/performance JSONB, ownership, active flag.
- **interactions**: user_id, contact_id, campaign_id?, interaction_type, details JSONB, ai_analysis JSONB, occurred_at; indexes contact_id, occurred_at DESC.
- **user_feedback**: feedback_type (score_adjustment/insight_validation/custom_fact/segmentation_criteria), entity_type/id, feedback_data JSONB, applied/applied_at; indexes user_id, type, entity.
- **agent_activity_log**: agent_type/action/entity, input/output JSONB, model, confidence, processing_time_ms, status/error; indexes agent_type, created_at DESC.

---

## 4. Shared Skills & Agents (per PDF spec)
- **Skills (shared library):**
  - ContactQuery (get/filter/semantic with Redis vectors).
  - SegmentationLogic (evaluate fit, user filters/scoring rules, update criteria).
  - Scoring (save contact_segment_scores).
  - InteractionLog (log interaction, list history).
  - FeedbackCapture (score adjust, insight validation, custom fact, feedback summary).
  - Embedding (generate/store vectors in Redis).
- **Agents:**
  - **ContactEnrichmentAgent:** contact + interactions + similar contacts → AI insights (pain/goals/buying signals/decision_style/communication_prefs/objections), store ai_insights, generate embedding, log activity.
  - **SegmentCalculatorAgent:** apply SegmentationLogic over segments, persist scores, update contact segmentation/priority.
  - **InsightValidatorAgent:** move ai_insights ↔ confirmed_facts, track validation accuracy, log feedback.
  - **RelationshipScorerAgent:** compute relationship strength/health from interactions; scheduled (nightly).
  - **NetworkAnalyzerAgent:** warm-path discovery (LinkedIn API if available), intro probability, generate intro templates.
  - **CampaignIntelligenceAgent:** campaign metrics (open/reply/meeting), segment breakdown, recommendations.

---

## 5. Human-in-the-Loop Workflows
- **Update segmentation criteria:** UI edits (add_filters/add_scoring_rules/remove_filters) → API `/segmentations/{id}/update-criteria` → SegmentationLogicSkill merges criteria, logs user_feedback; background job recalculates scores for all contacts in segment; UI shows delta (e.g., 98 → 127 contacts matched).
- **Validate AI insight:** UI confirm/reject insight with details → API `/contacts/{id}/insights/validate` → moves ai_insights → confirmed_facts or rejects; updates insight_validation; logs feedback; learning loop tracks AI confidence vs user validation.
- **Adjust relationship/other score:** UI slider sets adjusted_score with reason → FeedbackCaptureSkill logs adjustment, updates contact.profile path, marks pattern for learning; UI shows user-adjusted score.
- **Add custom fact:** UI modal → API `/contacts/{id}/add-fact` → add to confirmed_facts.<type>, user_added metadata, feedback logged; UI shows confirmed fact with timestamp.

---

## 6. Implementation Tasks (EPICs from PDF)
- **EPIC 1 – Database & Infra**
  - Create tables: interactions, segmentations, contact_segment_scores, user_feedback, agent_activity_log; add JSONB indexes to contacts (profile/ai_insights/confirmed_facts), full-text/search indexes; Redis setup (RediSearch, vector similarity, pooling); perf test 10k contacts.
- **EPIC 2 – Shared Skills Library**
  - ContactQuery (get/filter/semantic), SegmentationLogic (fit eval + criteria updates), FeedbackCapture (score adjust/insight validation/custom fact/summary), Scoring (save scores), InteractionLog (log/history), Embedding (generate+store in Redis); unit tests and perf targets (<50ms filters).
- **EPIC 3 – Agents**
  - ContactEnrichmentAgent, SegmentCalculatorAgent, InsightValidatorAgent, RelationshipScorerAgent (nightly), NetworkAnalyzerAgent (warm paths), CampaignIntelligenceAgent; integration tests; API endpoints: `/contacts/{id}/enrich`, `/contacts/{id}/calculate-scores`, `/contacts/{id}/insights/validate`, `/contacts/{id}/adjust-score`, `/contacts/{id}/add-fact`, `/contacts/{id}/warm-paths`, `/campaigns/{id}/analyze`.
- **EPIC 4 – APIs**
  - Segmentation CRUD + update-criteria; feedback APIs; search APIs (semantic/filter); enrichment/scoring/validation endpoints; background triggers for recalculation.
- **EPIC 5 – Frontend**
  - Segment builder (filters/scoring editor, preview matches), contact profile (AI insights vs confirmed facts, confirm/reject, add fact, adjust score), feedback dashboard (validation accuracy, most-adjusted scores, common patterns), suggestion cards for gap filling.
- **EPIC 6 – Testing & Docs**
  - Unit tests (>80% skills/agents), integration (end-to-end workflows, DB transactions, APIs), OpenAPI, skill usage examples, agent architecture diagrams, user guides for segmentation.

---

## 7. Current Coverage vs Plan (status snapshot)
- **Done/aligned:** JSONB fields on contacts; tables segmentations/contact_segment_scores/interactions/user_feedback; handlers for segmentation/interaction/feedback; shared skills (contact_query, segmentation_logic, scoring, interaction_log, feedback_capture, embedding); endpoints enrich/calculate-scores; tests passing.
- **Partial/missing vs spec:** smart ingestion & dedup pipeline (LinkedIn/Apollo/CSV canonical merge), gap-detector worker & suggestion cards, Redis vector search wiring, meeting intelligence, network analyzer, campaign intelligence, playbooks/stages/actions, full frontend flows (segment builder, insights HIL UI, feedback dashboard), background recalculation triggers.

This plan replaces the previous COSMO_PLAN.md to mirror the PDF spec. Keep evolving sections 6–7 as implementation progresses. 
