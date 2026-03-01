import Cookies from 'js-cookie';
import * as jose from 'jose';
import type { AuthTokens } from '@/models/auth';

const ACCESS_TOKEN_KEY = 'access_token';
const REFRESH_TOKEN_KEY = 'refresh_token';
const TOKEN_EXPIRY_KEY = 'token_expiry';

// Secret key for encoding (use environment variable in production)
const SECRET_KEY = new TextEncoder().encode(
  process.env.NEXT_PUBLIC_COOKIE_SECRET ||
    'default-secret-key-change-in-production'
);

async function encodeToken(token: string): Promise<string> {
  try {
    // Use CompactEncrypt for token encryption
    const jwt = await new jose.CompactEncrypt(new TextEncoder().encode(token))
      .setProtectedHeader({ alg: 'dir', enc: 'A256GCM' })
      .encrypt(SECRET_KEY);
    return jwt;
  } catch (error) {
    console.error('Error encoding token:', error);
    return token; // Fallback to unencoded token
  }
}

async function decodeToken(encodedToken: string): Promise<string> {
  try {
    // Check if the token looks like a JWE (has 5 parts separated by dots)
    const parts = encodedToken.split('.');
    if (parts.length !== 5) {
      // Not a JWE format, return as-is (likely unencrypted token)
      return encodedToken;
    }

    const { plaintext } = await jose.compactDecrypt(encodedToken, SECRET_KEY);
    return new TextDecoder().decode(plaintext);
  } catch (error) {
    // If decryption fails, the token might be from an old session or use a different key
    // Log a warning but return the token as-is to maintain backwards compatibility
    console.warn(
      'Token decryption failed, using token as-is. Consider clearing cookies if authentication issues persist.'
    );
    return encodedToken;
  }
}

// Default expiration: 1 hour in seconds
const DEFAULT_EXPIRES_IN = 3600;

export const cookieService = {
  async setTokens(tokens: AuthTokens): Promise<void> {
    try {
      // Encode tokens before storing
      const encodedAccessToken = await encodeToken(tokens.accessToken);
      const encodedRefreshToken = await encodeToken(tokens.refreshToken);

      // Use default expiration if expires_in is 0 or invalid
      const expiresIn = tokens.expiresIn > 0 ? tokens.expiresIn : DEFAULT_EXPIRES_IN;

      // Set cookies with secure attributes
      Cookies.set(ACCESS_TOKEN_KEY, encodedAccessToken, {
        expires: new Date(Date.now() + expiresIn * 1000),
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'strict',
      });

      // Store refresh token with longer expiration (e.g., 7 days)
      Cookies.set(REFRESH_TOKEN_KEY, encodedRefreshToken, {
        expires: 7, // 7 days
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'strict',
      });

      // Store token expiry time
      Cookies.set(
        TOKEN_EXPIRY_KEY,
        String(Date.now() + expiresIn * 1000),
        {
          expires: new Date(Date.now() + expiresIn * 1000),
          secure: process.env.NODE_ENV === 'production',
          sameSite: 'strict',
        }
      );
    } catch (error) {
      console.error('Error setting tokens:', error);
      throw error;
    }
  },

  async getAccessToken(): Promise<string | undefined> {
    const encodedToken = Cookies.get(ACCESS_TOKEN_KEY);
    if (!encodedToken) return undefined;
    return await decodeToken(encodedToken);
  },

  async getRefreshToken(): Promise<string | undefined> {
    const encodedToken = Cookies.get(REFRESH_TOKEN_KEY);
    if (!encodedToken) return undefined;
    return await decodeToken(encodedToken);
  },

  isTokenExpired(): boolean {
    const expiryTime = Cookies.get(TOKEN_EXPIRY_KEY);
    if (!expiryTime) return true;
    return Date.now() > parseInt(expiryTime, 10);
  },

  clearTokens(): void {
    Cookies.remove(ACCESS_TOKEN_KEY);
    Cookies.remove(REFRESH_TOKEN_KEY);
    Cookies.remove(TOKEN_EXPIRY_KEY);
  },
};
