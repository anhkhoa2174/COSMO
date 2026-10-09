'use client';

/**
 * Ask COSMO on the public site.
 *
 * A visitor can ask about the product before signing in. It is deliberately a
 * smaller thing than the assistant inside the app: no tools, no account, and
 * nothing kept. The thread lives in this component's state and ends when the
 * tab does — there is no session, no local storage, and nothing sent anywhere
 * that could be read back later. The notice in the header says so, because a
 * visitor typing a question into a chat box has every reason to assume the
 * opposite.
 */

import { useChat } from 'ai/react';
import { Send, Sparkles, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';

import { cn } from '@/lib/utils';

const STARTERS = [
  'What does COSMO actually do?',
  'How much does it cost?',
  'Does it send emails without me checking them?',
  'Which tools does it connect to?',
];

export function AskCosmoPublic() {
  const [isOpen, setIsOpen] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const { messages, input, setInput, handleSubmit, isLoading, error } = useChat(
    {
      id: 'ask-cosmo-public',
      api: '/api/ai/ask-cosmo',
    }
  );

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: 'smooth',
    });
  }, [messages, isLoading]);

  if (!isOpen) {
    return (
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        aria-label="Ask COSMO a question"
        className="group fixed bottom-5 right-5 z-40 flex h-14 items-center gap-2.5 rounded-full bg-gradient-to-br from-violet-600 to-indigo-700 px-3.5 text-white shadow-xl shadow-violet-600/30 ring-1 ring-white/20 transition-all hover:shadow-2xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-400"
      >
        <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-white shadow-sm">
          <Sparkles className="size-[1.15rem] text-violet-600" />
        </span>
        <span className="pr-1 text-sm font-semibold">Ask COSMO</span>
      </button>
    );
  }

  return (
    <div
      className={cn(
        'fixed inset-0 z-40 flex flex-col overflow-hidden bg-background',
        'sm:inset-auto sm:bottom-5 sm:right-5 sm:h-[560px] sm:max-h-[calc(100dvh-2.5rem)] sm:w-[400px] sm:rounded-xl sm:border sm:shadow-2xl'
      )}
    >
      <div className="shrink-0 bg-gradient-to-r from-violet-500 to-indigo-600 px-3.5 py-3 text-white">
        <div className="flex items-center gap-2.5">
          <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-white shadow-sm">
            <Sparkles className="size-4 text-violet-600" />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold leading-tight">Ask COSMO</p>
            <p className="text-[0.7rem] leading-tight text-white/80">
              Questions about the product
            </p>
          </div>
          <button
            type="button"
            onClick={() => setIsOpen(false)}
            aria-label="Close"
            className="grid size-7 shrink-0 place-items-center rounded-md text-white/80 transition-colors hover:bg-white/15 hover:text-white"
          >
            <X className="size-4" />
          </button>
        </div>
      </div>

      <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto p-3.5">
        {messages.length === 0 ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Ask anything about COSMO. This assistant answers from the product
              documentation only — it has no access to any account, so it cannot
              look anything up or send anything.
            </p>
            <div className="grid gap-1.5">
              {STARTERS.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => setInput(s)}
                  className="rounded-xl border bg-card px-3 py-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:border-violet-300 hover:bg-violet-500/10 hover:text-violet-700"
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        ) : (
          messages.map((m) => (
            <div
              key={m.id}
              className={cn(
                'max-w-[85%] whitespace-pre-wrap rounded-xl px-3 py-2 text-sm',
                m.role === 'user'
                  ? 'ml-auto bg-violet-600 text-white'
                  : 'bg-muted text-foreground'
              )}
            >
              {m.content}
            </div>
          ))
        )}

        {isLoading && (
          <div className="flex items-center gap-1 px-1">
            <span className="size-1.5 animate-bounce rounded-full bg-violet-500 [animation-delay:-0.3s]" />
            <span className="size-1.5 animate-bounce rounded-full bg-violet-500 [animation-delay:-0.15s]" />
            <span className="size-1.5 animate-bounce rounded-full bg-violet-500" />
          </div>
        )}

        {error && (
          // The route returns real messages — a rate limit reads very
          // differently from an outage, and a visitor deserves to know which.
          <p className="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">
            {error.message ||
              'Something went wrong. Please try again, or book a call.'}
          </p>
        )}
      </div>

      <form
        onSubmit={handleSubmit}
        className="flex shrink-0 items-end gap-2 border-t p-3"
      >
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              if (input.trim() && !isLoading) handleSubmit(e as never);
            }
          }}
          rows={1}
          maxLength={2000}
          placeholder="Ask about features, pricing, integrations…"
          aria-label="Your question"
          className="max-h-28 min-h-9 flex-1 resize-none rounded-lg border bg-background px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-violet-400"
        />
        <button
          type="submit"
          disabled={!input.trim() || isLoading}
          aria-label="Send"
          className="grid size-9 shrink-0 place-items-center rounded-lg bg-violet-600 text-white transition-colors hover:bg-violet-700 disabled:opacity-40"
        >
          <Send className="size-4" />
        </button>
      </form>
    </div>
  );
}
