'use client';

import { useState } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import type { MemoryReference } from '@/types/daily-actions';
import { cn } from '@/lib/utils';

interface AgentReasoningBlockProps {
  text: string;
  memoryRefs?: MemoryReference[];
}

export function AgentReasoningBlock({
  text,
  memoryRefs,
}: AgentReasoningBlockProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="relative my-3.5 rounded-lg border border-border bg-card p-[11px_14px] text-[0.88rem] italic leading-[1.65] text-muted-foreground">
      <span className="absolute -top-2 left-3 bg-card px-1.5 text-[0.56rem] font-bold not-italic uppercase tracking-[0.08em] text-muted-foreground/80 font-mono border border-border rounded-sm">
        reasoning
      </span>
      <span>
        {text.split(/(\*\*.*?\*\*|\[\[.*?\]\])/).map((part, i) => {
          const boldMatch = part.match(/^\*\*(.*?)\*\*$/);
          if (boldMatch) return <strong key={i} className="text-foreground not-italic">{boldMatch[1]}</strong>;
          const refMatch = part.match(/^\[\[(.*?)\]\]$/);
          if (refMatch) return <span key={i} className="rounded bg-indigo-500/[0.1] px-1.5 py-px not-italic font-semibold text-indigo-600 dark:text-indigo-400 border border-indigo-500/[0.25]">{refMatch[1]}</span>;
          return part;
        })}
      </span>

      {memoryRefs && memoryRefs.length > 0 && (
        <div className="mt-3 not-italic">
          <button
            onClick={() => setExpanded(!expanded)}
            className="flex items-center gap-1 text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
          >
            {expanded ? (
              <ChevronUp className="h-3 w-3" />
            ) : (
              <ChevronDown className="h-3 w-3" />
            )}
            {memoryRefs.length} memory reference
            {memoryRefs.length !== 1 ? 's' : ''}
          </button>

          <div
            className={cn(
              'mt-2 space-y-2 overflow-hidden transition-all',
              expanded ? 'max-h-96' : 'max-h-0'
            )}
          >
            {memoryRefs.map((ref, i) => (
              <div
                key={i}
                className="rounded-lg border border-border bg-muted/50 p-2.5 text-xs"
              >
                <div className="flex items-center justify-between">
                  <p className="font-medium text-foreground">
                    {ref.contact_name}
                  </p>
                  <span className="font-mono text-[10px] text-muted-foreground/80">
                    {new Date(ref.event_timestamp).toLocaleString([], {
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </span>
                </div>
                <p className="text-muted-foreground">{ref.event_summary}</p>
                <p className="mt-1 italic text-indigo-600 dark:text-indigo-400">{ref.relevance}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
