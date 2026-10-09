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
  /** The caller's own groups holding this conversation. */
  group_ids?: string[];
  cmetadata?: {
    ai_reply?: {
      draft_content: string;
      intent: string;
      generated_by: string;
      generated_at: string;
    };
    intent_detail?: {
      intent: string;
      confidence?: number;
      reasoning?: string;
      email_id?: string;
      /**
       * Set once a human has overridden the classifier. The machine's own
       * label, confidence and quoted evidence move to the original_* fields:
       * they argued for a conclusion that no longer stands, so they are kept
       * for audit but never shown as support for the corrected label.
       */
      corrected_by_user?: boolean;
      corrected_at?: string;
      correction_note?: string;
      original_intent?: string;
      original_confidence?: number;
      original_reasoning?: string;
    };
    [key: string]: unknown;
  };
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
export type ConversationSearchResponse =
  PaginateResponse<ConversationSearchResponseData>;

// delete
export type ConversationDeleteResponse = ApiResponse<string>;

// intent correction
export type ConversationIntentCorrection = {
  conversation_id: string;
  intent: string;
  previous_intent?: string;
  changed: boolean;
  /**
   * True when an AI draft written for the previous intent is still attached.
   * The backend deliberately leaves it alone — it may hold the rep's edits —
   * and reports it here so the UI can offer to regenerate instead of doing so
   * behind the rep's back.
   */
  draft_stale: boolean;
  /** '' | 'suppressed' | 'unsuppressed' — whether do_not_contact moved. */
  suppression_changed?: string;
};

export type ConversationIntentResponse =
  ApiResponse<ConversationIntentCorrection>;
