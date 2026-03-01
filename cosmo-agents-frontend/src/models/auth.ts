export enum UserRole {
  ADMIN = 'ADMIN',
  MEMBER = 'MEMBER',
  // Add other roles as needed
}

export interface AuthTokens {
  accessToken: string;
  expiresIn: number;
  refreshToken: string;
}

export interface LoginRequest {
  [key: string]: string;
}

export interface LoginResponse {
  data: {
    access_token: string;
    expires_in: number;
    id_token: string;
    refresh_token: string;
    token_type: 'Bearer';
  };
  status: string;
}

export interface User {
  id: string;
  created_at: string;
  email: string;
  is_deleted: boolean;
  job_title: string | null;
  last_history_id: string | null;
  name: string;
  notifications: any[];
  organizations: {
    id: string;
    company_url: string | null;
    name: string;
    roles: any[];
  }[];
  phone_number: string | null;
  picture: string | null;
  provider: 'google';
  roles: {
    id: string;
    name: 'admin' | 'member';
    organization: any;
  }[];
  staff_emails: any[];
  updated_at: string;
  ui_metadata?: OnboardingPayload;
}
export type OnboardingStatus =
  | 'completed'
  | 'not_started'
  | 'skipped'
  | 'in_progress';
export type OnboardingType = 'default' | 'education' | 'estate' | 'healthcare';
export interface OnboardingPayload {
  onboarding?: boolean;
  status?: string;
  step?: number;
  onboarding_type?: OnboardingType;
  ai_inboxes?: {
    onboarding?: boolean;
    step: number;
    status: OnboardingStatus;
  };
  campaigns?: {
    onboarding?: boolean;
    step: number;
    status: OnboardingStatus;
  };
  contacts?: {
    onboarding?: boolean;
    step: number;
    status: OnboardingStatus;
  };
  libraries?: {
    onboarding?: boolean;
    step: number;
    status: OnboardingStatus;
  };
  settings?: {
    onboarding?: boolean;
    step: number;
    status: OnboardingStatus;
  };
}
