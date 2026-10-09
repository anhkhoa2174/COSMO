import { getAccessToken } from '@/helpers/server/cookies';
import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const conversation_id = url.searchParams.get('conversation_id');

  const { agent_access_token, session_name } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  if (!conversation_id) {
    throw new Error('Missing conversation ID.');
  }

  const res = await fetch(
    `${process.env.API_COZE_BASE_URL}/v1/conversation/message/list?conversation_id=${conversation_id}`,
    {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${agent_access_token}`,
      },
    }
  );

  if (!res.ok) {
    throw new Error('Failed to get conversation');
  }

  const result = await res.json();

  return NextResponse.json({
    messages: result.data,
    agent_access_token,
    session_name,
  });
}

const createConversation = (token: string) =>
  fetch(`${process.env.API_COZE_BASE_URL}/v1/conversation/create`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  });

export async function POST(req: NextRequest) {
  let { agent_access_token, session_name } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  let res = await createConversation(agent_access_token);

  // The cookie holding the token outlives the token itself, so a cached value
  // can be rejected long before the browser drops it. Mint a fresh one once
  // and retry rather than leaving the assistant dead until the cookie expires.
  if (res.status === 401) {
    const refreshed = await getAccessToken(true);
    if (refreshed.agent_access_token) {
      agent_access_token = refreshed.agent_access_token;
      session_name = refreshed.session_name;
      res = await createConversation(agent_access_token);
    }
  }

  if (!res.ok) {
    // Coze puts its reason in the body; the Response object alone says nothing.
    const detail = await res.text().catch(() => '');
    console.error(
      `[coze] conversation/create failed: HTTP ${res.status} ${detail}`
    );
    return NextResponse.json(
      { error: 'Failed to create conversation', status: res.status, detail },
      { status: 502 }
    );
  }

  const result = await res.json();

  return NextResponse.json({
    conversation: result.data,
    agent_access_token,
    session_name,
  });
}

export async function DELETE(req: NextRequest) {
  const url = new URL(req.url);
  const conversation_id = url.searchParams.get('conversation_id');

  const { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  if (!conversation_id) {
    throw new Error('Missing conversation ID.');
  }

  const res = await fetch(
    `${process.env.API_COZE_BASE_URL}/v1/conversations/${conversation_id}/clear`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${agent_access_token}`,
      },
    }
  );

  if (!res.ok) {
    throw new Error('Failed to delete conversation');
  }

  return NextResponse.json({
    success: true,
  });
}
