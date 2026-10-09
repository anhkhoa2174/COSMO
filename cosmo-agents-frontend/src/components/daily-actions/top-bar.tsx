'use client';

import { format } from 'date-fns';
import { Zap } from 'lucide-react';
import type { PipelineSummary } from '@/types/daily-actions';

interface TopBarProps {
  pipelineSummary?: PipelineSummary;
}

export function TopBar({ pipelineSummary }: TopBarProps) {
  const today = format(new Date(), 'EEE, MMM d');

  return (
    <div className="flex h-[52px] shrink-0 items-center justify-between bg-card border-b border-border px-5 sm:px-7">
      {/* Left: agent branding */}
      <div className="flex items-center gap-2.5">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 text-white">
          <Zap className="h-3.5 w-3.5" />
        </div>
        <span className="text-[0.92rem] font-bold text-foreground">BD Agent</span>
        <span className="hidden text-[0.68rem] font-medium text-muted-foreground/80 sm:inline">
          v1.0
        </span>
      </div>

      {/* Right: pipeline status dots + date */}
      <div className="flex items-center gap-4 text-[0.78rem]">
        {pipelineSummary && (
          <div className="hidden items-center gap-3 sm:flex">
            <StatusDot
              color="bg-emerald-500"
              label="Approved"
              count={
                pipelineSummary.contacts_by_stage['approved'] ??
                pipelineSummary.total_active_contacts
              }
            />
            <StatusDot
              color="bg-amber-500"
              label="Pending"
              count={pipelineSummary.contacts_by_stage['pending'] ?? 0}
            />
            <StatusDot
              color="bg-violet-500"
              label="Meetings"
              count={pipelineSummary.meetings_booked_7d}
            />
          </div>
        )}
        <span className="text-muted-foreground/80">{today}</span>
      </div>
    </div>
  );
}

function StatusDot({
  color,
  label,
  count,
}: {
  color: string;
  label: string;
  count: number;
}) {
  return (
    <div className="flex items-center gap-1.5 text-muted-foreground">
      <span className={`h-2 w-2 rounded-full ${color}`} />
      <span>
        {count} {label}
      </span>
    </div>
  );
}
