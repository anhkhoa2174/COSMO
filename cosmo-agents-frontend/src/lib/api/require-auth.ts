import { cookies } from 'next/headers';
import { NextResponse } from 'next/server';

/**
 * Guard for the AI API routes.
 *
 * These endpoints spend money on every call, so an unauthenticated one is a
 * bill anybody on the internet can run up. Returns a 401 response to hand back
 * to the caller, or null when the request carries a session cookie.
 */
export function requireAuth(): NextResponse | null {
  const token = cookies().get('access_token')?.value;
  if (!token) {
    return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
  }
  return null;
}
