import { kyClient } from '@/lib/ky';
import type { Contact } from '@/models/contact';
import type {
  ApiResponse,
  BaseQuery,
  PaginateResponse,
} from '@/models/response';

interface ContactInList {
  entity: Contact;
  creator: string;
  number_of_campaign: number;
  size: number;
}
type GetFieldValue = {
  fields: string[];
  offset: number;
  limit: number;
};

interface FieldItem {
  company?: {
    list: [];
    total: number;
  };
  job_title?: {
    list: [];
    total: number;
  };
}

interface FieldValue {
  list: FieldItem[];
  offset: number;
  limit: number;
}

interface ContactSearchPayload {
  filter: Record<string, any>;
}

const ContactApi = {
  upload: async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const data = await kyClient.post('v2/contacts/import', { body: formData });
    return data.json<any>();
  },
  list: async (payload: ContactSearchPayload, params?: BaseQuery) => {
    // Use v1 to receive full profile/ai_insights/custom_fields for expanded row.
    const data = await kyClient.post('v1/contacts/search', {
      json: payload,
      searchParams: params,
    });
    return data.json<PaginateResponse<ContactInList>>();
  },
  create: async (payload: any) => {
    const data = await kyClient.post('v1/contacts', { json: payload });
    return data.json<any>();
  },
  update: async (contact_id: string, payload: any) => {
    const data = await kyClient.patch(`v1/contacts/${contact_id}`, {
      json: payload,
    });
    return data.json<any>();
  },
  getById: async (contact_id: string) => {
    const data = await kyClient.get(`v1/contact/${contact_id}`);
    return data.json<ApiResponse<Contact>>();
  },
  delete: async (payload: { ids: string[] }) => {
    const data = await kyClient.delete('v1/contacts', { json: payload });
    return data.json<ApiResponse<{ message: string }>>();
  },
  getValue: async (params: GetFieldValue) => {
    const searchParams = new URLSearchParams();

    params?.fields?.forEach((field) => searchParams.append('fields', field));
    searchParams.append('offset', String(params.offset));
    searchParams.append('limit', String(params.limit));

    const response = await kyClient.get('v1/contacts/values', { searchParams });
    return response.json<ApiResponse<FieldValue>>();
  },
  recalculateStatus: async () => {
    const response = await kyClient.post('v1/contacts/recalculate-status');
    return response.json<ApiResponse<{ message: string; updated_count: number }>>();
  },
};

export default ContactApi;

interface ExtractHeadersResponse {
  system: {
    name: string;
    mapping: string;
  }[];
  custom: {
    name: string;
    mapping: string | null;
  }[];
}

async function extractCsvHeaders(file: File) {
  const formData = new FormData();
  formData.append('file', file);
  const response = await kyClient.post(
    'v2/contacts/import/extract-csv-headers',
    {
      body: formData,
    }
  );
  return response.json<ApiResponse<ExtractHeadersResponse>>();
}

async function importCSV(file: File, headers: Record<string, string>) {
  const formData = new FormData();
  formData.append('file', file);
  for (const [key, value] of Object.entries(headers)) {
    formData.append(key, value);
  }
  const res = await kyClient.post('v3/contacts/import', { body: formData });
  return res.json<ApiResponse>();
}

type ImportHubspotRequest = {
  list_ids: string[];
};

async function importHubspot(body: ImportHubspotRequest) {
  const res = await kyClient.post('v2/contacts/import-hubspot', { json: body });
  return res.json<ApiResponse>();
}

export type { ExtractHeadersResponse, ImportHubspotRequest };
export { extractCsvHeaders, importCSV, importHubspot };
