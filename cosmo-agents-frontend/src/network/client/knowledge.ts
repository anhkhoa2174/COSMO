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
    const data = await kyClient.get<ApiResponse<any[]>>('v1/knowledge', {
      searchParams: params,
    });
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

/**
 * What a document is about. Retrieval searches the matching type first when
 * drafting a reply — the pricing sheet for a pricing question.
 */
export type KnowledgeType =
  | 'pricing'
  | 'product'
  | 'case_study'
  | 'faq'
  | 'other';

export const KNOWLEDGE_TYPES: { value: KnowledgeType; label: string }[] = [
  { value: 'other', label: 'General' },
  { value: 'pricing', label: 'Pricing' },
  { value: 'product', label: 'Product' },
  { value: 'faq', label: 'FAQ' },
  { value: 'case_study', label: 'Case study' },
];

export const KnowledgeApiV2 = {
  upload: async (files: File[], type: KnowledgeType = 'other') => {
    const formData = new FormData();

    files.forEach((file) => {
      formData.append('files', file);
    });
    formData.append('type', type);

    const data = await kyClient.post('v2/knowledge/upload', { body: formData });
    return data.json<any>();
  },

  list: async (params?: BaseQuery) => {
    const data = await kyClient.get<ApiResponse<any[]>>('v2/knowledge', {
      searchParams: params,
    });
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
