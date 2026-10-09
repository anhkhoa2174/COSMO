import { getAccessToken } from '@/helpers/server/cookies';
import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';

export async function POST(req: NextRequest) {
  const { conversation_id, messages, bot_id } = await req.json();

  let { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  if (!conversation_id) {
    throw new Error('Missing conversation ID.');
  }

  if (!messages) {
    throw new Error('Missing messages.');
  }

  const startChat = (token: string) =>
    fetch(
      `${process.env.API_COZE_BASE_URL}/v3/chat?conversation_id=${conversation_id}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          bot_id: bot_id || process.env.API_COZE_BOT_ID,
          user_id: process.env.API_COZE_USER_ID,
          stream: true,
          auto_save_history: true,
          additional_messages: messages,
        }),
      }
    );

  let res = await startChat(agent_access_token);

  // Same cached-token problem as conversation/create: retry once with a
  // freshly minted token before giving up.
  if (res.status === 401) {
    const refreshed = await getAccessToken(true);
    if (refreshed.agent_access_token) {
      res = await startChat(refreshed.agent_access_token);
    }
  }

  if (!res.ok) {
    const detail = await res.text().catch(() => '');
    console.error(`[coze] v3/chat failed: HTTP ${res.status} ${detail}`);
    return NextResponse.json(
      { error: 'Failed to create chat', status: res.status, detail },
      { status: 502 }
    );
  }

  const reader = res.body?.getReader();
  const decoder = new TextDecoder();

  return new NextResponse(
    new ReadableStream({
      async start(controller) {
        if (!reader) return;
        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            const chunk = decoder.decode(value);
            controller.enqueue(chunk);
          }
        } finally {
          reader.releaseLock();
          controller.close();
        }
      },
    }),
    {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      },
    }
  );
}
