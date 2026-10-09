import { getAccessToken } from '@/helpers/server/cookies';
import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';
export const revalidate = 0;

export async function POST(req: NextRequest) {
  const { dataset_ids } = await req.json();

  const { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  const res = await fetch(`${process.env.API_COZE_BASE_URL}/v1/bot/update`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${agent_access_token}`,
    },
    body: JSON.stringify({
      bot_id: process.env.API_COZE_BOT_ID,
      knowledge: {
        dataset_ids,
      },
    }),
    cache: 'no-store',
  });

  if (!res.ok) {
    throw new Error('Failed to update bot', { cause: res });
  }
  const result = await res.json();

  return NextResponse.json(
    {
      success: true,
      data: result,
    },
    { headers: { 'Cache-Control': 'no-store' } }
  );
}
