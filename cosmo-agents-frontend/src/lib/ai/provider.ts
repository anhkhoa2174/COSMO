import { createGoogleGenerativeAI } from '@ai-sdk/google';
import { createOpenAI } from '@ai-sdk/openai';

/**
 * The model provider, resolved from configuration.
 *
 * One key serves every AI feature, so when it runs out of credit they all fail
 * together and there is no way to keep a demonstration running. Gemini exposes
 * an OpenAI-compatible endpoint, so `OPENAI_BASE_URL` plus a model name is
 * normally enough to substitute it — the same lever `pkg/ai/provider.go` gives
 * the back end.
 *
 * "Normally" is doing real work in that sentence. The compatibility layer is
 * faithful for plain completions and diverges on streamed tool calls: Gemini
 * omits the `index` field inside each `tool_calls` delta, which the OpenAI
 * provider's schema requires, and the stream dies mid-answer with a validation
 * error rather than a useful message. The agentic assistant calls tools on
 * nearly every turn, so for it the shim is not usable at all.
 *
 * Hence two paths. Anything reached through the OpenAI-compatible endpoint uses
 * the OpenAI provider, except when the endpoint is Gemini's, where the native
 * provider speaks the same protocol Gemini actually streams.
 *
 * It remains a stand-in. The models are not the same, so the output is not the
 * same, and any figure measured through this path describes a different system
 * from the one the report evaluates.
 */

/** True when the client is pointed somewhere other than OpenAI. */
export function usingCompatibleHost(): boolean {
  return !!process.env.OPENAI_BASE_URL;
}

/** True when that somewhere is Gemini. */
function usingGemini(): boolean {
  return /generativelanguage\.googleapis\.com/i.test(
    process.env.OPENAI_BASE_URL || ''
  );
}

function modelFor(name: string) {
  if (usingGemini()) {
    // The native provider takes the bare host, not the /v1beta/openai/ shim.
    return createGoogleGenerativeAI({
      apiKey:
        process.env.GOOGLE_GENERATIVE_AI_API_KEY || process.env.OPENAI_API_KEY,
    })(name);
  }
  return createOpenAI({
    apiKey: process.env.OPENAI_API_KEY,
    // Passing undefined leaves the SDK's own default in place.
    baseURL: process.env.OPENAI_BASE_URL || undefined,
  })(name);
}

/**
 * The model for ordinary generation.
 *
 * The OpenAI-specific fallback applies only when talking to OpenAI. A
 * compatible host will not recognise "gpt-5.4", and falling back to it there
 * produces a 404 that reads like an outage rather than a missing setting.
 */
export function chatModel() {
  const name = process.env.OPENAI_MODEL;
  if (!name) {
    if (usingCompatibleHost()) {
      throw new Error(
        'OPENAI_BASE_URL is set but OPENAI_MODEL is not. A compatible host ' +
          'does not accept OpenAI model names, so there is no safe default.'
      );
    }
    return modelFor('gpt-5.4');
  }
  return modelFor(name);
}

/**
 * The model for cheap, high-volume calls: the scope gate, the inline command
 * bar, the copilot.
 *
 * When no cheap model is named, this falls back to the main one rather than to
 * a hard-coded OpenAI name — the same 404 argument as above. Paying more for
 * these calls is worse than free, but far better than having them fail.
 */
export function cheapModel() {
  const name = process.env.OPENAI_MODEL_CHEAP;
  if (name) return modelFor(name);
  if (usingCompatibleHost() || process.env.OPENAI_MODEL) return chatModel();
  return modelFor('gpt-4o-mini');
}
