import type { ApiResponse, PaginateResponse } from './response';

export type EmailIntent =
  | 'Interested'
  | 'Not interested'
  | 'Request for pricing'
  | 'Request for information'
  | 'Do not contact'
  | 'Out of office'
  | 'Referral'
  | 'Nurture'
  | 'Unknown intent';

export type EmailInbox = {
  added_by: string;
  connected_date: string;
  connected_inbox: string;
  email_provider: string;
};

export type EmailStatistics = {
  bounce_rate: string;
  open_rate: string;
  reply_rate: string;
  sent: string;
};

export type BaseEmail = {
  subject: string;
  content: string;
};

export type Email = BaseEmail & {
  id: string;
  from_email: string;
  to_email: string;
  attachments: string[];
  intents: EmailIntent[];
  gmail_message_id: string;
  conversation_id: string;
  created_at: string;
  updated_at: string;
};

// search
export type EmailSearchRequest = {
  filter: any;
};

type EmailSearchResponseData = {
  entity: Email;
};
export type EmailSearchResponse = PaginateResponse<EmailSearchResponseData>;

// reply
export type EmailReplyRequest = {
  subject?: string;
  content: string;
  cc: string[];
};

export type EmailReplyResponse = ApiResponse<string>;
