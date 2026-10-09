import type { NextRequest } from 'next/server';

import { cheapModel } from '@/lib/ai/provider';
import { convertToCoreMessages, streamText } from 'ai';
import { NextResponse } from 'next/server';
import { requireAuth } from '@/lib/api/require-auth';

export async function POST(req: NextRequest) {
  const unauthorized = requireAuth();
  if (unauthorized) return unauthorized;

  const {
    apiKey: _ignoredClientKey,
    messages,
    model: _ignoredClientModel,
    system,
  } = await req.json();

  // Không bao giờ dùng khoá hay tên model do client gửi lên: bất kỳ ai gọi
  // được endpoint này đều có thể chỉ định model đắt nhất và tính vào hoá đơn
  // của chúng ta.
  const apiKey = process.env.OPENAI_API_KEY;

  if (!apiKey) {
    return NextResponse.json(
      { error: 'Missing OpenAI API key.' },
      { status: 401 }
    );
  }

  try {
    const result = await streamText({
      maxTokens: 2048,
      messages: convertToCoreMessages(messages),
      model: cheapModel(),
      system: system,
    });

    return result.toDataStreamResponse();
  } catch {
    return NextResponse.json(
      { error: 'Failed to process AI request' },
      { status: 500 }
    );
  }
}
