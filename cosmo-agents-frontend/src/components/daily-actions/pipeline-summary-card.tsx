'use client';

import type { PipelineSummary } from '@/types/daily-actions';

interface PipelineSummaryCardProps {
  summary: PipelineSummary;
}

export function PipelineSummaryCard({ summary }: PipelineSummaryCardProps) {
  const stages = Object.entries(summary.contacts_by_stage);
  const totalContacts = summary.total_active_contacts;

  return (
    <div className="rounded-[10px] border border-border bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-indigo-500/[0.25] bg-indigo-500/[0.1] px-4 py-3">
        <span className="text-[0.95rem]">📊</span>
        <span className="text-[0.85rem] font-extrabold text-indigo-600 dark:text-indigo-400">Pipeline Summary</span>
        <span className="ml-1 rounded-md bg-indigo-500/[0.15] px-2 py-0.5 text-[0.68rem] font-extrabold font-mono text-indigo-600 dark:text-indigo-400 border border-indigo-500/[0.25]">
          {totalContacts} contacts
        </span>
      </div>

      {/* Status section */}
      <div className="border-b border-border px-4 py-3.5">
        <h4 className="mb-2 text-[0.65rem] font-bold uppercase tracking-[0.06em] text-muted-foreground/80 font-mono">
          Status
        </h4>
        <div className="flex gap-3">
          <StatBox
            label="Approved"
            value={summary.contacts_by_stage['approved'] ?? 0}
            color="text-emerald-600 dark:text-emerald-400"
          />
          <StatBox
            label="Pending"
            value={summary.contacts_by_stage['pending'] ?? 0}
            color="text-amber-600 dark:text-amber-400"
          />
          <StatBox
            label="Total"
            value={totalContacts}
            color="text-foreground"
          />
        </div>
      </div>

      {/* By Stage section */}
      {stages.length > 0 && (
        <div className="border-b border-border px-4 py-3.5">
          <h4 className="mb-2 text-[0.65rem] font-bold uppercase tracking-[0.06em] text-muted-foreground/80 font-mono">
            By Stage
          </h4>
          <div className="space-y-2">
            {stages.map(([stage, count]) => {
              const pct = totalContacts > 0 ? (count / totalContacts) * 100 : 0;
              return (
                <div key={stage} className="flex items-center gap-3">
                  <span className="w-24 truncate text-[0.78rem] font-medium capitalize text-muted-foreground">
                    {stage.replace(/_/g, ' ')}
                  </span>
                  <div className="h-[6px] flex-1 overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-indigo-600"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <span className="w-8 text-right font-mono text-[0.72rem] text-muted-foreground/80">
                    {count}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Metrics */}
      <div className="px-4 py-3.5">
        <h4 className="mb-2 text-[0.65rem] font-bold uppercase tracking-[0.06em] text-muted-foreground/80 font-mono">
          7-Day Metrics
        </h4>
        <div className="flex gap-3">
          <StatBox
            label="Response Rate"
            value={`${Math.round(summary.response_rate_7d)}%`}
            color="text-indigo-600 dark:text-indigo-400"
          />
          <StatBox
            label="Meetings"
            value={summary.meetings_booked_7d}
            color="text-violet-600 dark:text-violet-400"
          />
          <StatBox
            label="Avg Response"
            value={`${Math.round(summary.avg_response_time_hours)}h`}
            color="text-muted-foreground"
          />
        </div>
      </div>
    </div>
  );
}

function StatBox({
  label,
  value,
  color,
}: {
  label: string;
  value: string | number;
  color: string;
}) {
  return (
    <div className="min-w-[56px] text-center">
      <p className={`text-[1.35rem] font-extrabold font-mono ${color}`}>{value}</p>
      <p className="text-[0.62rem] font-semibold uppercase text-muted-foreground/80 font-mono">
        {label}
      </p>
    </div>
  );
}
