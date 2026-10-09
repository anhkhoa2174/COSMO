/**
 * Public Ask COSMO — the landing-page assistant.
 *
 * This is the only AI route in the project without a session cookie, so it is
 * built to a different standard from the authenticated ones. Three properties
 * matter more than features here:
 *
 *   No data. It is given no tools and no API client, so there is no code path
 *   from this endpoint to anyone's contacts, mailbox, or campaigns. That is a
 *   structural guarantee rather than a prompt instruction.
 *
 *   No history. Nothing is persisted. A visitor's questions are not stored
 *   against a session, an address, or a cookie; the thread lives in their
 *   browser tab and ends with it.
 *
 *   Bounded cost. Rate limited per caller, with the conversation and the reply
 *   both capped, because an open LLM endpoint is otherwise a bill anybody can
 *   run up.
 */

import { anthropic } from '@ai-sdk/anthropic';
import { chatModel } from '@/lib/ai/provider';
import { createDataStreamResponse, streamText } from 'ai';
import { NextResponse } from 'next/server';

import { ASK_COSMO_PUBLIC_PROMPT } from '@/lib/ai/cosmo-facts';
import { checkRateLimit, clientKey } from '@/lib/ai/public-rate-limit';

export const runtime = 'nodejs';

/**
 * Caps on what one request may cost.
 *
 * The turn limit matters as much as the character limit: without it a caller
 * can replay a long thread back and pay for the whole history on every message,
 * which is the cheap way to make a capped endpoint expensive.
 */
const MAX_TURNS = 16;
const MAX_CHARS_PER_MESSAGE = 2_000;
const MAX_CHARS_TOTAL = 12_000;
const MAX_OUTPUT_TOKENS = 500;

interface IncomingMessage {
  role?: string;
  content?: unknown;
}

/** Flattens the shapes the AI SDK may send into plain text. */
function textOf(content: unknown): string {
  if (typeof content === 'string') return content;
  if (Array.isArray(content)) {
    return content
      .map((part) =>
        part && typeof part === 'object' && 'text' in part
          ? String((part as { text?: unknown }).text ?? '')
          : ''
      )
      .join(' ');
  }
  return '';
}

export async function POST(req: Request) {
  const limit = checkRateLimit(clientKey(req.headers));
  if (!limit.ok) {
    return NextResponse.json(
      {
        error: `Too many questions in one ${limit.rule}. Please wait a moment — or book a call and talk to a person.`,
      },
      {
        status: 429,
        headers: { 'Retry-After': String(limit.retryAfter ?? 60) },
      }
    );
  }

  let incoming: IncomingMessage[] = [];
  try {
    const body = await req.json();
    incoming = Array.isArray(body?.messages) ? body.messages : [];
  } catch {
    return NextResponse.json(
      { error: 'Invalid request body' },
      { status: 400 }
    );
  }

  // Only the two conversational roles are accepted. A caller could otherwise
  // inject a `system` entry and rewrite the assistant's instructions from the
  // request body, which would defeat every rule in the prompt.
  const messages = incoming
    .filter((m) => m?.role === 'user' || m?.role === 'assistant')
    .map((m) => ({
      role: m.role as 'user' | 'assistant',
      content: textOf(m.content).slice(0, MAX_CHARS_PER_MESSAGE),
    }))
    .filter((m) => m.content.trim() !== '')
    .slice(-MAX_TURNS);

  if (messages.length === 0) {
    return NextResponse.json(
      { error: 'No question was asked' },
      { status: 400 }
    );
  }

  const totalChars = messages.reduce((n, m) => n + m.content.length, 0);
  if (totalChars > MAX_CHARS_TOTAL) {
    return NextResponse.json(
      { error: 'That conversation is too long. Please start a new one.' },
      { status: 413 }
    );
  }

  const useAnthropic =
    !!process.env.ANTHROPIC_API_KEY &&
    process.env.AGENTIC_PROVIDER !== 'openai';

  if (!useAnthropic && !process.env.OPENAI_API_KEY) {
    // Better a clear refusal than a stack trace streamed at a visitor.
    return NextResponse.json(
      {
        error:
          'The assistant is unavailable right now. Please book a call instead.',
      },
      { status: 503 }
    );
  }

  return createDataStreamResponse({
    execute: async (stream) => {
      const result = streamText({
        model: useAnthropic
          ? anthropic('claude-sonnet-4-6')
          : chatModel(),
        system: ASK_COSMO_PUBLIC_PROMPT,
        messages,
        // No tools, deliberately. See the file comment.
        maxTokens: MAX_OUTPUT_TOKENS,
        temperature: 0.3,
      });
      result.mergeIntoDataStream(stream);
    },
    onError: (error) => {
      console.error('[ask-cosmo/public] stream error:', error);
      return 'Something went wrong answering that. Please try again, or book a call.';
    },
  });
}
