import { kyClient } from '@/lib/ky';

/** The nightly rollup for one member. Absent until the job has reached them. */
export interface MemberMetrics {
  emails_sent: number;
  replies: number;
  meetings: number;
  reply_rate: number;
  computed_at: string;
}

export interface ProductivityMember {
  user_id: string;
  name: string;
  email: string;
  picture?: string;
  role: string;
  is_you: boolean;
  actions_completed: number;
  actions_skipped: number;
  meetings_booked: number;
  active_days: number;
  metrics?: MemberMetrics;
}

export interface ProductivityReport {
  /**
   * Decided by the server from the caller's role, never from the request.
   * `team` means the caller is an admin and every member is listed; `self`
   * means the list holds exactly the caller.
   */
  scope: 'team' | 'self';
  period: '7d' | '30d';
  days: number;
  members: ProductivityMember[];
  metrics_pending: boolean;
}

const TeamProductivityApi = {
  get: async (period: '7d' | '30d', orgId = 'me') => {
    const res = await kyClient.get<{ data: ProductivityReport }>(
      `v2/organizations/${orgId}/team-productivity`,
      { searchParams: { period } }
    );
    return (await res.json()).data;
  },
};

export default TeamProductivityApi;
