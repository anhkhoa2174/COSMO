import { generateObject } from 'ai';
import { cheapModel } from '@/lib/ai/provider';
import { z } from 'zod';
import { NextRequest, NextResponse } from 'next/server';
import { requireAuth } from '@/lib/api/require-auth';

const ParsedMessageSchema = z.object({
  messages: z.array(
    z.object({
      role: z.enum(['me', 'client']).describe('Who sent this message — "me" for the sales rep / BD person, "client" for the prospect / contact'),
      content: z.string().describe('The message content, cleaned up but preserving meaning'),
      channel: z.string().optional().describe('Channel if detectable: LinkedIn, Email, Call, Meeting, Note'),
    })
  ),
});

export async function POST(req: NextRequest) {
  const unauthorized = requireAuth();
  if (unauthorized) return unauthorized;

  try {
    const { rawText, contactName, myName } = await req.json();

    if (!rawText || typeof rawText !== 'string') {
      return NextResponse.json({ error: 'rawText is required' }, { status: 400 });
    }

    const result = await generateObject({
      model: cheapModel(),
      schema: ParsedMessageSchema,
      prompt: `You are parsing a conversation between a sales/BD representative and a prospect/contact.

Contact name: ${contactName || 'Unknown'}
${myName ? `Sales rep name: ${myName}` : ''}

Parse the following raw conversation text into individual messages. For each message, determine:
- "role": "me" if the message is from the sales rep/BD person, "client" if from the prospect/contact
- "content": the actual message content (clean up formatting but preserve meaning)
- "channel": the communication channel if detectable (LinkedIn, Email, Call, Meeting, Note)

Rules:
- Messages from the sales rep / outreach person should be "me"
- Messages from the prospect / lead / contact should be "client"
- If names appear, use them to determine roles. The contact name "${contactName || 'Unknown'}" is the client.
- Preserve the chronological order
- Split multi-message blocks into separate entries
- Remove timestamps/metadata but keep the actual content

Raw conversation:
${rawText}`,
    });

    return NextResponse.json(result.object);
  } catch (error: unknown) {
    console.error('[parse-conversation] Error:', error);
    const message = error instanceof Error ? error.message : 'Failed to parse conversation';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
