import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface LeadForm {
  id: string;
  /** The backend addresses forms by slug (or id); there is no "identifier". */
  slug: string;
  name: string;
  description?: string;
  fields?: any[];
  status?: string;
  submissions_count?: number;
  created_at: string;
  updated_at: string;
}

const LeadFormApi = {
  list: async () => {
    const data = await kyClient.get('v1/inbound-lead-forms');
    return data.json<ApiResponse<LeadForm[]>>();
  },
  getById: async (identifier: string) => {
    const data = await kyClient.get(`v1/inbound-lead-forms/${identifier}`);
    return data.json<ApiResponse<LeadForm>>();
  },
  create: async (payload: Partial<LeadForm>) => {
    const data = await kyClient.post('v1/inbound-lead-forms', { json: payload });
    return data.json<ApiResponse<LeadForm>>();
  },
  update: async (identifier: string, payload: Partial<LeadForm>) => {
    const data = await kyClient.put(`v1/inbound-lead-forms/${identifier}`, { json: payload });
    return data.json<ApiResponse<LeadForm>>();
  },
  delete: async (identifier: string) => {
    const data = await kyClient.delete(`v1/inbound-lead-forms/${identifier}`);
    return data.json<ApiResponse<any>>();
  },
};

export default LeadFormApi;
