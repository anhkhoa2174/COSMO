import type { ApiResponse, PaginateResponse } from './response';

export type Member = {
  id: string;
  name: string;
  email: string;
  picture: string | null;
  phone_number: string | null;
  job_title: string | null;
};

// search member
export type MemberSearchRequest = {
  filter: any;
};

export type MemberDeletePayload = {
  member_ids: string[];
};

export type MemberSearchResponseData = {
  entity: Member;
  role: {
    id: string;
    organization_id: string;
    status: string;
    job_title: string;
    name: string;
  };
};
export type MemberSearchResponse = PaginateResponse<MemberSearchResponseData>;

// create member
export type MemberCreateRequest = {
  redirect_uri: string;
  email: string;
  job_title: string;
  role: 'admin' | 'member';
};

type MemberCreateResponseData = Member & {
  last_history_id: null;
  created_at: string;
  provider: 'google' | null;
  staff_emails: string[];
  is_deleted: boolean;
  updated_at: string;
  organizations: any[];
  roles: any[];
  notifications: any[];
};
export type MemberCreateResponse = ApiResponse<MemberCreateResponseData>;
