/**
 * Daily Actions Chat API Route — SDK Tool Integration
 *
 * Upgraded from backend proxy to inline streamText using @ai-sdk/anthropic
 * with all 60+ cosmo-agents-sdk COSMO_TOOLS mounted.
 *
 * FR-027: FE chat has access to all cosmo-agents-sdk tools.
 * FR-028: Routes through the same tool registry as the SDK.
 * FR-029: All tools invocable via natural language.
 * FR-030: Responses match SDK output format and data fidelity.
 */

import { createDataStreamResponse, streamText } from 'ai';
import { anthropic } from '@ai-sdk/anthropic';
import { chatModel } from '@/lib/ai/provider';
import { cookies } from 'next/headers';
import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';
import * as jose from 'jose';
import { CosmoApiClient, ToolExecutor } from 'cosmo-agents-sdk';
import { buildBDAgentTools } from '@/lib/ai/bd-agent-tools';
import { cannedReply, checkScope } from '@/lib/ai/scope-gate';

const BD_AGENT_SYSTEM_PROMPT = `
You are COSMO BD Agent — an autonomous AI employee that manages the BD (Business Development) pipeline.

Your responsibilities:
- Daily pipeline analysis and prioritized action suggestions
- Contact lookups, enrichment, and personalized outreach draft generation
- Follow-up cadence tracking and meeting preparation briefings
- Analytics queries and pipeline health reporting
- Strategy advice and competitive positioning

When helping users:
- Always explain your reasoning with specific data points (contact names, timestamps, interaction counts)
- Reference conversation history when relevant (e.g., "Based on Phạm Quốc Bảo's reply at 14:32 on Feb 7...")
- Provide strategic context, not just task lists
- Support both English and Vietnamese languages — match the language the user writes in
- For pipeline summary/overview/statistics queries, ALWAYS use get_pipeline_summary tool to get structured data
- For meeting pipeline or contacts with meetings, use get_meeting_pipeline tool
- For analytics/performance reports, use daily_analytics tool
- For deep analysis on a contact, use full_analysis tool
- For bulk enrichment, use batch_enrichment tool
- For counting contacts matching a keyword, use count_contacts_by_keyword tool
- For contact searches, use search_contacts and suggest_outreach tools to get real data
- For analytics, use daily_analytics, count_contacts_created, and related time-based tools
- Never fabricate contact information — always use tools to fetch real data

When asked about contacts, outreach, meetings, or analytics, use the appropriate tools to provide accurate, data-driven answers.
`.trim();

const SECRET_KEY = new TextEncoder().encode(
  process.env.NEXT_PUBLIC_COOKIE_SECRET ||
    'default-secret-key-change-in-production'
);

async function decodeToken(encodedToken: string): Promise<string> {
  try {
    const parts = encodedToken.split('.');
    if (parts.length !== 5) return encodedToken;
    const { plaintext } = await jose.compactDecrypt(encodedToken, SECRET_KEY);
    return new TextDecoder().decode(plaintext);
  } catch {
    return encodedToken;
  }
}

export async function POST(req: NextRequest) {
  try {
    const cookieStore = cookies();
    const rawToken = cookieStore.get('access_token')?.value;
    if (!rawToken) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }

    const accessToken = await decodeToken(rawToken);
    const { messages } = await req.json();

    if (!messages || !Array.isArray(messages)) {
      return NextResponse.json({ error: 'Missing messages' }, { status: 400 });
    }

    // Gate before assembling tools. Every turn below ships 56 tool schemas
    // (~35k characters); a greeting or an off-topic question should never
    // cost that.
    const lastUser = [...messages]
      .reverse()
      .find((m: { role: string }) => m.role === 'user');
    const lastText =
      typeof lastUser?.content === 'string' ? lastUser.content : '';
    const verdict = await checkScope(lastText);
    if (verdict !== 'allow') {
      console.log('[daily-actions/chat] short-circuited as', verdict);
      const reply = cannedReply(verdict, lastText);
      return createDataStreamResponse({
        execute: (stream) => {
          stream.writeData({ scope: verdict });
          // Emit in chunks so the widget types it out like a real answer
          // instead of snapping the whole block into place at once.
          for (const chunk of reply.match(/\s*\S+|\s+/g) ?? [reply]) {
            stream.write(`0:${JSON.stringify(chunk)}\n`);
          }
        },
      });
    }

    const client = new CosmoApiClient({
      baseUrl: process.env.SERVER_BASE_URL!,
      apiKey: accessToken,
    });

    const executor = new ToolExecutor(client, {
      apolloApiKey: process.env.APOLLO_API_KEY,
    });

    const tools = buildBDAgentTools(executor);

    console.log(
      '[daily-actions/chat] Starting streamText with',
      Object.keys(tools).length,
      'tools'
    );
    console.log('[daily-actions/chat] Messages count:', messages.length);
    console.log(
      '[daily-actions/chat] ANTHROPIC_API_KEY set:',
      !!process.env.ANTHROPIC_API_KEY
    );

    const useAnthropic =
      !!process.env.ANTHROPIC_API_KEY &&
      process.env.AGENTIC_PROVIDER !== 'openai';
    const result = streamText({
      model: useAnthropic
        ? anthropic('claude-sonnet-4-6')
        : chatModel(),
      system: BD_AGENT_SYSTEM_PROMPT,
      messages,
      tools,
      maxSteps: 10,
      onStepFinish: ({ toolCalls, text }) => {
        console.log(
          '[daily-actions/chat] Step finished. Text length:',
          text?.length ?? 0,
          'Tool calls:',
          toolCalls?.length ?? 0
        );
      },
      onFinish: ({ text, usage }) => {
        console.log(
          '[daily-actions/chat] Stream finished. Text length:',
          text?.length ?? 0,
          'Usage:',
          JSON.stringify(usage)
        );
      },
    });

    return result.toDataStreamResponse({
      // Mặc định ai-sdk giấu lỗi ("An error occurred.") — trả message thật
      // để client/log còn debug được.
      getErrorMessage: (error) =>
        error instanceof Error
          ? `${error.name}: ${error.message}`
          : String(error),
    });
  } catch (err) {
    console.error('[daily-actions/chat] Error:', err);
    return NextResponse.json(
      { error: err instanceof Error ? err.message : 'Internal server error' },
      { status: 500 }
    );
  }
}
