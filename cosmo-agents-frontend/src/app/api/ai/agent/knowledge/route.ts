import { getAccessToken } from '@/helpers/server/cookies';
import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const page = url.searchParams.get('page');
  const size = url.searchParams.get('size');

  const { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  const res = await fetch(
    `${process.env.API_COZE_BASE_URL}/open_api/knowledge/document/list`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${agent_access_token}`,
        'Agw-Js-Conv': 'str',
      },
      body: JSON.stringify({
        dataset_id: process.env.API_COZE_DATASET_ID,
        page: Number(page),
        size: Number(size),
      }),
    }
  );

  if (!res.ok) {
    const errorData = await res.json();
    throw new Error('Failed to get knowledge', { cause: errorData });
  }

  const result = await res.json();

  return NextResponse.json({
    success: true,
    data: result.document_infos,
  });
}

export async function POST(req: NextRequest) {
  const { document_bases, chunk_strategy } = await req.json();

  const { agent_access_token } = await getAccessToken();
  if (!agent_access_token) {
    throw new Error('Missing access token.');
  }

  if (!document_bases || !chunk_strategy) {
    throw new Error('Missing required fields.');
  }

  try {
    const res = await fetch(
      `${process.env.API_COZE_BASE_URL}/open_api/knowledge/document/create`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${agent_access_token}`,
          'Agw-Js-Conv': 'str',
        },
        body: JSON.stringify({
          dataset_id: '7491992561326768146',
          document_bases,
          chunk_strategy,
        }),
      }
    );

    if (!res.ok) {
      const errorData = await res.json();
      throw new Error('Failed to create document', { cause: errorData });
    }

    const result = await res.json();

    return NextResponse.json({
      success: true,
      data: result.document_infos,
    });
  } catch (error: any) {
    throw new Error('Failed to create document', { cause: error });
  }
}
