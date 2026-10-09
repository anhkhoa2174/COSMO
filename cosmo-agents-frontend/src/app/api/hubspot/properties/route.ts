import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const accessToken = searchParams.get('accessToken');
  const refreshToken = searchParams.get('refreshToken');

  const headers = new Headers();

  headers.append('Content-Type', 'application/json');
  headers.append('Accept', 'application/json');

  if (accessToken && !request.headers.get('Authorization')) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  const options: RequestInit = {
    method: 'GET',
    headers,
    redirect: 'follow',
  };

  try {
    const response = await fetch(
      'https://api.hubapi.com/crm/v3/properties/contacts',
      options
    );
    const data = await response.json();
    if (!response.ok) {
      if (response.status === 401) {
        const res = await fetch(`${process.env.BASE_URL}/api/hubspot/token`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          body: JSON.stringify({
            grant_type: 'refresh_token',
            refresh_token: refreshToken,
          }),
        });
        const tokenData = await res.json();
        if (res.ok) {
          headers.set('Authorization', `Bearer ${tokenData.access_token}`);
          const retryResponse = await fetch(
            'https://api.hubapi.com/crm/v3/properties/contacts',
            {
              method: 'GET',
              headers,
              redirect: 'follow',
            }
          );
          const retryData = await retryResponse.json();
          return NextResponse.json(retryData, { status: retryResponse.status });
        }
        return NextResponse.json(tokenData, { status: res.status });
      }
      return NextResponse.json(data, { status: response.status });
    }
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { error: error.message || 'Failed to fetch HubSpot properties' },
      { status: 500 }
    );
  }
}
