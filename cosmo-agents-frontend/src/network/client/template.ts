import { useCallback } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { Contact } from '@/models/contact';
import type { EmailIntent } from '@/models/email';
import type { ApiResponse } from '@/models/response';

// Get Template
export type GetTemplateData = {
  id: string;
  type: string;
  subject: string;
  intent_type?: EmailIntent;
  content: string;
  send_after: number;
  knowledges: {
    id: string;
    embedding_gid: string;
  }[];
};
type GetTemplateResponse = ApiResponse<GetTemplateData>;

export const getTemplate = (template_id: string) => {
  return kyClient.get<GetTemplateResponse>(`v2/templates/${template_id}`).json();
};

export const useGetTemplateQuery = (template_id?: string) => {
  return useQuery({
    queryKey: ['templates', template_id],
    queryFn: () => getTemplate(template_id as string),
    enabled: !!template_id,
  });
};

// Create Template
export const createTemplate = (campaign_id: string, body: Partial<GetTemplateData>) => {
  return kyClient.post<GetTemplateResponse>(`v3/campaigns/${campaign_id}/templates/external`, { json: body }).json();
};

// Update Template
export const updateTemplate = (template_id: string, body: Partial<GetTemplateData>) => {
  return kyClient.patch<GetTemplateResponse>(`v2/templates/${template_id}`, { json: body }).json();
};

export const useUpdateTemplateMutation = (template_id: string) => {
  return useMutation({
    mutationFn: (data: Partial<GetTemplateData>) => updateTemplate(template_id, data),
  });
};

// Get Draft Template
export type GetDraftTemplateData = {
  id: string;
  intent: EmailIntent;
  campaign_id: string;
  template_id: string;
  created_at: string;
  updated_at: string;
  template: {
    type: string;
    subject: string;
    content: string;
  };
};
type GetDraftTemplateResponse = ApiResponse<GetDraftTemplateData>;

export const getDraftTemplate = (draft_template_id: string) => {
  return kyClient.get<GetDraftTemplateResponse>(`v2/draft-templates/${draft_template_id}`).json();
};

export const useGetDraftTemplateQuery = (draft_template_id?: string) => {
  return useQuery({
    queryKey: ['draft-templates', draft_template_id],
    queryFn: () => getDraftTemplate(draft_template_id as string),
    enabled: !!draft_template_id,
  });
};

// Generate Draft Template
export type GenerateDraftTemplateData = GetDraftTemplateData;
type GenerateDraftTemplateResponse = ApiResponse<GenerateDraftTemplateData>;

export const generateDraftTemplate = (campaign_id: string, intent_type: EmailIntent) => {
  return kyClient
    .get<GenerateDraftTemplateResponse>(`v2/campaigns/${campaign_id}/draft-templates`, {
      searchParams: { intent: intent_type },
    })
    .json();
};

export const useGenerateDraftTemplateQuery = (
  campaign_id: string,
  intent_type: EmailIntent,
  enabled: boolean
) => {
  return useQuery({
    queryKey: ['campaigns', campaign_id, 'draft-templates', intent_type],
    queryFn: () => generateDraftTemplate(campaign_id, intent_type),
    enabled,
  });
};

// Operation
type Operation<OInput = any> = {
  id: string;
  name: 'generate_template';
  status: 'in_progress' | 'success' | 'failed';
  input: OInput;
};
type OperationDetail<OInput = any, OOutput = any> = Operation<OInput> & {
  output: OOutput;
  created_at: string;
  updated_at: string;
};

function getOperation(operation_id: string) {
  return kyClient.get<ApiResponse<OperationDetail>>(`v3/operations/${operation_id}`).json();
}

const useOperationMutation = ({
  onSuccess,
  onError,
}: { onSuccess?: (data: Operation) => void; onError?: (error: Error) => void } = {}) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (operation_id: string) => {
      let operation: ApiResponse<OperationDetail>;

      while (true) {
        operation = await getOperation(operation_id);

        if (operation.data.status === 'success') {
          onSuccess?.(operation.data);
          break;
        }

        if (operation.data.status === 'failed') {
          onError?.(new Error(operation.data.output?.error || 'Operation failed'));
          break;
        }

        await new Promise((resolve) => setTimeout(resolve, 1000));
      }

      if (operation.data.status === 'failed') {
        throw new Error(operation.data.output?.error || 'Operation failed');
      }

      return operation;
    },
    onSuccess: (data, operation_id) => {
      queryClient.setQueryData(['operations', operation_id], data);
    },
  });
};

const useOperationFlow = <TRequest>(
  initialMutation: (request?: TRequest) => Promise<ApiResponse<Operation>>,
  {
    onSuccess,
    onError,
  }: { onSuccess?: (data: Operation) => void; onError?: (error: Error) => void } = {}
) => {
  const operationMutation = useOperationMutation({ onSuccess, onError });

  const initialMutate = useMutation({
    mutationFn: initialMutation,
    onSuccess: (data) => {
      operationMutation.mutate(data.data.id);
    },
  });

  const mutate = useCallback((request?: TRequest) => {
    return initialMutate.mutate(request);
  }, []);

  const mutateAsync = useCallback(async (request: TRequest) => {
    return initialMutate.mutateAsync(request);
  }, []);

  return {
    data: operationMutation.data,
    isLoading: initialMutate.isPending || operationMutation.isPending,
    error: initialMutate.error || operationMutation.error,
    mutate,
    mutateAsync,
  };
};

// Generate Template
type TemplateRequest = {
  contact_data: Contact;
  tone?: string;
  prompt?: string;
  document_gids?: string[];
};
function generateTemplateV3(campaign_id: string, body: TemplateRequest) {
  return kyClient
    .post<ApiResponse<Operation>>(`v3/campaigns/${campaign_id}/templates`, {
      json: body,
    })
    .json();
}

export const useGenerateTemplateQueryV3 = (
  campaign_id: string,
  body: TemplateRequest,
  enabled: boolean = false,
  pos: number
) => {
  return useQuery({
    queryKey: ['campaigns', campaign_id, 'templates', pos, { ...body }],
    queryFn: () => generateTemplateV3(campaign_id, body),
    enabled,
  });
};

// Regenerate Template
function regenerateTemplateV3(campaign_id: string, template_id: string, body: TemplateRequest) {
  return kyClient
    .post<ApiResponse<Operation>>(`v3/campaigns/${campaign_id}/templates/${template_id}`, {
      json: body,
    })
    .json();
}

// Generate Sample Response
type GenerateSampleResponseRequest = {
  intent: EmailIntent;
  contact_data: Contact;
  outreach_template: string;
};

function generateSampleResponseV3(campaign_id: string, body: GenerateSampleResponseRequest) {
  return kyClient
    .post<ApiResponse<Operation>>(`v3/campaigns/${campaign_id}/generate-sample-response`, {
      json: body,
    })
    .json();
}

// Generate Reply
type Conversation = {
  to_email: string;
  content: string;
  status: string;
  from_email: string;
  subject: string;
};
type GenerateReplyRequest = {
  conversation: Conversation[];
  intent: EmailIntent;
};

function generateReplyV3(campaign_id: string, body: GenerateReplyRequest) {
  return kyClient
    .post<ApiResponse<Operation>>(`v3/campaigns/${campaign_id}/generate-reply`, {
      json: body,
    })
    .json();
}

export {
  useOperationFlow,
  getOperation,
  generateTemplateV3,
  regenerateTemplateV3,
  generateSampleResponseV3,
  generateReplyV3,
  type OperationDetail,
  type TemplateRequest,
  type GenerateSampleResponseRequest,
  type GenerateReplyRequest,
};
