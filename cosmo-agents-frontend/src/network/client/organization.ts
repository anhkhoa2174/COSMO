import { useMutation } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import {
  MemberCreateRequest,
  MemberCreateResponse,
  MemberDeletePayload,
  MemberSearchRequest,
  MemberSearchResponse,
} from '@/models/organization';
import type { ApiResponse, BaseQuery } from '@/models/response';

export type Organization = {
  name: string;
  company_url?: string;
  company_description: string;
  company_targeting_persona: string[];
  value_offering: string;
};

const OrganizationApi = {
  searchMember: async (
    organization_id: string,
    payload: MemberSearchRequest,
    params?: BaseQuery
  ) => {
    const data = await kyClient.post<MemberSearchResponse>(
      `v2/organizations/${organization_id}/members/search`,
      {
        json: payload,
        searchParams: params,
      }
    );
    return data.json();
  },
  createMember: async (organization_id: string, data: MemberCreateRequest) => {
    const response = await kyClient.post<MemberCreateResponse>(
      `v2/organizations/${organization_id}/members/invite`,
      {
        json: data,
      }
    );
    return response.json();
  },
  deleteMember: async (
    organization_id: string,
    payload: MemberDeletePayload
  ) => {
    const response = await kyClient.delete<MemberSearchResponse>(
      `v2/organizations/${organization_id}/members/remove`,
      {
        json: payload,
      }
    );
    return response.json;
  },
  list: async (params?: BaseQuery) => {
    const data = await kyClient.get('v1/organizations', {
      searchParams: params,
    });
    return data.json<any>();
  },
  update: async (organization_id: string, payload: Organization) => {
    const data = await kyClient.patch(`v1/organizations/${organization_id}`, {
      json: payload,
    });
    return data.json<any>();
  },
};

export default OrganizationApi;

export type CreateOrganizationData = {
  name: string;
  company_description: string;
  company_targeting_persona: string[];
  value_offering: string;
  company_url: string;
  /** Onboarding step 3 — which CRM the team runs on today. */
  crm: string;
  /** How inbound leads are handled right now (multi-select). */
  lead_handling: string[];
  /** Free-text used when "Other" is ticked in lead_handling. */
  lead_handling_other: string;
};
type CreateOrganizationRequest = CreateOrganizationData;
type CreateOrganizationResponse = ApiResponse<CreateOrganizationData>;

export const createOrganization = (body: CreateOrganizationRequest) => {
  return kyClient
    .post<CreateOrganizationResponse>('v1/organizations', { json: body })
    .json();
};

export const useCreateOrganizationMutation = () => {
  return useMutation({
    mutationFn: createOrganization,
  });
};
