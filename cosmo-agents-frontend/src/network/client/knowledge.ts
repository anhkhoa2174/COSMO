import { kyClient } from '@/lib/ky';
import type { ApiResponse, BaseQuery } from '@/models/response';

const KnowledgeApi = {
  upload: async (files: File[]) => {
    const formData = new FormData();

    files.forEach((file) => {
      formData.append('files', file);
    });

    const data = await kyClient.post('v1/knowledge/upload', { body: formData });
    return data.json<any>();
  },

  list: async (params?: BaseQuery) => {
    const data = await kyClient.get<ApiResponse<any[]>>('v1/knowledge', { searchParams: params });
    return data.json();
  },

  delete: async (knowledgeId: string) => {
    const data = await kyClient.delete(`v1/knowledge/${knowledgeId}`);
    return data.json<any>();
  },

  getFile: async (key: string) => {
    const res = await kyClient.get(`v1/files/s3?key=${key}`);
    return res.blob();
  },
};

export const KnowledgeApiV2 = {
  upload: async (files: File[]) => {
    const formData = new FormData();

    files.forEach((file) => {
      formData.append('files', file);
    });

    const data = await kyClient.post('v2/knowledge/upload', { body: formData });
    return data.json<any>();
  },

  list: async (params?: BaseQuery) => {
    const data = await kyClient.get<ApiResponse<any[]>>('v2/knowledge', { searchParams: params });
    return data.json();
  },

  delete: async (knowledgeId: string) => {
    const data = await kyClient.delete(`v2/knowledge/${knowledgeId}`);
    return data.json<any>();
  },

  getFile: async (key: string) => {
    const res = await kyClient.get(`v1/files/s3?key=${key}`);
    return res.blob();
  },
};

export default KnowledgeApi;
