'use client';

/**
 * Message rendering for the Ask COSMO widget.
 *
 * Assistant text goes through react-markdown (same library the Daily Actions
 * chat uses); every tool call is surfaced as a compact chip so the user can
 * watch the agent act instead of staring at a blank pane.
 */

import type { ComponentProps } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Check, Loader2, Settings2 } from 'lucide-react';
import type { Message } from 'ai';

import { cn } from '@/lib/utils';
import { formatToolName } from '@/components/daily-actions/tool-usage-indicator';

/** Tailwind typography isn't installed in this app, so map the elements. */
const markdownComponents: ComponentProps<typeof Markdown>['components'] = {
  p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
  ul: ({ children }) => (
    <ul className="mb-2 list-disc space-y-1 pl-4 last:mb-0">{children}</ul>
  ),
  ol: ({ children }) => (
    <ol className="mb-2 list-decimal space-y-1 pl-4 last:mb-0">{children}</ol>
  ),
  li: ({ children }) => <li className="leading-relaxed">{children}</li>,
  strong: ({ children }) => (
    <strong className="font-semibold text-foreground">{children}</strong>
  ),
  a: ({ children, href }) => (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="font-medium text-violet-600 underline underline-offset-2 dark:text-violet-400"
    >
      {children}
    </a>
  ),
  code: ({ children }) => (
    <code className="rounded bg-violet-500/10 px-1 py-0.5 font-mono text-[0.75rem] text-violet-700 dark:text-violet-300">
      {children}
    </code>
  ),
  pre: ({ children }) => (
    <pre className="mb-2 overflow-x-auto rounded-lg bg-muted p-2.5 text-[0.75rem] last:mb-0">
      {children}
    </pre>
  ),
  h1: ({ children }) => (
    <h1 className="mb-1.5 text-sm font-semibold text-foreground">{children}</h1>
  ),
  h2: ({ children }) => (
    <h2 className="mb-1.5 text-sm font-semibold text-foreground">{children}</h2>
  ),
  h3: ({ children }) => (
    <h3 className="mb-1.5 text-sm font-semibold text-foreground">{children}</h3>
  ),
  table: ({ children }) => (
    <div className="mb-2 overflow-x-auto last:mb-0">
      <table className="w-full border-collapse text-[0.75rem]">{children}</table>
    </div>
  ),
  th: ({ children }) => (
    <th className="border border-border px-2 py-1 text-left font-semibold">
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td className="border border-border px-2 py-1 align-top">{children}</td>
  ),
};

export function ToolCallChip({
  toolName,
  done,
}: {
  toolName: string;
  done: boolean;
}) {
  return (
    <span
      className={cn(
        'inline-flex max-w-full items-center gap-1.5 rounded-lg border px-2 py-1 text-[0.7rem] font-medium',
        done
          ? 'border-violet-500/20 bg-violet-500/[0.07] text-violet-700 dark:text-violet-300'
          : 'border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300'
      )}
      title={toolName}
    >
      {done ? (
        <Check className="size-3 shrink-0" />
      ) : (
        <Loader2 className="size-3 shrink-0 animate-spin" />
      )}
      <Settings2 className="size-3 shrink-0 opacity-60" />
      <span className="truncate">{formatToolName(toolName)}</span>
    </span>
  );
}

/**
 * `message.parts` keeps text and tool calls in the order the model produced
 * them; older messages (or ones restored without parts) fall back to
 * `content` + `toolInvocations`.
 */
export function CosmoMessage({ message }: { message: Message }) {
  if (message.role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[85%] whitespace-pre-wrap break-words rounded-xl rounded-br-sm bg-gradient-to-br from-violet-500 to-indigo-600 px-3 py-2 text-sm leading-relaxed text-white shadow-sm">
          {message.content}
        </div>
      </div>
    );
  }

  const parts = message.parts?.length
    ? message.parts
    : [
        ...(message.toolInvocations ?? []).map(
          (toolInvocation) =>
            ({ type: 'tool-invocation', toolInvocation }) as const
        ),
        ...(message.content ? [{ type: 'text', text: message.content } as const] : []),
      ];

  const rendered = parts
    .map((part, index) => {
      if (part.type === 'text') {
        if (!part.text.trim()) return null;
        return (
          <div
            key={`text-${index}`}
            className="text-sm leading-relaxed text-muted-foreground"
          >
            <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
              {part.text}
            </Markdown>
          </div>
        );
      }
      if (part.type === 'tool-invocation') {
        return (
          <ToolCallChip
            key={`tool-${part.toolInvocation.toolCallId}-${index}`}
            toolName={part.toolInvocation.toolName}
            done={part.toolInvocation.state === 'result'}
          />
        );
      }
      return null;
    })
    .filter(Boolean);

  if (rendered.length === 0) return null;

  return <div className="flex flex-col items-start gap-1.5">{rendered}</div>;
}
