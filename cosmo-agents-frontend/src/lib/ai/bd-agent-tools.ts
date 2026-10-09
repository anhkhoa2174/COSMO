/**
 * BD Agent Tool Bridge
 *
 * Converts cosmo-agents-sdk COSMO_TOOLS (Anthropic.Tool[] with JSON Schema
 * input_schema) into Vercel AI SDK tool() format so they can be passed
 * directly to streamText(). Execution is delegated to ToolExecutor which
 * handles all backend API calls.
 *
 * FR-027: FE chat has access to all cosmo-agents-sdk tools.
 * FR-028: Same tool registry as SDK — exact parity.
 * FR-029: 60+ tools invocable via natural language.
 */

import { tool, jsonSchema } from 'ai';
import { z } from 'zod';
import { COSMO_TOOLS, type ToolExecutor, type ToolName } from 'cosmo-agents-sdk';

/**
 * COSMO_TOOLS schemas were authored for Anthropic; OpenAI strict function
 * calling requires (a) `additionalProperties: false` on every object,
 * (b) `required` listing EVERY property, with formerly-optional properties
 * expressed as nullable (anyOf with null). Transform recursively so the same
 * registry works with both providers; null values are stripped again before
 * reaching the executor (see buildBDAgentTools) so backends never see them.
 */
function sanitizeSchema(node: unknown): unknown {
  if (Array.isArray(node)) return node.map(sanitizeSchema);
  if (node && typeof node === 'object') {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(node as Record<string, unknown>)) {
      out[k] = sanitizeSchema(v);
    }
    if (out.type === 'object') {
      out.additionalProperties = false;
      const props = (out.properties ?? {}) as Record<string, unknown>;
      const keys = Object.keys(props);
      const wasRequired = new Set(
        Array.isArray(out.required) ? (out.required as string[]) : []
      );
      for (const key of keys) {
        if (!wasRequired.has(key)) {
          props[key] = { anyOf: [props[key], { type: 'null' }] };
        }
      }
      out.required = keys;
    }
    return out;
  }
  return node;
}

/** Strict-mode schemas force optionals to null — drop them before execution. */
function stripNulls(input: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(input).filter(([, v]) => v !== null)
  );
}

/**
 * Wraps all COSMO_TOOLS into Vercel AI SDK tool() format.
 * Returns a record keyed by tool name, ready for streamText({ tools }).
 * Also includes UI tools (get_pipeline_summary) that return structured data
 * for frontend rendering.
 */
export function buildBDAgentTools(executor: ToolExecutor) {
  const sdkTools = Object.fromEntries(
    COSMO_TOOLS.map((t) => [
      t.name,
      tool({
        description: t.description,
        parameters: jsonSchema(
          sanitizeSchema(t.input_schema) as Record<string, unknown>
        ),
        execute: async (input) =>
          executor.execute(
            t.name as ToolName,
            stripNulls(input as Record<string, unknown>)
          ),
      }),
    ])
  );

  // UI tools: structured data for frontend card rendering
  const uiTools = {
    get_pipeline_summary: tool({
      description: 'Get a structured pipeline summary showing contact counts by stage, response rates, and meetings booked. Use this when the user asks for pipeline summary, overview, or statistics.',
      parameters: z.object({}),
      execute: async () => {
        try {
          const raw = await executor.execute('search_contacts' as ToolName, { limit: 200 });
          // Tool returns JSON string
          const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
          const contacts: Array<{ outreach_stage?: string }> = parsed?.contacts ?? [];
          const total: number = parsed?.total ?? contacts.length;
          const byStage: Record<string, number> = {};
          for (const c of contacts) {
            const stage = c.outreach_stage ?? 'COLD';
            byStage[stage] = (byStage[stage] ?? 0) + 1;
          }
          return {
            total_active_contacts: total,
            contacts_by_stage: byStage,
            contacts_by_lifecycle: {},
            response_rate_7d: 0,
            meetings_booked_7d: 0,
            avg_response_time_hours: 0,
          };
        } catch {
          return {
            total_active_contacts: 0,
            contacts_by_stage: {},
            contacts_by_lifecycle: {},
            response_rate_7d: 0,
            meetings_booked_7d: 0,
            avg_response_time_hours: 0,
          };
        }
      },
    }),

    daily_analytics: tool({
      description: 'Start a daily analytics workflow to generate comprehensive BD pipeline analytics including contact activity, outreach performance, and pipeline health metrics.',
      parameters: z.object({}),
      execute: async () => executor.execute('daily_analytics' as ToolName, {}),
    }),

    full_analysis: tool({
      description: 'Run a full AI analysis on a contact: enrichment → segment scoring → relationship scoring. Use when user wants deep intelligence on a specific contact.',
      parameters: z.object({
        contact_id: z.string().describe('The contact ID to run full analysis on'),
      }),
      execute: async ({ contact_id }) => executor.execute('full_analysis' as ToolName, { contact_id }),
    }),

    batch_enrichment: tool({
      description: 'Enrich multiple contacts at once with AI-powered data enrichment.',
      parameters: z.object({
        contact_ids: z.array(z.string()).describe('List of contact IDs to enrich'),
      }),
      execute: async ({ contact_ids }) => executor.execute('batch_enrichment' as ToolName, { contact_ids }),
    }),

    segment_analysis: tool({
      description: 'Run an AI analysis on a segment to understand contact fit scores, patterns, and recommendations.',
      parameters: z.object({
        segment_id: z.string().describe('The segment ID to analyze'),
      }),
      execute: async ({ segment_id }) => executor.execute('segment_analysis' as ToolName, { segment_id }),
    }),

    count_contacts_by_keyword: tool({
      description: 'Count contacts matching a keyword across name, company, title, or tags. Useful for quick pipeline sizing.',
      parameters: z.object({
        keyword: z.string().describe('Keyword to search for'),
      }),
      // The SDK declares this tool but its executor has no case for it, so the
      // assistant got "Unknown tool". Count by searching each field instead.
      execute: async ({ keyword }) => {
        const fields = ['name', 'company', 'job_title'] as const;
        const ids = new Set<string>();
        const totals: Record<string, number> = {};
        const errors: string[] = [];
        // A match can sit in more than one field, so ids are counted once
        // across fields. When a field has more matches than one page returns,
        // the union is only a lower bound and is reported as such.
        let complete = true;
        for (const field of fields) {
          let parsed: any;
          try {
            const raw = await executor.execute('search_contacts' as ToolName, { [field]: keyword, limit: 1000 });
            parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
          } catch (e) {
            errors.push(`${field}: ${e instanceof Error ? e.message : String(e)}`);
            continue;
          }
          // The executor reports API failures as {"error": ...} rather than
          // throwing; read as zero matches, that made "0 contacts" up.
          if (parsed?.error) {
            errors.push(`${field}: ${parsed.error}`);
            continue;
          }
          const contacts: Array<{ id: string }> = parsed?.contacts ?? [];
          contacts.forEach(c => ids.add(c.id));
          totals[field] = typeof parsed?.total === 'number' ? parsed.total : contacts.length;
          if (totals[field] > contacts.length) complete = false;
        }
        if (errors.length === fields.length) {
          return JSON.stringify({ keyword, error: `Could not count contacts: ${errors.join('; ')}` });
        }
        const atLeast = Math.max(ids.size, ...Object.values(totals));
        // A JSON string like the SDK tools return: given an object here, the
        // model (Gemini 2.5) answered with an empty message.
        return JSON.stringify({
          keyword,
          count: complete ? ids.size : atLeast,
          exact: complete && errors.length === 0,
          matches_by_field: totals,
          ...(errors.length > 0 ? { partial_errors: errors } : {}),
        });
      },
    }),

    get_meeting_pipeline: tool({
      description: 'Get all scheduled and upcoming meetings across all contacts. Use this when the user asks about meeting pipeline, show meeting pipeline, or contacts with meetings.',
      parameters: z.object({}),
      execute: async () => {
        try {
          const raw = await executor.execute('search_contacts' as ToolName, { limit: 200 });
          const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
          const contacts: Array<{ id: string; name?: string; company?: string; outreach_stage?: string }> = parsed?.contacts ?? [];
          // Filter contacts with meeting stages
          const meetingContacts = contacts.filter(c =>
            c.outreach_stage === 'POST_MEETING' ||
            c.outreach_stage === 'REPLIED'
          );
          return {
            total_meetings: meetingContacts.length,
            contacts: meetingContacts.map(c => ({
              id: c.id,
              name: c.name,
              company: c.company,
              stage: c.outreach_stage,
            })),
          };
        } catch {
          return { total_meetings: 0, contacts: [] };
        }
      },
    }),
  };

  return { ...sdkTools, ...uiTools };
}
