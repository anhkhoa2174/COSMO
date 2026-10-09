import { NextRequest, NextResponse } from 'next/server';
import { getAccessToken } from '@/helpers/server/cookies';

export async function GET(req: NextRequest) {
  const { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  const res = await fetch(
    `${process.env.API_COZE_BASE_URL}/v1/bot/get_online_info?bot_id=${process.env.API_COZE_BOT_ID}`,
    {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${agent_access_token}`,
      },
      cache: 'no-store',
    }
  );

  if (!res.ok) {
    throw new Error('Failed to get bot info');
  }

  const result = await res.json();

  return NextResponse.json(
    {
      success: true,
      data: {
        ...result,
        data: {
          bot_id: result.data?.bot_id,
          knowledge: {
            knowledge_infos: result.data?.knowledge?.knowledge_infos || [],
          },
        },
      },
    },
    { headers: { 'Cache-Control': 'no-store' } }
  );
}
