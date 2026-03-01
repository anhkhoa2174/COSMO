import { kyClient } from '@/lib/ky';
import type { LoginRequest, LoginResponse, User } from '@/models/auth';
import type { ApiResponse } from '@/models/response';
import { cookieService } from './cookieService';

class AuthService {
  private static instance: AuthService;
  private isAuthenticated: boolean = false;
  private refreshPromise: Promise<void> | null = null;

  private constructor() {
    // Check if user is authenticated by checking token existence and expiry
    this.checkAuthStatus();
  }

  private async checkAuthStatus() {
    const hasToken = !!(await cookieService.getAccessToken());
    this.isAuthenticated = hasToken && !cookieService.isTokenExpired();
  }

  public static getInstance(): AuthService {
    if (!AuthService.instance) {
      AuthService.instance = new AuthService();
    }
    return AuthService.instance;
  }

  public async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await kyClient
      .get('v2/auth/oauth2callback', {
        searchParams: credentials,
      })
      .json<LoginResponse>();

    // Store tokens in cookies
    await cookieService.setTokens({
      accessToken: response.data.access_token,
      refreshToken: response.data.refresh_token,
      expiresIn: response.data.expires_in,
    });

    this.isAuthenticated = true;
    return response;
  }

  public async getMe(): Promise<ApiResponse<User>> {
    const accessToken = await cookieService.getAccessToken();
    if (!accessToken) {
      throw new Error('No access token available');
    }

    return kyClient
      .get('v1/users/me', {
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      })
      .json<ApiResponse<User>>();
  }

  public async refreshToken(): Promise<void> {
    // If a refresh is already in progress, wait for it
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this._performRefresh();

    try {
      await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async _performRefresh(): Promise<void> {
    const refreshToken = await cookieService.getRefreshToken();
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await kyClient
        .post('v2/auth/refresh', {
          searchParams: { refresh_token: refreshToken },
        })
        .json<LoginResponse>();

      // Store new tokens
      await cookieService.setTokens({
        accessToken: response.data.access_token,
        refreshToken: response.data.refresh_token,
        expiresIn: response.data.expires_in,
      });

      this.isAuthenticated = true;
    } catch (error) {
      // If refresh fails, clear tokens and logout
      cookieService.clearTokens();
      this.isAuthenticated = false;
      throw error;
    }
  }

  public async logout(): Promise<void> {
    cookieService.clearTokens();
    this.isAuthenticated = false;
    // Redirect to login page
    if (typeof window !== 'undefined') {
      window.location.href = '/auth/login';
    }
  }

  public async update(arg: Record<string, any>): Promise<void> {
    await cookieService.setTokens({
      accessToken: arg.access_token,
      refreshToken: arg.refresh_token,
      expiresIn: arg.expires_in,
    });
    this.isAuthenticated = true;
  }

  public async getAuthStatus(): Promise<boolean> {
    await this.checkAuthStatus();
    return this.isAuthenticated && !cookieService.isTokenExpired();
  }

  public async getAccessToken(): Promise<string | undefined> {
    return cookieService.getAccessToken();
  }
}

export const authService = AuthService.getInstance();
