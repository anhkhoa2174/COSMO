'use client';

import { useState } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import type { MeetingActionData } from '@/types/daily-actions';
import { cn } from '@/lib/utils';

interface MeetingPrepContentProps {
  data: MeetingActionData;
}

/** Labeled section box. */
function BriefingSection({
  label,
  msgCount,
  children,
}: {
  label: string;
  msgCount?: number;
  children: React.ReactNode;
}) {
  return (
    <div className="my-2 rounded-lg border border-border bg-muted/50 px-[18px] py-3.5">
      <div className="mb-2 flex items-center gap-1.5 text-[0.7rem] font-bold uppercase tracking-[0.5px] text-muted-foreground/80 font-mono">
        {label}
        {msgCount != null && msgCount > 0 && (
          <span className="rounded bg-emerald-500/[0.1] border border-emerald-500/[0.2] px-2 py-0.5 text-[0.68rem] text-emerald-600 dark:text-emerald-400 font-semibold">
            {msgCount} messages tracked
          </span>
        )}
      </div>
      {children}
    </div>
  );
}

export function MeetingPrepContent({ data }: MeetingPrepContentProps) {
  const [showDetails, setShowDetails] = useState(false);

  return (
    <div className="space-y-2">
      {/* Meeting info */}
      <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground/80">
        <span className="font-medium text-foreground">{data.meeting_title}</span>
        <span>{data.meeting_duration_minutes}min</span>
        <span className="font-mono text-[0.7rem]">{data.meeting_channel}</span>
        <span
          className={cn(
            'rounded px-2 py-0.5 text-[0.7rem] font-semibold',
            data.hours_until_meeting <= 2
              ? 'bg-destructive/[0.1] border border-destructive/[0.2] text-destructive'
              : 'bg-violet-500/[0.1] border border-violet-500/[0.2] text-violet-600 dark:text-violet-400'
          )}
        >
          In {Math.round(data.hours_until_meeting)}h
        </span>
      </div>

      {/* Expand/collapse toggle */}
      <button
        onClick={() => setShowDetails(!showDetails)}
        className="flex items-center gap-1 text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
      >
        {showDetails ? (
          <ChevronUp className="h-3 w-3" />
        ) : (
          <ChevronDown className="h-3 w-3" />
        )}
        {showDetails ? 'Hide' : 'Show'} meeting prep
      </button>

      <div
        className={cn(
          'space-y-1 overflow-hidden transition-all',
          showDetails ? 'max-h-[2000px]' : 'max-h-0'
        )}
      >
        {/* Prospect profile */}
        <BriefingSection
          label="Prospect Profile"
          msgCount={data.briefing.conversation_summary?.touchpoint_count}
        >
          <p className="text-[0.88rem] leading-relaxed text-muted-foreground">
            {data.briefing.prospect_profile_summary}
          </p>
        </BriefingSection>

        {/* Conversation summary */}
        {data.briefing.conversation_summary && (
          <BriefingSection label="Conversation Summary">
            <div className="space-y-1 text-[0.88rem] text-muted-foreground">
              <p>
                {data.briefing.conversation_summary.duration_days} days,{' '}
                {data.briefing.conversation_summary.touchpoint_count}{' '}
                touchpoints. Tone:{' '}
                {data.briefing.conversation_summary.tone_assessment}
              </p>
              {data.briefing.conversation_summary.key_topics.length > 0 && (
                <p>
                  Topics:{' '}
                  {data.briefing.conversation_summary.key_topics.join(', ')}
                </p>
              )}
            </div>
          </BriefingSection>
        )}

        {/* Discovery questions */}
        {data.briefing.discovery_questions.length > 0 && (
          <BriefingSection label="Discovery Questions">
            <ol className="list-decimal pl-[18px] text-[0.88rem] text-muted-foreground">
              {data.briefing.discovery_questions.map((q, i) => (
                <li key={i} className="mb-1">
                  {q}
                </li>
              ))}
            </ol>
          </BriefingSection>
        )}

        {/* Risk flags */}
        {data.briefing.risk_flags.length > 0 && (
          <BriefingSection label="Risk Flags">
            <ul className="list-disc pl-[18px] text-[0.88rem] text-destructive">
              {data.briefing.risk_flags.map((flag, i) => (
                <li key={i} className="mb-1">
                  {flag}
                </li>
              ))}
            </ul>
          </BriefingSection>
        )}

        {/* Suggested agenda */}
        <BriefingSection label="Suggested Agenda">
          <ol className="list-decimal pl-[18px] text-[0.88rem] text-muted-foreground">
            {data.briefing.suggested_agenda.map((item, i) => (
              <li key={i} className="mb-1">
                {item.topic} ({item.duration_minutes}min)
              </li>
            ))}
          </ol>
        </BriefingSection>

        {/* Next-steps box — amber variant */}
        {data.briefing.recommended_next_steps.length > 0 && (
          <div className="my-2 rounded-lg border border-amber-500/[0.2] bg-amber-500/[0.1] px-[18px] py-3.5">
            <div className="mb-2 text-[0.7rem] font-bold uppercase tracking-[0.5px] text-amber-600 dark:text-amber-400 font-mono">
              Next Steps
            </div>
            {data.briefing.recommended_next_steps.map((step, i) => (
              <p key={i} className="mb-1 text-[0.88rem] text-muted-foreground">
                {step}
              </p>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
