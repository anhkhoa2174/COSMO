import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';
import type { Language } from '@/network/client/outreach';

export interface PipelineSummary {
  total: number;
  /** COLD, NO_REPLY, REPLIED, POST_MEETING, DROPPED. */
  by_stage: Record<string, number>;
  /** Contacts with 0, 1, 2, and 3+ follow-ups. */
  followup_depth: [number, number, number, number];
  /** Still in play, no interaction for five days or more. */
  stalled: number;
}
import type {
  ContactInteractionsResponse,
  ContactsPipelineResponse,
  DailyAction,
  DailyActionsBriefing,
  DailyActionsStatusResponse,
  GenerateRequest,
  GenerateResponseData,
  OutcomeMetrics,
  UpdateActionRequest,
  UpdateActionResponseData,
  UpdateTagsContact,
} from '@/types/daily-actions';

const DailyActionsApi = {
  /**
   * Trigger daily actions generation.
   * Call on page open (morning briefing) or explicit refresh.
   * Returns immediately; listen on SSE or poll GET for completion.
   */
  generate: async (options?: GenerateRequest) => {
    const data = await kyClient.post('v1/daily-actions/generate', {
      json: options ?? {},
    });
    return data.json<ApiResponse<GenerateResponseData>>();
  },

  /**
   * Get today's computed daily actions briefing.
   * Cheap read from cache/DB — no AI computation.
   */
  getDailyActions: async (options?: {
    language?: Language;
    include_completed?: boolean;
  }) => {
    const data = await kyClient.get('v1/daily-actions', {
      searchParams: options
        ? Object.fromEntries(
            Object.entries(options)
              .filter(([, v]) => v !== undefined)
              .map(([k, v]) => [k, String(v)])
          )
        : {},
    });
    return data.json<ApiResponse<DailyActionsBriefing>>();
  },

  /**
   * Load more actions for a specific category (pagination).
   * FR-021: Categories with 5+ actions load in batches.
   */
  loadMoreActions: async (
    categoryId: string,
    offset: number,
    limit: number = 5
  ) => {
    const data = await kyClient.get(
      `v1/daily-actions/categories/${categoryId}/actions`,
      {
        searchParams: { offset: String(offset), limit: String(limit) },
      }
    );
    return data.json<
      ApiResponse<{
        actions: DailyAction[];
        has_more: boolean;
        next_offset?: number;
      }>
    >();
  },

  /**
   * Update a single action's state.
   * Used for: Mark Sent, Skip, Snooze, Complete, Reopen.
   */
  updateAction: async (actionId: string, request: UpdateActionRequest) => {
    const data = await kyClient.patch(`v1/daily-actions/${actionId}`, {
      json: request,
    });
    return data.json<ApiResponse<UpdateActionResponseData>>();
  },

  /**
   * Open SSE event stream for real-time updates.
   * Returns an EventSource instance the caller must manage.
   */
  openEventStream: (token?: string) => {
    const baseUrl = process.env.NEXT_PUBLIC_SERVER_BASE_URL;
    const url = token
      ? `${baseUrl}/v1/daily-actions/events?token=${encodeURIComponent(token)}`
      : `${baseUrl}/v1/daily-actions/events`;
    return new EventSource(url);
  },

  // ===========================================================================
  // MCP ENDPOINTS (from cosmo-agents-mcp)
  // ===========================================================================

  /**
   * Get performance metrics: reply rates by channel, strategy, industry, time of day.
   * Call before creating daily actions for data-driven recommendations.
   */
  getOutcomeMetrics: async (period: '7d' | '30d' = '30d') => {
    const data = await kyClient.get('v1/mcp/outcome-metrics', {
      searchParams: { period },
    });
    return data.json<ApiResponse<OutcomeMetrics>>();
  },

  /**
   * Get contacts with outreach pipeline states, sorted by priority.
   * Shows conversation state, next steps, days since last contact.
   */
  getContactsPipeline: async (options?: {
    limit?: number;
    type?: 'mixed' | 'cold' | 'followup';
  }) => {
    const searchParams: Record<string, string> = {};
    if (options?.limit) searchParams.limit = String(options.limit);
    if (options?.type) searchParams.type = options.type;
    const data = await kyClient.get('v1/mcp/contacts/pipeline', {
      searchParams,
    });
    return data.json<ApiResponse<ContactsPipelineResponse>>();
  },

  /**
   * Counts of the caller's contacts by outreach stage, follow-up depth and
   * staleness, aggregated over every contact rather than the suggestion list.
   */
  getPipelineSummary: async () => {
    const data = await kyClient.get('v1/mcp/contacts/pipeline-summary');
    return data.json<ApiResponse<PipelineSummary>>();
  },

  /**
   * Get full interaction history for a specific contact.
   * Includes messages, channels, direction, and sentiment.
   */
  getContactInteractions: async (contactId: string, limit: number = 20) => {
    const data = await kyClient.get(
      `v1/mcp/contacts/${contactId}/interactions`,
      { searchParams: { limit: String(limit) } }
    );
    return data.json<ApiResponse<ContactInteractionsResponse>>();
  },

  /**
   * Get current daily actions status with progress tracking.
   * Shows pending, completed, snoozed, and skipped actions.
   */
  getDailyActionsStatus: async (language?: Language) => {
    const searchParams: Record<string, string> = {};
    if (language) searchParams.language = language;
    const data = await kyClient.get('v1/mcp/daily-actions/status', {
      searchParams,
    });
    return data.json<ApiResponse<DailyActionsStatusResponse>>();
  },

  /**
   * Update tags for contacts in batch.
   */
  updateContactTags: async (contacts: UpdateTagsContact[]) => {
    const data = await kyClient.post('v2/contacts/batch', {
      json: { contacts },
    });
    return data.json<ApiResponse<unknown>>();
  },
};

export default DailyActionsApi;
