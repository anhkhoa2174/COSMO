import { cleanText, extractEmailParts, generateId } from '@/helpers';
import { setToken } from '@/helpers/client/cookies';
import ky from 'ky';

const kyClient = ky.create({ prefixUrl: '/api/ai' });

export const getMessage = ({
  content,
  content_type,
  role = 'user',
}: {
  content: string;
  content_type: string;
  role: 'user' | 'assistant';
}) => {
  return {
    content,
    content_type,
    created_at: Date.now(),
    id: generateId(),
    role,
  } as Message;
};

export const getMessageError = (error: any) => {
  return {
    content: error.message || 'Oops! Something went wrong',
    content_type: 'text',
    role: 'assistant',
  } as Message;
};

export const extractOnboardingTypeWithRegex = (
  inputString: string
): string | null => {
  const regex = /"onboarding_type":\s*"(.*?)"/;
  const match = inputString.match(regex);
  return match && match[1] ? match[1] : null;
};

export type Message = {
  bot_id?: string;
  chat_id?: string;
  content: string;
  content_type?: string;
  conversation_id?: string;
  created_at?: number;
  id?: string;
  meta_data?: Record<string, any>;
  reasoning_content?: string;
  role: string;
  section_id?: string;
  type?: string;
  updated_at?: number;
};

const chatApi = {
  postOnboardingAnswer: async ({
    conversationId,
    messages,
  }: {
    conversationId: string;
    messages: Message[];
  }) => {
    try {
      const response = await kyClient.post('agent/chat', {
        json: {
          bot_id: '7511599013192515602',
          conversation_id: conversationId,
          messages,
        },
      });

      if (!response.body) {
        const err = new Error('Response body is empty');
        throw err;
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let partialMessage = '';
      let accumulatedData: any[] = [];

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = (partialMessage + chunk).split('\n');
        partialMessage = lines.pop() || '';

        for (const line of lines) {
          if (!line.startsWith('data:')) continue;
          try {
            const data = JSON.parse(line.slice(5));
            if (data.type === 'answer' && data.role === 'assistant') {
              accumulatedData.push(data);
            }
            if (data?.status === 'completed') {
              const lastItem = accumulatedData[accumulatedData.length - 1];
              if (lastItem) {
                const onboardingType = extractOnboardingTypeWithRegex(
                  lastItem.content
                );
                return onboardingType;
              }
            }
          } catch (parseError) {
            console.error('Error parsing JSON:', parseError);
          }
        }
      }
    } catch (error) {
      console.error('Failed to post onboarding answer:', error);
      return null;
    }
  },
  newChat: async (
    conversationId: string,
    newUserMessages: Message[],
    callback?: (answer: string, lastItem: Message) => void
  ) => {
    let answerTemp = '';
    let subjectTemp = '';
    let contentTemp = '';
    try {
      const response = await kyClient.post('agent/chat', {
        json: {
          conversation_id: conversationId,
          messages: newUserMessages,
        },
        hooks: {
          beforeRequest: [
            (request) => {
              request.headers.set('Content-Type', 'application/json');
            },
          ],
        },
      });

      if (!response.body) {
        const err = new Error('Response body is empty');
        throw err;
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let partialMessage = '';
      let accumulatedData: any[] = [];

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = (partialMessage + chunk).split('\n');
        partialMessage = lines.pop() || '';

        for (const line of lines) {
          if (!line.startsWith('data:')) continue;
          try {
            const data = JSON.parse(line.slice(5));
            if (data.type === 'answer' && data.role === 'assistant') {
              accumulatedData.push(data);
            }
            if (data?.status === 'completed') {
              const lastItem = accumulatedData[accumulatedData.length - 1];
              if (lastItem) {
                const accumulatedText = lastItem.content;
                const { text, subject, content } =
                  extractEmailParts(accumulatedText);
                const answerText = text || accumulatedText;
                callback?.(answerText, lastItem);
                subjectTemp = cleanText(subject);
                contentTemp = cleanText(content);
                answerTemp = answerText;
              }
            }
          } catch (parseError) {
            subjectTemp = '';
            contentTemp = '';
            answerTemp = '';
            console.error('Error parsing JSON:', parseError);
          }
        }
      }
    } catch (error: any) {
      console.error('Error during generate template:', error);
      return { subject: '', content: '', answer: '' };
    } finally {
      return { subject: subjectTemp, content: contentTemp, answer: answerTemp };
    }
  },
  postCreateConversation: async () => {
    try {
      const data = await kyClient.post('agent/conversation').json<any>();
      setToken({
        agent_access_token: data?.agent_access_token,
        session_name: data?.session_name,
      });
      return data?.conversation?.id ?? null;
    } catch (error) {
      console.error('Failed to create conversation:', error);
      return null;
    }
  },
  deleteClearConversation: async (conversationId: string) => {
    if (!conversationId) return false;
    try {
      const data = await kyClient
        .delete(`agent/conversation?conversation_id=${conversationId}`)
        .json<any>();
      return !!data?.success;
    } catch (error) {
      console.error('Failed to close conversation:', error);
      return false;
    }
  },
  getConversation: async (conversationId: string) => {
    if (!conversationId) return [];
    try {
      const data = await kyClient
        .get(`agent/conversation?conversation_id=${conversationId}`)
        .json<any>();
      const message = getMessage({
        content:
          "Greetings! I'm here to help enhance email content for better engagement and clarity.",
        content_type: 'text',
        role: 'assistant',
      });
      setToken({
        agent_access_token: data?.agent_access_token,
        session_name: data?.session_name,
      });
      return [message, ...(data?.messages || [])];
    } catch (error) {
      console.error('Failed to get conversation:', error);
      const message = getMessageError(error);
      return [message];
    }
  },
  botInfo: async () => {
    try {
      const result = await kyClient.get('agent/info').json<any>();
      return result?.data?.data || null;
    } catch (error) {
      console.error('Failed to get bot info:', error);
      return null;
    }
  },
  updateBot: async (datasetIds: string[] | null) => {
    try {
      const result = await kyClient
        .post('agent/update', { json: { dataset_ids: datasetIds } })
        .json<any>();
      return !!result.success;
    } catch (error) {
      console.error('Failed to update bot:', error);
      return false;
    }
  },
  publicBot: async () => {
    try {
      const result = await kyClient.post('agent/public').json<any>();
      return !!result.success;
    } catch (error) {
      console.error('Failed to make bot public:', error);
      return false;
    }
  },
};

export default chatApi;
