/**
 * Agentic Daily Action Generation Route
 *
 * POST /api/ai/daily-actions/generate
 *
 * Assembles full contact context via SDK tools, calls Claude for strategic
 * recommendations, then POSTs results to backend to create DailyAction records.
 *
 * 006-agentic-daily-actions
 */

import { generateText } from 'ai';
import { anthropic } from '@ai-sdk/anthropic';
import { chatModel } from '@/lib/ai/provider';
import { cookies } from 'next/headers';
import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';
import * as jose from 'jose';
import { CosmoApiClient, ToolExecutor } from 'cosmo-agents-sdk';
import {
  assembleAgentContext,
  buildAgentPrompt,
  type AgentGenerationResult,
} from '@/lib/ai/agentic-generation';

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
  const startTime = Date.now();

  try {
    // Auth
    const cookieStore = cookies();
    const rawToken = cookieStore.get('access_token')?.value;
    if (!rawToken) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }

    const accessToken = await decodeToken(rawToken);
    const { language = 'vi' } = await req.json().catch(() => ({}));

    const baseUrl = process.env.SERVER_BASE_URL!;

    // Create SDK client + executor
    const client = new CosmoApiClient({ baseUrl, apiKey: accessToken });
    const executor = new ToolExecutor(client, {});

    // Step 1: Assemble context via SDK tools
    console.log('[agentic-gen] Assembling context...');
    const context = await assembleAgentContext(executor, baseUrl, accessToken);

    if (context.contacts.length === 0) {
      console.log('[agentic-gen] No contacts to evaluate, falling back');
      return NextResponse.json({ status: 'fallback', reason: 'no_contacts' });
    }

    console.log(
      `[agentic-gen] Context assembled: ${context.contacts.length} contacts, ` +
        `${Object.keys(context.knowledgeResults).length} KB industries, ` +
        `metrics: ${context.outcomeMetrics ? 'yes' : 'no'}`
    );

    // Step 2: Build prompt
    const { system, user } = buildAgentPrompt(context);

    // Step 3: Call the LLM. Ưu tiên Anthropic khi có key, ngược lại dùng
    // OpenAI (OPENAI_MODEL của hệ thống) — key Anthropic hiện không có credit.
    const useAnthropic =
      !!process.env.ANTHROPIC_API_KEY && process.env.AGENTIC_PROVIDER !== 'openai';
    const model = useAnthropic
      ? anthropic('claude-sonnet-4-20250514')
      : chatModel();
    console.log(`[agentic-gen] Calling ${useAnthropic ? 'Claude' : 'OpenAI'}...`);
    const { text, usage } = await generateText({
      model,
      system,
      prompt: user,
      // gpt-5.4 (reasoning) từ chối max_tokens -> chỉ giới hạn khi dùng Claude
      ...(useAnthropic ? { maxTokens: 4096 } : {}),
    });

    console.log(
      `[agentic-gen] Claude responded: ${usage?.totalTokens ?? '?'} tokens`
    );

    // Step 4: Parse structured response
    let result: AgentGenerationResult;
    try {
      // Extract JSON from response (Claude may wrap in ```json blocks)
      let jsonText = text.trim();
      if (jsonText.startsWith('```')) {
        jsonText = jsonText.replace(/^```(?:json)?\n?/, '').replace(/\n?```$/, '');
      }
      result = JSON.parse(jsonText);
    } catch (parseErr) {
      console.error('[agentic-gen] Failed to parse Claude response:', parseErr);
      console.error('[agentic-gen] Raw response:', text.slice(0, 500));
      return NextResponse.json({ status: 'fallback', reason: 'parse_error' });
    }

    if (!result.recommendations?.length) {
      console.log('[agentic-gen] No recommendations in response, falling back');
      return NextResponse.json({
        status: 'fallback',
        reason: 'empty_recommendations',
      });
    }

    // Step 5: POST to backend to create DailyAction records
    console.log(
      `[agentic-gen] Creating ${result.recommendations.length} actions...`
    );
    const createRes = await fetch(
      `${baseUrl}/v1/daily-actions/create-from-agent`,
      {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${accessToken}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          language,
          recommendations: result.recommendations,
          strategic_plan: result.strategic_plan ?? '',
          focus_areas: result.focus_areas ?? [],
          outcome_insights: result.outcome_insights ?? [],
        }),
      }
    );

    if (!createRes.ok) {
      const errorBody = await createRes.text().catch(() => 'unknown');
      console.error(
        `[agentic-gen] Backend create failed: ${createRes.status} ${errorBody}`
      );
      return NextResponse.json({ status: 'fallback', reason: 'backend_error' });
    }

    const createData = await createRes.json();
    const duration = Date.now() - startTime;

    console.log(
      `[agentic-gen] Done in ${duration}ms. ` +
        `Actions: ${result.recommendations.length}, ` +
        `Tokens: ${usage?.totalTokens ?? '?'}, ` +
        `Cost est: $${((usage?.totalTokens ?? 0) * 0.000003).toFixed(4)}`
    );

    return NextResponse.json({
      status: 'ready',
      generation_id: createData.data?.generation_id,
      action_count: result.recommendations.length,
      duration_ms: duration,
      tokens_used: usage?.totalTokens,
    });
  } catch (err) {
    console.error('[agentic-gen] Unexpected error:', err);
    return NextResponse.json({
      status: 'fallback',
      reason: 'unexpected_error',
      message: err instanceof Error ? err.message : 'Unknown error',
    });
  }
}
