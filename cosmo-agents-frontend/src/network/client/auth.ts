import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

type CallbackData = {
  token_type: 'Bearer';
  access_token: string;
  refresh_token: string;
  id_token: string;
  expires_in: number;
};
type CallbackResponse = ApiResponse<CallbackData>;

type AuthURLData = {
  url: string;
};
type AuthURLResponse = ApiResponse<AuthURLData>;

const AuthApi = {
  getGoogleAuthURL: (redirectUri: string) => {
    const url = 'v2/auth/adapter/google';
    const searchParams = { redirect_uri: redirectUri };
    return kyClient.get(url, { searchParams }).json<AuthURLResponse>();
  },
  getGmailAuthURL: (redirectUri: string) => {
    const url = 'v2/google/gmail';
    const searchParams = { redirect_uri: redirectUri };
    return kyClient.get(url, { searchParams }).json<AuthURLResponse>();
  },
  processGmailAuthCallback: (searchParams: any) => {
    const url = 'v2/google/gmail/oauth2callback';
    return kyClient.get(url, { searchParams }).json();
  },
  processInviteCallback: async (searchParams: any) => {
    const url = 'v2/auth/members/invite-callback';
    return kyClient.get(url, { searchParams }).json<ApiResponse>();
  },
  processMemberAuthCallback: async (searchParams: any) => {
    const url = 'v2/auth/members/oauth2callback';
    return kyClient.get(url, { searchParams }).json<CallbackResponse>();
  },
};

export default AuthApi;
