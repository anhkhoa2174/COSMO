import { kyClient } from '@/lib/ky';
import type {
  EmailReplyRequest,
  EmailReplyResponse,
  EmailSearchRequest,
  EmailSearchResponse,
} from '@/models/email';
import type { ApiResponse, BaseQuery, PaginateResponse } from '@/models/response';

const EmailApi = {
  search: async (payload: EmailSearchRequest, params?: BaseQuery) => {
    const data = await kyClient.post<EmailSearchResponse>('v2/emails/search', {
      searchParams: params,
      json: payload,
    });
    return data.json();
  },
  reply: async (email_id: string, payload: EmailReplyRequest) => {
    const data = await kyClient.post<EmailReplyResponse>(`v2/emails/${email_id}/reply`, {
      json: payload,
    });
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
