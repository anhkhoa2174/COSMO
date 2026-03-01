import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export interface ExtractFromURLRequest {
  url: string;
}

export interface ExtractFromURLResponse {
  url: string;
  extracted_data: Record<string, any>;
  fields_added: string[];
  message: string;
  contact_updated: boolean;
}

const UrlExtractionApi = {
  async extractFromURL(
    contactId: string,
    url: string
  ): Promise<ApiResponse<ExtractFromURLResponse>> {
    const response = await kyClient
      .post(`v1/contacts/${contactId}/extract-from-url`, {
        json: { url },
        timeout: 120000,
      })
      .json<ApiResponse<ExtractFromURLResponse>>();
    return response;
  },
  async extractFromImage(
    contactId: string,
    image: File
  ): Promise<ApiResponse<ExtractFromURLResponse>> {
    const formData = new FormData();
    formData.append('image', image);
    const response = await kyClient
      .post(`v1/contacts/${contactId}/extract-from-image`, {
        body: formData,
        timeout: 120000,
      })
      .json<ApiResponse<ExtractFromURLResponse>>();
    return response;
  },
  // Extract from image without contactId (for create mode)
  async extractFromImagePreview(
    image: File
  ): Promise<ApiResponse<{ extracted_data: Record<string, any> }>> {
    const formData = new FormData();
    formData.append('image', image);
    const response = await kyClient
      .post(`v1/extract-from-image-preview`, {
        body: formData,
        timeout: 120000,
      })
      .json<ApiResponse<{ extracted_data: Record<string, any> }>>();
    return response;
  },
};

export default UrlExtractionApi;
