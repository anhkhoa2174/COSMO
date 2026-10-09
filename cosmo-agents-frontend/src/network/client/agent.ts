import { useQuery } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { Agent } from '@/models/agent';
import type { EmailInbox, EmailStatistics } from '@/models/email';
import type {
  ApiResponse,
  BaseQuery,
  PaginateResponse,
} from '@/models/response';

interface AgentInList {
  email_statistics?: EmailStatistics;
  entity: Agent;
  inbox_detail?: EmailInbox;
}

interface AgentData {
  name: string;
  cmetadata: {
    email_limitation: {
      daily_limit: number;
      email_per_second: number;
    };
    working_hours: Record<string, unknown>;
  };
  persona: string[];
  daily_limit: number;
  max_daily_limit: number;
  email: string;
  status: string;
  email_provider: string;
  signature: string;
  picture: string;
}

const AgentApi = {
  list: async (payload: any, params?: BaseQuery) => {
    const data = await kyClient.post('v1/agents/search', {
      json: payload,
      searchParams: params,
    });
    return data.json<PaginateResponse<AgentInList>>();
  },
  create: async (payload: AgentData) => {
    const data = await kyClient.post('v1/agents', { json: payload });
    return data.json<PaginateResponse<AgentInList>>();
  },
  update: async (agent_id: string, payload: any) => {
    const data = await kyClient.patch(`v1/agents/${agent_id}`, {
      json: payload,
    });
    return data.json<ApiResponse<AgentInList>>();
  },
  delete: async (agent_id: string) => {
    const data = await kyClient.delete(`v1/agents/${agent_id}`);
    return data.json<PaginateResponse<AgentInList>>();
  },
};

export default AgentApi;

export type GetAgentData = {
  email_statistics: EmailStatistics;
  entity: Agent;
  inbox_detail: EmailInbox;
};
type GetAgentResponse = ApiResponse<GetAgentData>;

export const getAgent = (agentId: string) => {
  return kyClient.get<GetAgentResponse>(`v1/agents/${agentId}`).json();
};

export const useGetAgentQuery = (agentId: string | null) => {
  return useQuery({
    queryKey: ['agents', agentId],
    queryFn: () => getAgent(agentId as string),
    enabled: !!agentId,
  });
};
