import { useMutation, useQuery } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface EnrollmentCriteria {
  fit_score_threshold: number;
  engagement_score_threshold?: number;
  require_human_approval: boolean;
}

export interface AutomationRuleRead {
  automation_rule_id: string;
  name: string;
  segment_id: string;
  segment_name: string;
  playbook_id: string;
  playbook_name: string;
  enrollment_criteria: EnrollmentCriteria;
  is_active: boolean;
  stats: {
    contacts_enrolled: number;
    contacts_pending_approval: number;
    contacts_in_progress: number;
    contacts_completed: number;
  };
  created_at: string;
  updated_at: string;
}

export interface AutomationRuleCreateRequest {
  name: string;
  segment_id: string;
  playbook_id: string;
  enrollment_criteria: EnrollmentCriteria;
  is_active: boolean;
}

const AutomationRuleApi = {
  list: async () => {
    const data = await kyClient.get('v1/automation-rules');
    return data.json<ApiResponse<AutomationRuleRead[]>>();
  },
  create: async (payload: AutomationRuleCreateRequest) => {
    const data = await kyClient.post('v1/automation-rules', { json: payload });
    return data.json<ApiResponse<AutomationRuleRead>>();
  },
  toggle: async (id: string, is_active: boolean) => {
    const data = await kyClient.patch(`v1/automation-rules/${id}/toggle`, {
      json: { is_active },
    });
    return data.json<ApiResponse<any>>();
  },
};

export default AutomationRuleApi;

export const useAutomationRulesQuery = () => {
  return useQuery({
    queryKey: ['automation-rules'],
    queryFn: () => AutomationRuleApi.list(),
  });
};

export const useCreateAutomationRuleMutation = () => {
  return useMutation({
    mutationFn: (payload: AutomationRuleCreateRequest) =>
      AutomationRuleApi.create(payload),
  });
};

export const useToggleAutomationRuleMutation = () => {
  return useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      AutomationRuleApi.toggle(id, is_active),
  });
};
