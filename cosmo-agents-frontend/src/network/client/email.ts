import { kyClient } from '@/lib/ky';
import type {
  EmailReplyRequest,
  EmailReplyResponse,
  EmailSearchRequest,
  EmailSearchResponse,
} from '@/models/email';
import type {
  ApiResponse,
  BaseQuery,
  PaginateResponse,
} from '@/models/response';

const EmailApi = {
  search: async (payload: EmailSearchRequest, params?: BaseQuery) => {
    const data = await kyClient.post<EmailSearchResponse>('v2/emails/search', {
      searchParams: params,
      json: payload,
    });
    return data.json();
  },
  reply: async (email_id: string, payload: EmailReplyRequest) => {
    const data = await kyClient.post<EmailReplyResponse>(
      `v2/emails/${email_id}/reply`,
      {
        json: payload,
      }
    );
    return data.json();
  },
};

export default EmailApi;

function getEmails(params?: any) {
  return kyClient
    .extend({
      headers: {
        Authorization: 'Basic cm9ja3NoaXA6cm9ja3NoaXA=',
      },
    })
    .get<PaginateResponse>('v1/emails', {
      searchParams: params,
    })
    .json();
}

function getEmail(email_id: string) {
  return kyClient
    .extend({
      headers: {
        Authorization: 'Basic cm9ja3NoaXA6cm9ja3NoaXA=',
      },
    })
    .get<ApiResponse>(`v1/emails/${email_id}`)
    .json();
}

export { getEmails, getEmail };

/**
 * Regenerates the AI reply for a conversation.
 *
 * The back end composes it from the label stored on the thread when a person
 * has corrected one, and only falls back to classifying the message afresh
 * when nobody has. That distinction is the whole point of offering this after
 * a correction: re-classifying would read the same message and return the same
 * label the representative had just rejected.
 */
export async function regenerateAIReply(conversationId: string) {
  const res = await kyClient.post<{
    data: {
      from_email: string;
      to_email: string;
      subject: string;
      content: string;
      type: string;
    };
  }>('v1/ai/emails/reply', { json: { conversation_id: conversationId } });
  return (await res.json()).data;
}
