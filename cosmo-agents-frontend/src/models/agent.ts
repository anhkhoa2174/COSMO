export type AgentStatus =
  | 'active'
  | 'inactive'
  | 'insufficient scopes'
  | 'needs sync setup'
  | 'invalid Google grant';

/** The states that are fixed by re-running the Gmail OAuth flow. */
export const AGENT_RECONNECT_STATUSES: AgentStatus[] = [
  'insufficient scopes',
  'needs sync setup',
  'invalid Google grant',
];

export type Agent = {
  id: string;
  name: string;
  user_id: string;
  organization_id: string;
  cmetadata: {
    email_limitation: {
      daily_limit: number;
      email_per_second: number;
    };
    working_hours: Record<string, any>;
  };
  is_deleted: boolean;
  persona: string[] | null;
  daily_limit: number;
  max_daily_limit: number;
  email: string;
  /**
   * Mirrors AgentStatus on the backend. The three trailing values all mean
   * "the Google connection is broken" — they are not the same as `inactive`,
   * which is a deliberate pause.
   */
  status: AgentStatus | null;
  email_provider: string;
  signature: string | null;
  picture: string;
};
