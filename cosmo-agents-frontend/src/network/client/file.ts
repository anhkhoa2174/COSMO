import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface FileItem {
  key: string;
  file_name: string;
  size: number;
  content_type?: string;
  last_modified?: string;
  url?: string;
}

const FileApi = {
  list: async () => {
    const data = await kyClient.get('v1/files/s3/list');
    return data.json<ApiResponse<FileItem[]>>();
  },
  upload: async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const data = await kyClient.post('v1/files/s3', { body: formData });
    return data.json<ApiResponse<any>>();
  },
  getPresignedUrl: async (key: string) => {
    const data = await kyClient.get('v1/files/s3/presigned-url', {
      searchParams: { key },
    });
    return data.json<ApiResponse<{ url: string }>>();
  },
  delete: async (key: string) => {
    const data = await kyClient.delete('v1/files/s3', {
      searchParams: { key },
    });
    return data.json<ApiResponse<{ message: string; key: string }>>();
  },
};

export default FileApi;
