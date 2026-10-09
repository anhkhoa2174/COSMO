'use client';

import type { DailySummaryData } from '@/types/daily-actions';

interface SummaryBlockProps {
  summary: DailySummaryData;
}

export function SummaryBlock({ summary }: SummaryBlockProps) {
  const { progress, breakdown, outcomes, carry_over } = summary;

  return (
    <div className="mt-3.5 rounded-[10px] border border-border bg-card p-4 text-[0.85rem] leading-[1.65] text-muted-foreground">
      <strong className="text-foreground">Daily Summary:</strong>{' '}
      {summary.agent_summary} You&apos;ve completed{' '}
      <strong className="text-foreground">
        {progress.completed} out of {progress.total}
      </strong>{' '}
      actions today.{' '}
      <strong className="text-foreground">
        {breakdown.outreach_sent} outreach messages
      </strong>{' '}
      sent,{' '}
      <strong className="text-foreground">
        {breakdown.replies_handled} replies
      </strong>{' '}
      handled,{' '}
      <strong className="text-foreground">
        {breakdown.meetings_prepped} meetings
      </strong>{' '}
      prepped.
      {progress.snoozed > 0 && (
        <>
          {' '}
          <strong className="text-foreground">{progress.snoozed}</strong> actions
          snoozed.
        </>
      )}
      {carry_over.length > 0 && (
        <>
          {' '}
          <strong className="text-foreground">{carry_over.length}</strong>{' '}
          actions carried forward.
        </>
      )}
      {outcomes.responses_received_today > 0 && (
        <>
          {' '}
          Pipeline activity:{' '}
          <strong className="text-foreground">
            +{outcomes.responses_received_today} new responses
          </strong>{' '}
          received today.
        </>
      )}
    </div>
  );
}
