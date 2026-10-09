import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface ResearchSuggestion {
  category: string;
  data_point: string;
  why_important: string;
  priority: string;
  where_to_find: string;
}

export interface ContactEnrichmentResponse {
  contact_id: string;
  insights_generated: number;
  embedding_created: boolean;
  confidence_avg: number;
  ai_insights: {
    suspected_pain_points: Array<{
      pain_point: string;
      confidence: number;
      evidence: string[];
      detected_at: string;
      status: string;
    }>;
    suspected_goals: Array<{
      goal: string;
      confidence: number;
      evidence: string[];
      detected_at: string;
      status: string;
    }>;
    buying_signals: Array<{
      signal: string;
      strength: string;
      occurred_at: string;
      recency_score: number;
    }>;
    decision_style: {
      type: string;
      confidence: number;
      evidence: string[];
      key_decision_factors: string[];
    };
    communication_preferences: {
      preferred_channel: string;
      best_time_to_contact: string;
      confidence: number;
    };
    anticipated_objections: Array<{
      objection: string;
      likelihood: number;
      suggested_response: string;
    }>;
    research_suggestions?: ResearchSuggestion[];
    avg_confidence: number;
  };
}

export interface SegmentScoreResult {
  segmentation_id: string;
  segmentation_name: string;
  fit_score: number;
  score_breakdown: Record<string, number>;
  passes_filters: boolean;
}

export interface CalculateScoresResponse {
  contact_id: string;
  segments_evaluated: number;
  segments_matched: number;
  scores: SegmentScoreResult[];
  priority_score: number;
}

export interface ContactSearchResult {
  contact_id: string;
  similarity: number;
  name?: string;
  first_name?: string;
  last_name?: string;
  company?: string;
  job_title?: string;
  email?: string;
  metadata?: Record<string, any>;
}

export interface VectorSearchContactsResponse {
  query: string;
  results: ContactSearchResult[];
  count: number;
}

export interface ValidateInsightRequest {
  insight_type: 'pain_point' | 'goal' | 'objection' | 'signal' | string;
  insight_text: string;
  validation: 'confirmed' | 'rejected';
  confirmed_data?: Record<string, any>;
}

export interface MeetingBriefInteraction {
  type: string;
  date: string;
  summary: string;
  sentiment: string;
}

export interface MeetingBriefFact {
  category: string;
  facts: string[];
}

export interface MeetingBriefSegment {
  name: string;
  fit_score: number;
}

export interface MeetingBriefResponse {
  last_interactions: MeetingBriefInteraction[];
  confirmed_facts: MeetingBriefFact[];
  segments: MeetingBriefSegment[];
  talking_points: string[];
  discovery_questions: string[];
  risk_flags: string[];
}

const IntelligenceApi = {
  // Enrich a contact with AI insights and generate embeddings
  enrichContact: async (contactId: string, forceRefresh = false) => {
    // Backend route: /v1/contacts/{id}/enrich
    const data = await kyClient.post(`v1/contacts/${contactId}/enrich`, {
      json: { force_refresh: forceRefresh },
    });
    return data.json<ApiResponse<ContactEnrichmentResponse>>();
  },

  // Calculate segment scores for a contact
  calculateScores: async (contactId: string, segmentationIds?: string[]) => {
    // Backend route: /v1/contacts/{id}/calculate-scores
    const data = await kyClient.post(
      `v1/contacts/${contactId}/calculate-scores`,
      {
        json: {
          segmentation_ids: segmentationIds || [],
        },
      }
    );
    return data.json<ApiResponse<CalculateScoresResponse>>();
  },

  // Search contacts using vector similarity
  vectorSearch: async (query: string, limit = 10, threshold = 1.0) => {
    const data = await kyClient.post('v1/intelligence/vector-search/contacts', {
      json: {
        query,
        limit,
        threshold,
      },
    });
    return data.json<ApiResponse<VectorSearchContactsResponse>>();
  },

  // Add research finding to contact
  // Add research finding to contact (writes to profile.research_findings + custom_fields on backend)
  addResearchFinding: async (
    contactId: string,
    finding: {
      category: string;
      field_name: string;
      value: string;
      source?: string;
      priority?: string;
      why_important?: string;
    }
  ) => {
    const data = await kyClient.post(
      `v1/contacts/${contactId}/research-findings`,
      {
        json: finding,
      }
    );
    return data.json<ApiResponse<any>>();
  },

  // Validate or reject an AI insight (moves to confirmed_facts or removes)
  validateInsight: async (
    contactId: string,
    payload: ValidateInsightRequest
  ) => {
    const data = await kyClient.post(
      `v1/contacts/${contactId}/insights/validate`,
      {
        json: payload,
      }
    );
    return data.json<ApiResponse<any>>();
  },

  // Re-embed all contacts (best-effort async)
  reEmbedAllContacts: async () => {
    const data = await kyClient.post('v1/contacts/re-embed-all');
    return data.json<ApiResponse<any>>();
  },

  // Generate meeting brief with AI insights
  generateMeetingBrief: async (contactId: string) => {
    const data = await kyClient.post(
      `v1/contacts/${contactId}/generate-meeting-brief`
    );
    return data.json<ApiResponse<MeetingBriefResponse>>();
  },
};

export default IntelligenceApi;
