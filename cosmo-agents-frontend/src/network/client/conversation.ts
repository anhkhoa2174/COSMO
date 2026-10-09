import { kyClient } from '@/lib/ky';
import type {
  ConversationDeleteResponse,
  ConversationIntentResponse,
  ConversationSearchParams,
  ConversationSearchRequest,
  ConversationSearchResponse,
} from '@/models/conversation';

const ConversationApi = {
  search: async (
    agent_id: string,
    payload: ConversationSearchRequest,
    params?: ConversationSearchParams
  ) => {
    const data = await kyClient.post<ConversationSearchResponse>(
      `v1/agents/${agent_id}/conversations/search`,
      {
        json: payload,
        searchParams: params,
      }
    );
    return data.json();
  },
  /** Soft delete — the conversation moves to Trash and can be restored. */
  delete: async (conversation_id: string) => {
    const data = await kyClient.delete<ConversationDeleteResponse>(
      `v1/conversations/${conversation_id}`
    );
    return data.json();
  },
  /** Moves a trashed conversation back into the inbox. */
  restore: async (conversation_id: string) => {
    const data = await kyClient.post<ConversationDeleteResponse>(
      `v1/conversations/${conversation_id}/restore`
    );
    return data.json();
  },
  /** Permanent removal. The backend rejects this unless the row is trashed. */
  purge: async (conversation_id: string) => {
    const data = await kyClient.delete<ConversationDeleteResponse>(
      `v1/conversations/${conversation_id}/permanent`
    );
    return data.json();
  },
  emptyTrash: async () => {
    const data = await kyClient.delete<ConversationDeleteResponse>(
      'v1/conversations/trash'
    );
    return data.json();
  },
  /**
   * Overrides the classified intent with the rep's own judgement. The backend
   * also reverses the suppression a wrong "Do not contact" caused and records
   * the correction as feedback, so this is not a cosmetic relabel.
   */
  correctIntent: async (
    conversation_id: string,
    payload: { intent: string; note?: string }
  ) => {
    const data = await kyClient.post<ConversationIntentResponse>(
      `v1/conversations/${conversation_id}/intent`,
      { json: payload }
    );
    return data.json();
  },
  assign: async (conversation_id: string, params: any) => {
    const data = await kyClient.post<any>(
      `v1/conversations/${conversation_id}/assign`,
      {
        searchParams: params,
      }
    );
    return data.json();
  },
};

export default ConversationApi;
