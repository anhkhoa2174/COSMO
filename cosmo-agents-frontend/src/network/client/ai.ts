import { useMutation } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';
import { CreateOrganizationData } from './organization';

export type ExtractCompanyInfoData = Omit<
  CreateOrganizationData,
  'name' | 'company_url'
>;
type ExtractCompanyInfoResponse = ApiResponse<ExtractCompanyInfoData>;

export const extractCompanyInfo = (company_url: string) => {
  return kyClient
    .get<ExtractCompanyInfoResponse>(
      `v1/ai/companies/extract?company_url=${company_url}`
    )
    .json();
};

export type ClassifyIntentData = {
  id: number;
  intent: string;
};
type ClassifyIntentRequest = {
  content: string;
};
type ClassifyIntentResponse = ApiResponse<ClassifyIntentData>;

export const classifyIntent = (body: ClassifyIntentRequest) => {
  return kyClient
    .post<ClassifyIntentResponse>('v1/ai/emails/classify-intent', {
      json: body,
      timeout: 60000,
    })
    .json();
};

export const useClassifyIntentMutation = () => {
  return useMutation({
    mutationFn: classifyIntent,
  });
};
