import { NextRequest, NextResponse } from 'next/server';

export async function POST(request: NextRequest) {
  const res = await request.json();
  const { grant_type, code, refresh_token } = res;

  const headers = new Headers();

  headers.append('Content-Type', 'application/x-www-form-urlencoded');
  headers.append('Accept', 'application/json');

  const body = new URLSearchParams();

  body.append('grant_type', grant_type);
  body.append('client_id', process.env.HUBSPOT_CLIENT_ID!);
  body.append('client_secret', process.env.HUBSPOT_CLIENT_SECRET!);

  if (grant_type === 'authorization_code') {
    body.append('redirect_uri', process.env.HUBSPOT_REDIRECT_URI!);
    body.append('code', code);
  }

  if (grant_type === 'refresh_token') {
    body.append('refresh_token', refresh_token);
  }

  const options: RequestInit = {
    method: 'POST',
    headers,
    body,
    redirect: 'follow',
  };

  try {
    const response = await fetch(
      'https://api.hubapi.com/oauth/v1/token',
      options
    );
    const data = await response.json();
    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { error: error.message || 'Failed to fetch HubSpot token' },
      { status: 500 }
    );
  }
}
