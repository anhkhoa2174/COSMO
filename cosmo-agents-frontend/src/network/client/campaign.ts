import { useMutation, useQuery } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type {
  Campaign,
  CampaignConfig,
  CampaignStatus,
} from '@/models/campaign';
import type { Contact } from '@/models/contact';
import type { EmailIntent } from '@/models/email';
import type {
  ApiResponse,
  BaseQuery,
  PaginateResponse,
} from '@/models/response';

interface CampaignInList {
  entity: Campaign;
  creator: string;
  sent: number;
  reply: number;
  reply_rate: number;
  interested: number;
  interest_rate: number;
}

export interface CampaignCreateRequest {
  name: string;
  playbook: string;
}

export interface CampaignUpdateRequest {
  name: string;
  list_contact_id: string;
  agent_id: string;
  schedule: string | null;
  notification_ids: string[];
  status: CampaignStatus;
}

interface AssignRequest {
  config: CampaignConfig[];
}

const CampaignApi = {
  mergeTags: async (campaign_id: string) => {
    const data = await kyClient
      .get<PaginateResponse<any>>(`v2/campaigns/${campaign_id}/merge-tags`)
      .json();
    return data;
  },
  search: async (payload: any, params?: BaseQuery) => {
    const data = await kyClient.post('v2/campaigns/search', {
      json: payload,
      searchParams: params,
    });
    return data.json<PaginateResponse<CampaignInList>>();
  },
  create: async (payload: CampaignCreateRequest) => {
    const data = await kyClient.post('v1/campaigns', { json: payload });
    return data.json<ApiResponse<Campaign>>();
  },
  update: async (
    campaign_id: string,
    payload: Partial<CampaignUpdateRequest>
  ) => {
    const data = await kyClient.patch(`v1/campaigns/${campaign_id}`, {
      json: payload,
    });
    return data.json<ApiResponse<Campaign>>();
  },
  delete: async (campaign_id: string) => {
    const data = await kyClient.delete(`v1/campaigns/${campaign_id}`);
    return data.json<ApiResponse<string>>();
  },
  assign: async (campaign_id: string, payload: AssignRequest) => {
    const data = await kyClient.post(`v1/campaigns/${campaign_id}/assign`, {
      json: payload,
    });
    return data.json<ApiResponse>();
  },
  uploadKnowledge: async (template_id: string, file: File) => {
    const formData = new FormData();
    formData.append('files', file);
    const data = await kyClient.post(`v2/templates/${template_id}/knowledges`, {
      body: formData,
    });
    return data.json<ApiResponse<any>>();
  },
};

export default CampaignApi;

/* Get campaign detail */
type GetCampaignResponse = ApiResponse<Campaign>;

export const getCampaign = (campaign_id: string) => {
  return kyClient
    .get<GetCampaignResponse>(`v1/campaigns/${campaign_id}`)
    .json();
};

export const useGetCampaignQuery = (campaign_id: string) => {
  return useQuery({
    queryKey: ['campaigns', campaign_id],
    queryFn: () => getCampaign(campaign_id),
  });
};

/* Generate sample response: using for Let AI Reply */
type GenerateSampleResponseRequest = {
  intent: EmailIntent;
  contact_data: Contact;
  outreach_template: string;
};

export type GenerateSampleResponseData = {
  intent: EmailIntent;
  response: string;
};

type GenerateSampleResponseResponse = ApiResponse<GenerateSampleResponseData>;

export const generateSampleResponse = (
  campaign_id: string,
  body: GenerateSampleResponseRequest
) => {
  return kyClient
    .post<GenerateSampleResponseResponse>(
      `v2/campaigns/${campaign_id}/generate-sample-response`,
      {
        json: body,
      }
    )
    .json();
};

export const useGenerateSampleResponseMutation = (campaign_id: string) => {
  return useMutation({
    mutationFn: (body: GenerateSampleResponseRequest) =>
      generateSampleResponse(campaign_id, body),
  });
};

/* Generate reply: using for Let AI Reply and Preview */
type GenerateReplyRequest = {
  conversation: {
    to_email: string;
    content: string;
    status: string;
    from_email: string;
    subject: string;
  }[];
  intent: EmailIntent;
};

export type GenerateReplyData = any;

type GenerateReplyResponse = ApiResponse<GenerateReplyData>;

export const generateReply = (
  campaign_id: string,
  body: GenerateReplyRequest
) => {
  return kyClient
    .post<GenerateReplyResponse>(`v2/campaigns/${campaign_id}/generate-reply`, {
      json: body,
    })
    .json();
};

export const useGenerateReplyMutation = (campaign_id: string) => {
  return useMutation({
    mutationFn: (body: GenerateReplyRequest) =>
      generateReply(campaign_id, body),
  });
};
