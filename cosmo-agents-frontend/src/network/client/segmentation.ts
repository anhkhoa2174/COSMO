import { useQuery } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface SegmentationResponse {
  id: string;
  name: string;
  description?: string;
  is_active: boolean;
}

const SegmentationApi = {
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
