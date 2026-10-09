import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

/** The colours a group may take; the server rejects anything else. */
export const GROUP_COLORS = [
  'slate',
  'red',
  'orange',
  'amber',
  'green',
  'teal',
  'blue',
  'violet',
  'pink',
] as const;

export type GroupColor = (typeof GROUP_COLORS)[number];

/** A user's own label for AI Inbox conversations. Never shared. */
export interface ConversationGroup {
  id: string;
  user_id: string;
  name: string;
  color: GroupColor;
  created_at: string;
  updated_at: string;
  /** Conversations in the group that are not in the trash. */
  conversation_count?: number;
}

const ConversationGroupApi = {
  list: async () => {
    const res = await kyClient.get('v1/conversation-groups');
    return (await res.json<ApiResponse<ConversationGroup[]>>()).data ?? [];
  },
  create: async (payload: { name: string; color?: GroupColor }) => {
    const res = await kyClient.post('v1/conversation-groups', {
      json: payload,
    });
    return (await res.json<ApiResponse<ConversationGroup>>()).data;
  },
  update: async (
    id: string,
    payload: { name?: string; color?: GroupColor }
  ) => {
    const res = await kyClient.patch(`v1/conversation-groups/${id}`, {
      json: payload,
    });
    return (await res.json<ApiResponse<ConversationGroup>>()).data;
  },
  remove: async (id: string) => {
    await kyClient.delete(`v1/conversation-groups/${id}`);
  },
  addConversation: async (groupId: string, conversationId: string) => {
    await kyClient.post(`v1/conversation-groups/${groupId}/conversations`, {
      json: { conversation_id: conversationId },
    });
  },
  removeConversation: async (groupId: string, conversationId: string) => {
    await kyClient.delete(
      `v1/conversation-groups/${groupId}/conversations/${conversationId}`
    );
  },
};

export default ConversationGroupApi;
