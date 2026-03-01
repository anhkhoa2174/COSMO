import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export type PlaybookType = 'nurture' | 'outreach' | 're_engagement' | 'upsell';
export type StageType = 'email' | 'linkedin' | 'call' | 'wait' | 'conditional';

export interface PlaybookStage {
  id: string;
  order: number;
  name: string;
  type: StageType;
  trigger_conditions: {
    wait_duration: number;
    wait_for_event?: string;
  };
  content_config?: {
    template?: string;
    ai_generation_prompt?: string;
  };
  success_criteria: {
    on_reply: 'advance' | 'complete';
    on_timeout: 'next_stage' | 'pause';
  };
}

export interface CreatePlaybookRequest {
  name: string;
  description: string;
  playbook_type: PlaybookType;
  stages: PlaybookStage[];
}

export interface PlaybookRead {
  playbook_id: string;
  name: string;
  description: string;
  playbook_type: PlaybookType;
  config: {
    stages: PlaybookStage[];
  };
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

const PlaybookApi = {
  create: async (payload: CreatePlaybookRequest) => {
    const data = await kyClient.post('v1/playbooks', { json: payload });
    return data.json<ApiResponse<PlaybookRead>>();
  },
  list: async () => {
    const data = await kyClient.get('v1/playbooks');
    return data.json<ApiResponse<PlaybookRead[]>>();
  },
  getById: async (id: string) => {
    const data = await kyClient.get(`v1/playbooks/${id}`);
    return data.json<ApiResponse<PlaybookRead>>();
  },
  delete: async (id: string) => {
    const data = await kyClient.delete(`v1/playbooks/${id}`);
    return data.json<ApiResponse<{ message: string }>>();
  },
};

export default PlaybookApi;
