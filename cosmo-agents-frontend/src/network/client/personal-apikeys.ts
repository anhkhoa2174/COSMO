import { kyClient } from '@/lib/ky';
import type {
  ApiResponse,
  BaseQuery,
  PaginateResponse,
} from '@/models/response';

export interface PersonalApiKey {
  raw_key?: string;
  entity: {
    id: string;
    user_id: string;
    name: string;
    prefix: string;
    expires_at: string;
    last_used_at: string;
    created_at: string;
    updated_at: string;
  };
}

const PersonalApiKeysApi = {
  list: async ({ params, userId }: { params?: BaseQuery; userId: string }) => {
    const data = await kyClient.get<PaginateResponse<PersonalApiKey>>(
      `v1/users/${userId}/personal-api-keys`,
      {
        searchParams: params,
      }
    );
    return data.json();
  },

  delete: async ({
    personalApiKey,
    userId,
  }: {
    personalApiKey: string;
    userId: string;
  }) => {
    const data = await kyClient.delete<ApiResponse<any>>(
      `v1/users/${userId}/personal-api-keys/${personalApiKey}`
    );
    return data.json();
  },

  create: async ({
    body,
    userId,
    params,
  }: {
    body: { name: string; expires_at: number };
    userId: string;
    params?: BaseQuery;
  }) => {
    const data = await kyClient.post<ApiResponse<PersonalApiKey>>(
      `v1/users/${userId}/personal-api-keys`,
      {
        json: body,
        searchParams: params,
      }
    );
    return data.json();
  },
};

export default PersonalApiKeysApi;
