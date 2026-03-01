import type { Email, EmailIntent } from './email';
import type { ApiResponse, BaseQuery, PaginateResponse } from './response';

export type Conversation = {
  id: string;
  user_id: string;
  labels: string[];
  replied: boolean;
  campaign_id: string;
  assignee_id: string;
  intents: EmailIntent[];
  status: 'read' | 'unread';
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
};

// search
export type ConversationSearchParams = BaseQuery & {
  newest?: 'true' | 'fasle';
  conversation_type?: 'sent' | 'assign_to_ai' | 'assign_to_human';
};

export type ConversationSearchRequest = {
  filter: any;
};

type ConversationSearchResponseData = {
  entity: Conversation;
  latest_email: Omit<Email, 'conversation_id' | 'gmail_message_id'>;
};
export type ConversationSearchResponse = PaginateResponse<ConversationSearchResponseData>;

// delete
export type ConversationDeleteResponse = ApiResponse<string>;
