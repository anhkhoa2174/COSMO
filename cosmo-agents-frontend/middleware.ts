import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import * as jose from 'jose';

// Secret key for decoding (must match the one in cookieService)
const SECRET_KEY = new TextEncoder().encode(
  process.env.NEXT_PUBLIC_COOKIE_SECRET ||
    'default-secret-key-change-in-production'
);

const API_BASE_URL =
  process.env.SERVER_BASE_URL || process.env.NEXT_PUBLIC_SERVER_BASE_URL;

async function decodeToken(encodedToken: string): Promise<string | null> {
  try {
    const { plaintext } = await jose.compactDecrypt(encodedToken, SECRET_KEY);
    return new TextDecoder().decode(plaintext);
  } catch (error) {
    return null;
  }
}

async function hasValidToken(request: NextRequest): Promise<boolean> {
  const accessToken = request.cookies.get('access_token')?.value;
  const tokenExpiry = request.cookies.get('token_expiry')?.value;

  if (!accessToken || !tokenExpiry) {
    return false;
  }

  // Check if token is expired
  if (Date.now() > parseInt(tokenExpiry, 10)) {
    return false;
  }

  // Try to decode the token to verify it's valid
  const decoded = await decodeToken(accessToken);
  return decoded !== null;
}

async function getUserData(request: NextRequest): Promise<any | null> {
  const accessToken = request.cookies.get('access_token')?.value;

  if (!accessToken) {
    return null;
  }

  try {
    const decodedToken = await decodeToken(accessToken);
    if (!decodedToken) {
      return null;
    }

    const response = await fetch(`${API_BASE_URL}/v1/users/me`, {
      headers: {
        Authorization: `Bearer ${decodedToken}`,
      },
    });

    if (!response.ok) {
      return null;
    }

    const data = await response.json();
    return data.data;
  } catch (error) {
    console.error('Error fetching user data:', error);
    return null;
  }
}

function needsOnboarding(user: any): boolean {
  if (!user) return false;
  return user.roles?.length === 0 || user.organizations?.length === 0;
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Protected routes that require organization membership
  const protectedRoutes = [
    '/contact-lists',
    '/contacts',
    '/custom-fields',
    '/hubspot-integration',
    '/agents',
    '/company-information',
    '/sales-reps',
    '/settings',
    '/team-members',
    '/ai-inboxes',
    '/campaigns',
    '/libraries',
  ];

  const isProtectedRoute = protectedRoutes.some((route) =>
    pathname.startsWith(route)
  );

  // Check if user needs onboarding for protected routes
  if (isProtectedRoute) {
    // Check for valid authentication
    const isAuthenticated = await hasValidToken(request);

    if (!isAuthenticated) {
      // Redirect to login with the current path as next parameter
      const url = request.nextUrl.clone();
      url.pathname = '/auth/login';
      url.searchParams.set('next', pathname);
      return NextResponse.redirect(url);
    } else {
      const user = await getUserData(request);

      if (needsOnboarding(user)) {
        // Redirect to onboarding if user hasn't completed it
        const url = request.nextUrl.clone();
        url.pathname = '/onboarding';
        return NextResponse.redirect(url);
      }
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
};
