'use client';

import type { RespondActionData } from '@/types/daily-actions';

interface RespondCardContentProps {
  data: RespondActionData;
}

export function RespondCardContent({ data }: RespondCardContentProps) {
  const replyAge = Math.round(
    (Date.now() - new Date(data.reply_timestamp).getTime()) / (1000 * 60 * 60)
  );

  return (
    <div className="space-y-2.5">
      {/* Intent assessment badge row */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="font-mono rounded bg-emerald-500/[0.1] border border-emerald-500/[0.2] px-2.5 py-0.5 text-[0.7rem] font-bold uppercase text-emerald-600 dark:text-emerald-400">
          {data.intent_assessment.replace('_', ' ')}
        </span>
        <span className="text-[0.78rem] text-muted-foreground/80">
          via {data.reply_channel} · {replyAge}h ago
        </span>
      </div>

      {/* Prospect reply — green border quote box */}
      <div className="rounded-lg border border-emerald-500/[0.2] bg-emerald-500/[0.1] px-[18px] py-3.5">
        <div className="mb-1.5 flex items-center justify-between">
          <span className="text-[0.7rem] font-bold uppercase tracking-[0.5px] text-emerald-600 dark:text-emerald-400 font-mono">
            Prospect Reply
          </span>
          <span className="text-[0.7rem] font-medium text-muted-foreground/80 font-mono">
            {new Date(data.reply_timestamp).toLocaleString([], {
              month: 'short',
              day: 'numeric',
              hour: '2-digit',
              minute: '2-digit',
            })}
          </span>
        </div>
        <p className="text-[0.9rem] italic leading-relaxed text-muted-foreground">
          &ldquo;{data.reply_preview}&rdquo;
        </p>
      </div>

      {/* Recommended action — inline */}
      <p className="text-[0.88rem] leading-relaxed text-muted-foreground">
        <strong className="text-foreground">Recommended: </strong>
        {data.recommended_action}
      </p>

      {/* Intent reasoning */}
      <p className="text-[0.85rem] text-muted-foreground/80">{data.intent_reasoning}</p>

      {/* Draft response — bordered box */}
      {data.draft_response && (
        <div className="rounded-[10px] border border-border bg-muted/50 px-5 py-[18px]">
          <p className="whitespace-pre-wrap text-[0.9rem] leading-[1.7] text-foreground">
            {data.draft_response}
          </p>
        </div>
      )}
    </div>
  );
}
