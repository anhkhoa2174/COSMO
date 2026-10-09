import { getAccessToken } from '@/helpers/server/cookies';
import { NextRequest, NextResponse } from 'next/server';

export async function POST(req: NextRequest) {
  const { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  const res = await fetch(`${process.env.API_COZE_BASE_URL}/v1/bot/publish`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${agent_access_token}`,
    },
    body: JSON.stringify({
      bot_id: process.env.API_COZE_BOT_ID,
      connector_ids: ['1024'],
    }),
    cache: 'no-store',
  });

  if (!res.ok) {
    throw new Error('Failed to publish bot', { cause: res });
  }

  return NextResponse.json({
    success: true,
  }, { headers: { 'Cache-Control': 'no-store' } });
}
