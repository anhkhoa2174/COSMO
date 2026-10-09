import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

// ============ Types ============

export type ConversationState =
  | 'COLD'
  | 'NO_REPLY'
  | 'REPLIED'
  | 'POST_MEETING'
  | 'DROPPED';
export type ContextLevel = 'LOW' | 'MEDIUM' | 'HIGH';
export type OutreachIntent =
  | 'INTRO'
  | 'FOLLOW_UP'
  | 'RE_ENGAGE'
  | 'POST_MEETING';
export type OutreachScenario =
  | 'role_based'
  | 'industry_based'
  | 'no_reply_followup'
  | 'post_reply'
  | 'post_meeting'
  | 're_engage';
export type LastOutcome =
  | 'none'
  | 'sent'
  | 'no_reply'
  | 'replied'
  | 'meeting_booked'
  | 'meeting_done'
  | 'dropped';
export type NextStep =
  | 'SEND' // Send initial message
  | 'FOLLOW_UP_1' // Follow-up #1 (Day 4-5)
  | 'FOLLOW_UP_2' // Follow-up #2 (Day 9-12)
  | 'SET_MEETING' // Propose meeting
  | 'FOLLOW_UP_MEETING_1' // Meeting confirmation follow-up #1
  | 'FOLLOW_UP_MEETING_2' // Meeting confirmation follow-up #2
  | 'PREPARE_MEETING' // Prepare meeting materials
  | 'WAIT' // Wait for response / wait for meeting day
  | 'FOLLOW_UP' // Follow-up deal (post-meeting)
  | 'DROP'; // Drop contact
export type MeetingStatus = 'scheduled' | 'completed' | 'cancelled' | 'no_show';
export type InteractionChannel =
  | 'LinkedIn'
  | 'Email'
  | 'Call'
  | 'Meeting'
  | 'Note';
export type Sentiment = 'positive' | 'neutral' | 'negative';
export type FeedbackAction =
  | 'used_draft'
  | 'modified_draft'
  | 'wrote_own'
  | 'skipped';
export type FeedbackOutcome =
  | 'no_action'
  | 'sent'
  | 'replied'
  | 'meeting_booked'
  | 'meeting_done'
  | 'dropped';

export interface OutreachState {
  id: string;
  user_id: string;
  contact_id: string;
  conversation_state: ConversationState;
  context_level: ContextLevel;
  outreach_intent: OutreachIntent;
  scenario: OutreachScenario;
  message_draft?: string;
  last_outcome: LastOutcome;
  next_step: NextStep;
  last_interaction_at?: string;
  days_since_last_interaction: number;
  followup_count: number;
  max_followups: number;
  created_at: string;
  updated_at: string;
}

export interface InteractionLog {
  id: string;
  user_id: string;
  contact_id: string;
  channel: InteractionChannel;
  direction: 'outgoing' | 'incoming' | 'internal';
  content: string;
  subject?: string;
  url?: string;
  attachments?: unknown[];
  sentiment?: Sentiment;
  timestamp: string;
  created_at: string;
  updated_at: string;
}

export interface Meeting {
  id: string;
  user_id: string;
  contact_id: string;
  title?: string;
  time: string;
  duration_minutes: number;
  channel: string;
  location?: string;
  meeting_url?: string;
  status: MeetingStatus;
  note?: string;
  outcome?: string;
  next_steps?: string;
  meeting_content?: string;
  meeting_prep?: string;
  participants?: unknown[];
  created_at: string;
  updated_at: string;
}

export interface ContactSummary {
  id: string;
  name?: string;
  email?: string;
  company?: string;
  job_title?: string;
  linkedin_url?: string;
}

export interface OutreachSuggestion {
  contact: ContactSummary;
  state?: OutreachState;
  type: 'cold' | 'followup';
  next_step: NextStep;
  days_since: number;
  message_draft?: string;
}

export type Language = 'en' | 'vi';

export interface GenerateDraftResponse {
  contact_id: string;
  draft: string;
  scenario: OutreachScenario;
  context_level: ContextLevel;
  state: OutreachState;
  notes?: InteractionLog[]; // Team notes for context
  language?: Language;
}

export interface UpdateOutreachResponse {
  contact_id: string;
  previous_state: ConversationState;
  new_state: ConversationState;
  next_step: NextStep;
  state: OutreachState;
}

// Feedback Loop Types (Task 8)
export interface ScenarioStats {
  scenario: OutreachScenario;
  context_level: ContextLevel;
  total_suggested: number;
  drafts_used: number;
  drafts_modified: number;
  wrote_own: number;
  skipped: number;
  total_sent: number;
  total_replied: number;
  total_meetings: number;
  reply_rate: number;
  meeting_rate: number;
  draft_usage_rate: number;
  avg_days_to_reply: number;
}

export interface FeedbackStats {
  total_suggestions: number;
  total_actions: number;
  total_sent: number;
  total_replied: number;
  total_meetings: number;
  draft_used_count: number;
  draft_modified_count: number;
  wrote_own_count: number;
  skipped_count: number;
  overall_reply_rate: number;
  overall_meeting_rate: number;
  draft_usage_rate: number;
  avg_days_to_reply: number;
  insights: string[];
}

// ============ API Client ============

const OutreachApi = {
  /**
   * Get suggested contacts for outreach
   * @param type - 'cold', 'followup', or 'mixed'
   * @param limit - Maximum number of contacts to return
   */
  suggestOutreach: async (
    type: 'cold' | 'followup' | 'mixed' = 'mixed',
    limit = 10
  ) => {
    const data = await kyClient.get(`v1/outreach/suggest`, {
      searchParams: { type, limit },
    });
    return data.json<ApiResponse<OutreachSuggestion[]>>();
  },

  /**
   * Generate outreach draft for a contact
   * @param contactId - Contact ID
   * @param language - Language for the draft ('en' for English, 'vi' for Vietnamese)
   */
  generateDraft: async (contactId: string, language: Language = 'vi') => {
    const data = await kyClient.post(
      `v1/outreach/contacts/${contactId}/draft`,
      {
        json: { language },
      }
    );
    return data.json<ApiResponse<GenerateDraftResponse>>();
  },

  /**
   * Update outreach state after an event
   * @param contactId - Contact ID
   * @param event - Event type (sent, replied, no_reply, meeting_booked, meeting_confirmed, no_confirmation, meeting_done, drop)
   * @param options - Additional options
   */
  updateOutreach: async (
    contactId: string,
    event:
      | 'sent'
      | 'replied'
      | 'no_reply'
      | 'meeting_booked'
      | 'meeting_confirmed'
      | 'no_confirmation'
      | 'meeting_done'
      | 'drop',
    options?: {
      content?: string;
      channel?: InteractionChannel;
      sentiment?: Sentiment;
    }
  ) => {
    const data = await kyClient.post(
      `v1/outreach/contacts/${contactId}/update`,
      {
        json: {
          event,
          content: options?.content,
          channel: options?.channel || 'LinkedIn',
          sentiment: options?.sentiment,
        },
      }
    );
    return data.json<ApiResponse<UpdateOutreachResponse>>();
  },

  /**
   * Get outreach state for a contact
   * @param contactId - Contact ID
   */
  getOutreachState: async (contactId: string) => {
    const data = await kyClient.get(`v1/outreach/contacts/${contactId}/state`);
    return data.json<ApiResponse<OutreachState>>();
  },

  /**
   * Get interaction history for a contact
   * @param contactId - Contact ID
   * @param limit - Maximum number of interactions to return
   */
  getInteractionHistory: async (contactId: string, limit = 20) => {
    const data = await kyClient.get(
      `v1/outreach/contacts/${contactId}/interactions`,
      {
        searchParams: { limit },
      }
    );
    return data.json<ApiResponse<InteractionLog[]>>();
  },

  /**
   * Add interaction to conversation history
   * @param contactId - Contact ID
   * @param input - Interaction details
   */
  addInteraction: async (
    contactId: string,
    input: {
      content: string;
      role: 'me' | 'client';
      channel?: InteractionChannel;
      sentiment?: Sentiment;
    }
  ) => {
    const data = await kyClient.post(
      `v1/outreach/contacts/${contactId}/interactions`,
      {
        json: input,
      }
    );
    return data.json<ApiResponse<InteractionLog>>();
  },

  /**
   * Create a new meeting
   * @param input - Meeting details
   */
  createMeeting: async (input: {
    contact_id: string;
    title?: string;
    time: string;
    duration_minutes?: number;
    channel?: string;
    location?: string;
    meeting_url?: string;
    note?: string;
  }) => {
    const data = await kyClient.post('v1/outreach/meetings', {
      json: input,
    });
    return data.json<ApiResponse<Meeting>>();
  },

  /**
   * Update a meeting
   * @param meetingId - Meeting ID
   * @param input - Update details
   */
  updateMeeting: async (
    meetingId: string,
    input: {
      status?: MeetingStatus;
      note?: string;
      outcome?: string;
      next_steps?: string;
      meeting_content?: string;
    }
  ) => {
    const data = await kyClient.patch(`v1/outreach/meetings/${meetingId}`, {
      json: input,
    });
    return data.json<ApiResponse<Meeting>>();
  },

  /**
   * Generate meeting prep document (talking points, discovery questions) using AI
   * This is generated BEFORE the meeting based on conversation history and contact info
   * @param meetingId - Meeting ID
   * @param language - Language for the prep ('en' for English, 'vi' for Vietnamese)
   */
  generateMeetingPrep: async (meetingId: string, language: Language = 'vi') => {
    const data = await kyClient.post(
      `v1/outreach/meetings/${meetingId}/generate-prep`,
      {
        json: { language },
        timeout: 130_000, // 130s - slightly longer than backend 120s timeout
      }
    );
    return data.json<ApiResponse<Meeting>>();
  },

  /**
   * Get meetings for a contact
   * @param contactId - Contact ID
   */
  getMeetings: async (contactId: string) => {
    const data = await kyClient.get(
      `v1/outreach/contacts/${contactId}/meetings`
    );
    return data.json<ApiResponse<Meeting[]>>();
  },

  /**
   * Get all upcoming scheduled meetings for the authenticated user
   */
  getAllMeetings: async () => {
    const data = await kyClient.get('v1/outreach/meetings');
    return data.json<ApiResponse<Meeting[]>>();
  },

  /**
   * Delete a meeting
   * @param meetingId - Meeting ID
   */
  deleteMeeting: async (meetingId: string) => {
    const data = await kyClient.delete(`v1/outreach/meetings/${meetingId}`);
    return data.json<ApiResponse<{ message: string }>>();
  },

  // ============ Feedback Loop API (Task 8) ============

  /**
   * Record what BD actually did with a suggestion
   * @param feedbackId - Feedback record ID
   * @param action - Action taken (used_draft, modified_draft, wrote_own, skipped)
   * @param actualContent - The actual message sent (optional)
   */
  recordFeedbackAction: async (
    feedbackId: string,
    action: FeedbackAction,
    actualContent?: string
  ) => {
    const data = await kyClient.post('v1/outreach/feedback/action', {
      json: {
        feedback_id: feedbackId,
        action,
        actual_content: actualContent,
      },
    });
    return data.json<ApiResponse<{ message: string }>>();
  },

  /**
   * Record the outcome of an outreach attempt
   * @param contactId - Contact ID
   * @param outcome - Outcome (sent, replied, meeting_booked, meeting_done, dropped)
   * @param sentiment - Sentiment for replies (optional)
   */
  recordFeedbackOutcome: async (
    contactId: string,
    outcome: FeedbackOutcome,
    sentiment?: Sentiment
  ) => {
    const data = await kyClient.post('v1/outreach/feedback/outcome', {
      json: {
        contact_id: contactId,
        outcome,
        sentiment,
      },
    });
    return data.json<ApiResponse<{ message: string }>>();
  },

  /**
   * Get overall feedback statistics with insights
   */
  getFeedbackStats: async () => {
    const data = await kyClient.get('v1/outreach/feedback/stats');
    return data.json<ApiResponse<FeedbackStats>>();
  },

  /**
   * Get statistics grouped by scenario
   */
  getScenarioStats: async () => {
    const data = await kyClient.get('v1/outreach/feedback/scenarios');
    return data.json<ApiResponse<ScenarioStats[]>>();
  },

  // ============ Contact Notes API (Team Conversation History) ============

  /**
   * Add a note to a contact
   * @param contactId - Contact ID
   * @param content - Note content
   */
  addNote: async (contactId: string, content: string) => {
    const data = await kyClient.post(
      `v1/outreach/contacts/${contactId}/notes`,
      {
        json: { content },
      }
    );
    return data.json<ApiResponse<InteractionLog>>();
  },

  /**
   * Get notes for a contact
   * @param contactId - Contact ID
   * @param limit - Maximum number of notes to return
   */
  getNotes: async (contactId: string, limit = 50) => {
    const data = await kyClient.get(`v1/outreach/contacts/${contactId}/notes`, {
      searchParams: { limit },
    });
    return data.json<ApiResponse<InteractionLog[]>>();
  },

  /**
   * Update a note
   * @param contactId - Contact ID
   * @param noteId - Note ID
   * @param content - New content
   */
  updateNote: async (contactId: string, noteId: string, content: string) => {
    const data = await kyClient.patch(
      `v1/outreach/contacts/${contactId}/notes/${noteId}`,
      {
        json: { content },
      }
    );
    return data.json<ApiResponse<InteractionLog>>();
  },

  /**
   * Delete a note
   * @param contactId - Contact ID
   * @param noteId - Note ID
   */
  deleteNote: async (contactId: string, noteId: string) => {
    const data = await kyClient.delete(
      `v1/outreach/contacts/${contactId}/notes/${noteId}`
    );
    return data.json<ApiResponse<{ message: string }>>();
  },
};

export default OutreachApi;
