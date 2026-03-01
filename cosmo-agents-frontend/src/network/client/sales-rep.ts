import { kyClient } from '@/lib/ky';
import type { ApiResponse, BaseQuery } from '@/models/response';
import type {
  // SalesRepCreateRequest,
  SalesRepCreateResponse,
  SalesRepSearchRequest,
  SalesRepSearchResponse,
} from '@/models/sales-rep';

const SalesRepApi = {
  search: async (payload: SalesRepSearchRequest, params?: BaseQuery) => {
    const data = await kyClient.post<SalesRepSearchResponse>('v1/sale-reps/search', {
      json: payload,
      searchParams: params,
    });
    return data.json();
  },
  create: async (payload: any) => {
    const formData = new FormData();

    formData.append('first_name', payload.first_name);
    formData.append('last_name', payload.last_name);
    formData.append('email', payload.email);
    if (payload.calendar_link) {
      formData.append('calendar_link', payload.calendar_link);
    }

    return kyClient
      .post<SalesRepCreateResponse>('v1/sale-reps', {
        body: formData,
      })
      .json();
  },
  delete: (sales_rep_id: string) => {
    return kyClient.delete<ApiResponse<string>>(`v1/sale-reps/${sales_rep_id}`).json();
  },
  update: (sales_rep_id: string, payload: any) => {
    return kyClient
      .patch<SalesRepCreateResponse>(`v1/sale-reps/${sales_rep_id}`, { json: payload })
      .json();
  },
};

export default SalesRepApi;
