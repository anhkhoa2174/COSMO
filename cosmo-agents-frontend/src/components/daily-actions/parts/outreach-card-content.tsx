'use client';

import type { OutreachActionData } from '@/types/daily-actions';

interface OutreachCardContentProps {
  data: OutreachActionData;
}

export function OutreachCardContent({ data }: OutreachCardContentProps) {
  return (
    <div className="space-y-2.5">
      {/* AI Strategic Reasoning */}
      {data.strategic_reasoning && (
        <div className="rounded-lg border border-indigo-400/20 bg-indigo-600/5 px-4 py-3">
          <div className="mb-1.5 flex items-center gap-1.5">
            <span className="text-[0.65rem]">🧠</span>
            <span className="text-[0.7rem] font-bold uppercase tracking-[0.5px] text-indigo-600 dark:text-indigo-400 font-mono">
              AI Strategy
            </span>
            {data.confidence_level && (
              <span
                className={`ml-auto text-[0.6rem] font-mono px-1.5 py-0.5 rounded ${
                  data.confidence_level === 'high'
                    ? 'bg-green-500/10 text-green-400'
                    : data.confidence_level === 'medium'
                      ? 'bg-yellow-500/10 text-yellow-400'
                      : 'bg-gray-500/10 text-gray-400'
                }`}
              >
                {data.confidence_level}
              </span>
            )}
          </div>
          <p className="text-[0.82rem] leading-relaxed text-foreground/80">
            {data.strategic_reasoning}
          </p>
          {data.messaging_strategy && (
            <div className="mt-2 inline-flex items-center gap-1.5 rounded-md border border-violet-400/20 bg-violet-500/5 px-2 py-0.5">
              <span className="text-[0.65rem] text-violet-600 dark:text-violet-400 font-mono">
                Strategy: {data.messaging_strategy}
              </span>
            </div>
          )}
          {data.outcome_pattern_cited && (
            <p className="mt-1.5 text-[0.7rem] text-muted-foreground/80 italic">
              📊 {data.outcome_pattern_cited}
            </p>
          )}
          {data.recommended_channel && (
            <p className="mt-1 text-[0.7rem] text-muted-foreground/80">
              📡 Recommended: {data.recommended_channel}
            </p>
          )}
        </div>
      )}

      {/* Referenced knowledge */}
      {data.referenced_knowledge && data.referenced_knowledge.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {data.referenced_knowledge.map((item, i) => (
            <span
              key={i}
              className="inline-flex items-center gap-1 rounded-md border border-border bg-card px-2 py-0.5 text-[0.65rem] text-muted-foreground"
            >
              📄 {item}
            </span>
          ))}
        </div>
      )}

      {/* Company context — labeled box */}
      {data.company_context && !data.strategic_reasoning && (
        <div className="rounded-lg border border-border bg-muted/50 px-4 py-3">
          <div className="mb-1 text-[0.7rem] font-bold uppercase tracking-[0.5px] text-muted-foreground/80 font-mono">
            Context
          </div>
          <p className="text-[0.85rem] leading-relaxed text-muted-foreground">
            {data.company_context}
          </p>
        </div>
      )}

      {/* Draft message — bordered box */}
      <div className="rounded-[10px] border border-border bg-muted/50 px-5 py-[18px]">
        <p className="whitespace-pre-wrap text-[0.9rem] leading-[1.7] text-foreground">
          {data.draft_message}
        </p>
      </div>
    </div>
  );
}
