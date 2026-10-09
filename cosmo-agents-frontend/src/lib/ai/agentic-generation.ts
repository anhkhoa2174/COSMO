/**
 * Agentic Daily Action Generation
 *
 * Assembles full contact context via SDK tools, builds a structured prompt,
 * and calls Claude to produce strategically reasoned action recommendations.
 *
 * 006-agentic-daily-actions: Single-prompt pattern with pre-assembled context.
 */

import type { ToolExecutor, ToolName } from 'cosmo-agents-sdk';

// ---- Types ----

export interface AgentContext {
  contacts: AgentContact[];
  outcomeMetrics: OutcomeMetricsData | null;
  knowledgeResults: Record<string, string[]>;
}

interface AgentContact {
  id: string;
  name: string;
  company: string;
  job_title: string;
  industry: string;
  email?: string;
  linkedin_url?: string;
  outreach_stage: string;
  next_step: string;
  days_since_last_interaction: number;
  followup_count: number;
  context_level: string;
  scores?: Record<string, unknown>;
  interactions: InteractionSummary[];
  notes: string[];
}

interface InteractionSummary {
  direction: string;
  channel: string;
  content: string;
  timestamp: string;
}

interface OutcomeMetricsData {
  total_sent: number;
  total_replied: number;
  reply_rate_overall: number;
  reply_rate_by_channel: Record<string, number>;
  reply_rate_by_industry: Record<string, number>;
  top_performing_strategies: Array<{
    strategy: string;
    reply_rate: number;
  }>;
}

export interface AgentRecommendation {
  contact_id: string;
  priority_rank: number;
  action_type: string;
  category_id: string;
  recommended_channel: string;
  messaging_strategy: string;
  strategic_reasoning: string;
  draft_message: string;
  referenced_knowledge: string[];
  confidence_level: 'high' | 'medium' | 'low';
  outcome_pattern_cited: string;
}

export interface AgentGenerationResult {
  recommendations: AgentRecommendation[];
  strategic_plan: string;
  focus_areas: string[];
  outcome_insights: string[];
}

// ---- Context Assembly ----

export async function assembleAgentContext(
  executor: ToolExecutor,
  outcomeMetricsBaseUrl?: string,
  authToken?: string
): Promise<AgentContext> {
  // 1. Get suggested contacts
  let contacts: AgentContact[] = [];
  try {
    const suggestResult = await executor.execute(
      'suggest_outreach' as ToolName,
      { type: 'mixed', limit: 15 }
    );
    const parsed =
      typeof suggestResult === 'string'
        ? JSON.parse(suggestResult)
        : suggestResult;
    const suggestions = parsed?.suggestions ?? parsed?.data ?? [];

    // 2. Get interaction histories (batch)
    contacts = await Promise.all(
      suggestions.map(async (s: Record<string, unknown>) => {
        const contact = (s.contact ?? s) as Record<string, unknown>;
        const state = (s.state ?? {}) as Record<string, unknown>;
        const contactId = String(
          contact.id ?? contact.contact_id ?? ''
        );

        let interactions: InteractionSummary[] = [];
        try {
          const historyResult = await executor.execute(
            'get_interaction_history' as ToolName,
            { contact_id: contactId, limit: 10 }
          );
          const histParsed =
            typeof historyResult === 'string'
              ? JSON.parse(historyResult)
              : historyResult;
          const logs = histParsed?.interactions ?? histParsed?.data ?? [];
          interactions = logs.slice(0, 10).map((l: Record<string, unknown>) => ({
            direction: String(l.direction ?? ''),
            channel: String(l.channel ?? ''),
            content: String(l.content ?? '').slice(0, 200),
            timestamp: String(l.timestamp ?? ''),
          }));
        } catch {
          // No history available
        }

        let notes: string[] = [];
        try {
          const notesResult = await executor.execute(
            'get_notes' as ToolName,
            { contact_id: contactId, limit: 3 }
          );
          const notesParsed =
            typeof notesResult === 'string'
              ? JSON.parse(notesResult)
              : notesResult;
          const notesList = notesParsed?.notes ?? notesParsed?.data ?? [];
          notes = notesList
            .slice(0, 3)
            .map((n: Record<string, unknown>) =>
              String(n.content ?? '').slice(0, 100)
            );
        } catch {
          // No notes
        }

        return {
          id: contactId,
          name: String(contact.name ?? ''),
          company: String(contact.company ?? ''),
          job_title: String(contact.job_title ?? ''),
          industry: String(contact.industry ?? ''),
          email: contact.email ? String(contact.email) : undefined,
          linkedin_url: contact.linkedin_url
            ? String(contact.linkedin_url)
            : undefined,
          outreach_stage: String(
            state.conversation_state ?? contact.outreach_stage ?? 'COLD'
          ),
          next_step: String(state.next_step ?? contact.next_step ?? 'SEND'),
          days_since_last_interaction: Number(
            state.days_since_last_interaction ?? s.days_since ?? 0
          ),
          followup_count: Number(state.followup_count ?? 0),
          context_level: String(state.context_level ?? 'LOW'),
          scores: contact.scores as Record<string, unknown> | undefined,
          interactions,
          notes,
        };
      })
    );
  } catch {
    // suggest_outreach failed — return empty context
  }

  // 3. Search knowledge base for unique industries
  const knowledgeResults: Record<string, string[]> = {};
  const industries = [...new Set(contacts.map((c) => c.industry).filter(Boolean))];
  for (const industry of industries.slice(0, 3)) {
    try {
      const kbResult = await executor.execute(
        'search_knowledge' as ToolName,
        { query: `${industry} case study product`, limit: 3 }
      );
      const kbParsed =
        typeof kbResult === 'string' ? JSON.parse(kbResult) : kbResult;
      const results = kbParsed?.results ?? kbParsed?.data ?? [];
      knowledgeResults[industry] = results
        .slice(0, 3)
        .map(
          (r: Record<string, unknown>) =>
            String(r.content ?? r.text ?? '').slice(0, 300)
        );
    } catch {
      // KB not available
    }
  }

  // 4. Get outcome metrics
  let outcomeMetrics: OutcomeMetricsData | null = null;
  if (outcomeMetricsBaseUrl && authToken) {
    try {
      const res = await fetch(
        `${outcomeMetricsBaseUrl}/v1/daily-actions/outcome-metrics?period=30d`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      );
      if (res.ok) {
        const json = await res.json();
        outcomeMetrics = json.data ?? null;
      }
    } catch {
      // Metrics not available
    }
  }

  return { contacts, outcomeMetrics, knowledgeResults };
}

// ---- Prompt Builder ----

const AGENT_SYSTEM_PROMPT = `You are COSMO BD Agent — an AI strategist that analyzes a BD pipeline and produces a prioritized daily action plan.

For each contact, decide:
1. PRIORITY: Why this contact matters today (strategic reasoning, not generic rules)
2. ACTION TYPE: outreach (cold), followup, respond, meeting_prep, or enrich
3. CATEGORY: new_outreach, followup, replied, meeting_prep, or enrichment
4. CHANNEL: Best channel (LinkedIn, Email, Call) with justification
5. STRATEGY: Messaging approach (value-add, reference post, share case study, ask question, etc.)
6. MESSAGE: Draft message aligned with the strategy
7. REASONING: Explain your decision with specific data points from the context

Rules:
- Maximum 15 contacts
- Reference outcome metrics when relevant (e.g., "LinkedIn follow-ups have 18% reply rate")
- Use knowledge base content when available for the contact's industry
- Differentiate contacts in the same outreach state — don't give the same advice to everyone
- If a contact was messaged 2+ times on one channel with no reply, suggest switching channels
- If a contact replied, prioritize them highly
- Contacts with upcoming meetings (<48h) should get meeting_prep actions

Return a JSON object with this exact schema:
{
  "recommendations": [
    {
      "contact_id": "uuid",
      "priority_rank": 1,
      "action_type": "outreach|followup|respond|meeting_prep|enrich",
      "category_id": "new_outreach|followup|replied|meeting_prep|enrichment",
      "recommended_channel": "LinkedIn|Email|Call",
      "messaging_strategy": "brief description of approach",
      "strategic_reasoning": "detailed reasoning with data points",
      "draft_message": "the actual message to send",
      "referenced_knowledge": ["item1.pdf"],
      "confidence_level": "high|medium|low",
      "outcome_pattern_cited": "specific metric if relevant, or empty string"
    }
  ],
  "strategic_plan": "1-2 sentence daily plan summary",
  "focus_areas": ["area1", "area2", "area3"],
  "outcome_insights": ["insight1", "insight2"]
}`;

export function buildAgentPrompt(context: AgentContext): {
  system: string;
  user: string;
} {
  let userPrompt = '';

  // Outcome metrics
  if (context.outcomeMetrics) {
    const m = context.outcomeMetrics;
    userPrompt += `## Your Performance Data (last 30 days)\n`;
    userPrompt += `- Total sent: ${m.total_sent}, Total replies: ${m.total_replied}, Overall reply rate: ${(m.reply_rate_overall * 100).toFixed(1)}%\n`;
    if (m.reply_rate_by_channel) {
      userPrompt += `- Reply rate by channel: ${Object.entries(m.reply_rate_by_channel).map(([k, v]) => `${k}: ${(v * 100).toFixed(1)}%`).join(', ')}\n`;
    }
    if (m.reply_rate_by_industry) {
      userPrompt += `- Reply rate by industry: ${Object.entries(m.reply_rate_by_industry).map(([k, v]) => `${k}: ${(v * 100).toFixed(1)}%`).join(', ')}\n`;
    }
    if (m.top_performing_strategies?.length) {
      userPrompt += `- Top strategies: ${m.top_performing_strategies.map((s) => `${s.strategy} (${(s.reply_rate * 100).toFixed(1)}%)`).join(', ')}\n`;
    }
    userPrompt += '\n';
  }

  // Knowledge base
  const kbEntries = Object.entries(context.knowledgeResults).filter(
    ([, v]) => v.length > 0
  );
  if (kbEntries.length > 0) {
    userPrompt += `## Knowledge Base (relevant excerpts)\n`;
    for (const [industry, excerpts] of kbEntries) {
      userPrompt += `### ${industry}\n`;
      for (const excerpt of excerpts) {
        userPrompt += `- ${excerpt}\n`;
      }
    }
    userPrompt += '\n';
  }

  // Contacts
  userPrompt += `## Contacts to Evaluate (${context.contacts.length})\n\n`;
  for (const c of context.contacts) {
    userPrompt += `### ${c.name} — ${c.job_title} at ${c.company}\n`;
    userPrompt += `- ID: ${c.id}\n`;
    userPrompt += `- Industry: ${c.industry || 'Unknown'}\n`;
    userPrompt += `- Stage: ${c.outreach_stage}, Next step: ${c.next_step}\n`;
    userPrompt += `- Days since last interaction: ${c.days_since_last_interaction}\n`;
    userPrompt += `- Follow-up count: ${c.followup_count}\n`;
    userPrompt += `- Channels available: ${[c.linkedin_url ? 'LinkedIn' : '', c.email ? 'Email' : ''].filter(Boolean).join(', ') || 'Unknown'}\n`;

    if (c.interactions.length > 0) {
      userPrompt += `- Recent interactions:\n`;
      for (const i of c.interactions.slice(0, 5)) {
        userPrompt += `  - [${i.timestamp}] ${i.direction} via ${i.channel}: ${i.content}\n`;
      }
    }

    if (c.notes.length > 0) {
      userPrompt += `- Team notes: ${c.notes.join(' | ')}\n`;
    }

    userPrompt += '\n';
  }

  userPrompt += `## Task\nAnalyze these contacts and produce a prioritized daily action plan. Return JSON matching the schema exactly.`;

  return { system: AGENT_SYSTEM_PROMPT, user: userPrompt };
}
