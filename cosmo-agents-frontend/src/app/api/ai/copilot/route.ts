import type { NextRequest } from 'next/server';

import { cheapModel } from '@/lib/ai/provider';
import { generateText } from 'ai';
import { NextResponse } from 'next/server';
import { requireAuth } from '@/lib/api/require-auth';

export async function POST(req: NextRequest) {
  const unauthorized = requireAuth();
  if (unauthorized) return unauthorized;

  const {
    apiKey: _ignoredClientKey,
    model: _ignoredClientModel,
    prompt,
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
    const result = await generateText({
      abortSignal: req.signal,
      maxTokens: 50,
      model: cheapModel(),
      prompt: prompt,
      system,
      temperature: 0.7,
    });

    return NextResponse.json(result);
  } catch (error: any) {
    if (error.name === 'AbortError') {
      return NextResponse.json(null, { status: 408 });
    }

    return NextResponse.json(
      { error: 'Failed to process AI request' },
      { status: 500 }
    );
  }
}
