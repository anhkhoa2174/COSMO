import { kyClient } from '@/lib/ky';
import type { Contact } from '@/models/contact';
import type {
  ApiResponse,
  BaseQuery,
  PaginateResponse,
} from '@/models/response';
import { useQuery } from '@tanstack/react-query';

export interface PayloadFormInBoundRequest {
  name: string;
  slug: string;
  fields: {
    name: string;
    display_name: string;
    is_required: boolean;
    fallback_value?: string;
    field_type?: string;
    select_options?: string[];
    ui_metadata: any;
  }[];
  ui_metadata: any;
}

export type InboundLeadForm = {
  id: string;
  slug: string;
  name?: string;
};

export type GetContactListData = {
  id: string;
  name: string;
  source: string;
  source_id: string;
  hubspot_id: string | null;
  inbound_lead_forms?: InboundLeadForm[];
  created_at: string;
  updated_at: string;
  contacts: Contact[];
};
type GetContactListResponse = ApiResponse<GetContactListData>;

export const useGetContactListQuery = (contact_list_id: string | null) => {
  return useQuery({
    queryKey: ['contactLists', contact_list_id],
    queryFn: () => contactListApi.detail(contact_list_id as string),
    enabled: !!contact_list_id,
  });
};

type CreateContactListRequest = {
  name: string;
  contact_ids: string[];
  inbound_lead_form_ids?: string[];
};
type CreateContactListResponse = ApiResponse<GetContactListData>;

type UpdateContactListRequest = {
  name: string;
  contact_ids: string[];
  inbound_lead_form_ids?: string[];
};
type UpdateContactListResponse = ApiResponse<GetContactListData>;

type DeleteContactListRequest = {
  ids: string[];
};
type DeleteContactListResponse = ApiResponse<{ message: string }>;

type searchContactListRequest = {
  filter_: Record<string, any>;
};
type SearchContactListResponse = PaginateResponse<{
  entity: GetContactListData;
  creator: string;
  number_of_inbound_forms: number;
  size: number;
}>;

export const contactListApi = {
  search: async (body: searchContactListRequest, params?: BaseQuery) => {
    const data = await kyClient
      .post<SearchContactListResponse>('v1/list-contacts/search', {
        json: body,
        searchParams: params,
      })
      .json();
    return data;
  },
  detail: async (id: string) => {
    const data = await kyClient
      .get<GetContactListResponse>(`v1/list-contacts/${id}`)
      .json();
    return data;
  },
  create: async (body: CreateContactListRequest) => {
    const data = await kyClient
      .post<CreateContactListResponse>('v1/list-contacts', { json: body })
      .json();
    return data;
  },
  update: async (id: string, body: UpdateContactListRequest) => {
    const data = await kyClient
      .patch<UpdateContactListResponse>(`v1/list-contacts/${id}`, {
        json: body,
      })
      .json();
    return data;
  },
  delete: async (body: DeleteContactListRequest) => {
    const data = await kyClient
      .delete<DeleteContactListResponse>(`v1/list-contacts/`, { json: body })
      .json();
    return data;
  },
  getListFormInBound: async (campaign_id: string) => {
    const data = await kyClient
      .get<ApiResponse<any>>(`v1/inbound-lead-forms?campaign_id=${campaign_id}`)
      .json();
    return data;
  },
  getFormInBoundBySlug: async (slug: string) => {
    const data = await kyClient
      .get<ApiResponse<any>>(`v1/inbound-lead-forms/${slug}`)
      .json();
    return data;
  },
  // For the shared form page: visitors there are not signed in.
  getPublicFormBySlug: async (slug: string) => {
    const data = await kyClient
      .get<ApiResponse<any>>(`v1/public/inbound-lead-forms/${slug}`)
      .json();
    return data;
  },
  postFormInBound: async (payload: PayloadFormInBoundRequest) => {
    const data = await kyClient
      .post<ApiResponse<any>>(`v1/inbound-lead-forms`, { json: payload })
      .json();
    return data;
  },
  putFormInBoundBySlug: async (
    slug: string,
    payload: PayloadFormInBoundRequest
  ) => {
    const data = await kyClient
      .put<ApiResponse<any>>(`v1/inbound-lead-forms/${slug}`, { json: payload })
      .json();
    return data;
  },
  deleteFormInBoundBySlug: async (slug: string) => {
    const data = await kyClient
      .delete<ApiResponse<any>>(`v1/inbound-lead-forms/${slug}`)
      .json();
    return data;
  },
  submitFormInBoundBySlug: async (
    slug: string,
    payload: PayloadFormInBoundRequest
  ) => {
    const data = await kyClient
      .post<
        ApiResponse<any>
      >(`v1/public/inbound-lead-forms/${slug}/submit`, { json: payload })
      .json();
    return data;
  },
};
