import { kyClient } from '@/lib/ky';

export interface ChatSession {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
}

export interface ChatSessionMessage {
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

/**
 * Ask COSMO conversation history.
 *
 * The assistant streams from the front end, so these endpoints only remember
 * what was said. Nothing here generates a reply.
 */
const ChatSessionsApi = {
  list: async () => {
    const res = await kyClient.get<{ data: { sessions: ChatSession[] } }>(
      'v2/chat/sessions'
    );
    return (await res.json()).data.sessions;
  },

  /** The first question names the conversation; the server truncates it. */
  create: async (firstQuestion: string) => {
    const res = await kyClient.post<{ data: ChatSession }>('v2/chat/sessions', {
      json: { first_question: firstQuestion },
    });
    return (await res.json()).data;
  },

  messages: async (sessionId: string) => {
    const res = await kyClient.get<{
      data: { session: ChatSession; messages: ChatSessionMessage[] };
    }>(`v2/chat/sessions/${sessionId}/messages`);
    return (await res.json()).data;
  },

  append: async (
    sessionId: string,
    messages: { role: 'user' | 'assistant'; content: string }[]
  ) => {
    await kyClient.post(`v2/chat/sessions/${sessionId}/messages`, {
      json: { messages },
    });
  },

  remove: async (sessionId: string) => {
    await kyClient.delete(`v2/chat/sessions/${sessionId}`);
  },
};

export default ChatSessionsApi;
