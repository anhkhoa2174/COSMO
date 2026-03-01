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
  status: 'active' | 'inactive' | null;
  email_provider: string;
  signature: string | null;
  picture: string;
};
