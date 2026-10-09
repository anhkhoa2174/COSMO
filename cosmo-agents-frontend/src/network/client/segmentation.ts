import { useQuery } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface SegmentationResponse {
  id: string;
  name: string;
  description?: string;
  is_active: boolean;
}

export interface CreateSegmentationRequest {
  name: string;
  description?: string;
  /** 1–10; the backend validates the range. */
  priority?: number;
  criteria?: Record<string, unknown>;
  icp_definition?: Record<string, unknown>;
  is_active?: boolean;
}

const SegmentationApi = {
  create: async (payload: CreateSegmentationRequest) => {
    const data = await kyClient.post('v1/segmentations', { json: payload });
    return data.json<ApiResponse<SegmentationResponse>>();
  },
  list: async (onlyActive = true) => {
    const data = await kyClient.get('v1/segmentations', {
      searchParams: { active: onlyActive ? 'true' : 'false' },
    });
    return data.json<ApiResponse<SegmentationResponse[]>>();
  },
  delete: async (id: string) => {
    const data = await kyClient.delete(`v1/segmentations/${id}`);
    return data.json<ApiResponse<{ message: string }>>();
  },
};

export default SegmentationApi;

export const useSegmentationsQuery = () => {
  return useQuery({
    queryKey: ['segmentations'],
    queryFn: () => SegmentationApi.list(true),
  });
};
