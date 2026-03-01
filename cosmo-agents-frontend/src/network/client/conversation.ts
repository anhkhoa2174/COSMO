import { kyClient } from '@/lib/ky';
import type {
  ConversationDeleteResponse,
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
  delete: async (conversation_id: string) => {
    const data = await kyClient.delete<ConversationDeleteResponse>(
      `v1/conversations/${conversation_id}`
    );
    return data.json();
  },
  assign: async (conversation_id: string, params: any) => {
    const data = await kyClient.post<any>(`v1/conversations/${conversation_id}/assign`, {
      searchParams: params,
    });
    return data.json();
  },
};

export default ConversationApi;
