'use client';

import { useState } from 'react';
import { useAtom } from 'jotai';
import { Copy, Check, SkipForward, Clock } from 'lucide-react';
import { toast } from 'sonner';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Input } from '@/components/ui/input';
import type {
  DailyAction,
  ActionType,
  ContactSource,
} from '@/types/daily-actions';
import { useActionMutation } from '@/hooks/use-action-mutation';
import { activePanelAtom } from '@/stores/daily-actions';
import { cn } from '@/lib/utils';
import { OutreachCardContent } from './outreach-card-content';
import { MeetingPrepContent } from './meeting-prep-content';
import { EnrichmentCardContent } from './enrichment-card-content';
import { RespondCardContent } from './respond-card-content';
import { ContactAvatar } from './contact-avatar';

function getStageBadge(
  action: DailyAction
): { label: string; className: string } | null {
  switch (action.type as ActionType) {
    case 'outreach':
      return {
        label: 'Cold',
        className:
          'bg-indigo-500/[0.12] text-indigo-600 dark:text-indigo-400 border border-indigo-500/[0.25]',
      };
    case 'followup': {
      const nextStep = action.outreach_data?.outreach_state?.next_step;
      const n = action.outreach_data?.followup_number ?? 1;
      const isFinal = action.outreach_data?.is_final_followup;
      if (nextStep === 'FOLLOW_UP')
        return {
          label: 'Follow-up Deal',
          className:
            'bg-violet-500/[0.12] text-violet-600 dark:text-violet-400 border border-violet-500/[0.2]',
        };
      if (n >= 2 || isFinal)
        return {
          label: `Follow-up ${n}`,
          className:
            'bg-destructive/[0.1] text-destructive border border-destructive/[0.2]',
        };
      return {
        label: `Follow-up ${n}`,
        className:
          'bg-orange-500/[0.1] text-orange-600 dark:text-orange-400 border border-orange-500/[0.2]',
      };
    }
    case 'respond':
      return {
        label: 'Replied',
        className:
          'bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400 border border-emerald-500/[0.2]',
      };
    case 'meeting_prep':
      return {
        label: 'Meeting Ready',
        className:
          'bg-violet-500/[0.1] text-violet-600 dark:text-violet-400 border border-violet-500/[0.2]',
      };
    case 'enrich':
      return {
        label: 'Pending',
        className:
          'bg-amber-500/[0.1] text-amber-600 dark:text-amber-400 border border-dashed border-amber-500/[0.3]',
      };
    default:
      return null;
  }
}

function formatSource(source: ContactSource): string {
  switch (source) {
    case 'LinkedIn':
      return 'in LinkedIn';
    case 'Apollo':
      return 'Ap Apollo';
    case 'Manual':
      return '✏ Manual';
    case 'HubSpot':
      return 'via HubSpot';
    default:
      return source;
  }
}

interface ActionCardProps {
  action: DailyAction;
  secondaryActions?: DailyAction[];
}

export function ActionCard({ action, secondaryActions }: ActionCardProps) {
  const [, setActivePanel] = useAtom(activePanelAtom);
  const [skipPopoverOpen, setSkipPopoverOpen] = useState(false);
  const [skipReason, setSkipReason] = useState('');
  const [snoozeOpen, setSnoozeOpen] = useState(false);
  const [fadingOut, setFadingOut] = useState(false);
  const mutation = useActionMutation();
  const isTerminal =
    action.status === 'completed' || action.status === 'skipped';
  const isSnoozed = action.status === 'snoozed';
  const isPending = mutation.isPending;

  if (isSnoozed) return null;

  const handleCopyDraft = () => {
    const draft =
      action.outreach_data?.draft_message ||
      action.respond_data?.draft_response;
    if (draft) {
      navigator.clipboard.writeText(draft);
      toast.success('Copied to clipboard');
    }
  };

  // The card collapses the moment an action is taken. If the request then
  // fails the cache rolls back, so the card has to come back too — otherwise
  // it stays at max-h-0/opacity-0 and the action silently disappears.
  const revealOnError = { onError: () => setFadingOut(false) };

  const handleMarkSent = () => {
    setFadingOut(true);
    const draft =
      action.outreach_data?.draft_message ||
      action.respond_data?.draft_response;
    mutation.mutate(
      {
        actionId: action.id,
        request: {
          transition: 'mark_sent',
          content: draft || undefined,
        },
      },
      revealOnError
    );
  };

  const handleSkip = (reason?: string) => {
    setFadingOut(true);
    mutation.mutate(
      {
        actionId: action.id,
        request: { transition: 'skip', skip_reason: reason || undefined },
      },
      revealOnError
    );
    setSkipPopoverOpen(false);
    setSkipReason('');
  };

  const handleSnooze = (key: '1h' | '3h' | '5pm' | 'tmr') => {
    const snoozeUntil = new Date();
    switch (key) {
      case '1h':
        snoozeUntil.setTime(snoozeUntil.getTime() + 60 * 60 * 1000);
        break;
      case '3h':
        snoozeUntil.setTime(snoozeUntil.getTime() + 3 * 60 * 60 * 1000);
        break;
      case '5pm':
        snoozeUntil.setHours(17, 0, 0, 0);
        break;
      case 'tmr':
        snoozeUntil.setDate(snoozeUntil.getDate() + 1);
        snoozeUntil.setHours(9, 0, 0, 0);
        break;
    }
    setFadingOut(true);
    mutation.mutate(
      {
        actionId: action.id,
        request: {
          transition: 'snooze_custom',
          snooze_until: snoozeUntil.toISOString(),
        },
      },
      revealOnError
    );
    setSnoozeOpen(false);
  };

  const handleContactClick = () => {
    setActivePanel({ contactId: action.contact.id });
  };

  // FR-022: Completed/skipped cards show collapsed summary
  if (isTerminal) {
    return (
      <div className="flex items-center justify-between border-b border-border p-5 opacity-50">
        <div className="flex items-center gap-3">
          <ContactAvatar name={action.contact.name} size={24} />
          <span className="text-sm text-muted-foreground">{action.contact.name}</span>
          <span className="text-xs capitalize text-muted-foreground/80">
            — {action.type.replace('_', ' ')}
          </span>
        </div>
        <span
          className={cn(
            'font-mono text-[0.7rem] font-bold uppercase',
            action.status === 'completed' ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground/80'
          )}
        >
          {action.status === 'completed' ? '✓ Completed' : 'Skipped'}
        </span>
      </div>
    );
  }

  const stageBadge = getStageBadge(action);

  return (
    <div
      className={cn(
        'border-b border-border p-5 transition-all duration-300 last:border-b-0 hover:bg-muted',
        fadingOut && 'max-h-0 overflow-hidden !p-0 opacity-0'
      )}
    >
      {/* Contact header */}
      <div className="mb-2.5 flex items-start gap-3">
        <ContactAvatar
          name={action.contact.name}
          size={40}
          onClick={handleContactClick}
        />
        <div className="min-w-0 flex-1">
          <button onClick={handleContactClick} className="text-left">
            <span
              className="border-b-[1.5px] border-dashed border-indigo-500/[0.35] text-[0.95rem] font-bold text-indigo-600 transition-colors hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              {action.contact.name}
            </span>
          </button>
          {action.contact.company && (
            <p className="mt-0.5 text-[0.82rem] text-muted-foreground/80">
              {action.contact.job_title && `${action.contact.job_title} · `}
              {action.contact.company}
            </p>
          )}
          {/* Badge row: stage + source */}
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            {stageBadge && (
              <span
                className={cn(
                  'rounded px-2 py-0.5 font-mono text-[0.65rem] font-bold uppercase tracking-[0.06em]',
                  stageBadge.className
                )}
              >
                {stageBadge.label}
              </span>
            )}
            <span className="rounded border border-border bg-muted px-2 py-0.5 font-mono text-[0.65rem] font-medium text-muted-foreground/80">
              {formatSource(action.contact.source)}
            </span>
            {action.type === 'followup' && action.outreach_data && (
              <span className="rounded bg-muted px-2 py-0.5 font-mono text-[0.65rem] font-semibold text-muted-foreground">
                Day {action.outreach_data.days_since_last_interaction}
              </span>
            )}
            {action.type === 'meeting_prep' && action.meeting_data && (
              <span className="font-mono text-[0.82rem] font-semibold text-violet-600 dark:text-violet-400">
                🗓{' '}
                {new Date(
                  action.meeting_data.meeting_time
                ).toLocaleDateString()}{' '}
                {new Date(action.meeting_data.meeting_time).toLocaleTimeString(
                  [],
                  { hour: '2-digit', minute: '2-digit' }
                )}
              </span>
            )}
          </div>
          {action.type === 'followup' &&
            action.outreach_data &&
            (action.outreach_data.previous_messages_count > 0 ||
              action.outreach_data.last_sent_date) && (
              <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                {action.outreach_data.previous_messages_count > 0 && (
                  <span className="flex cursor-pointer items-center gap-1 rounded border border-indigo-500/[0.25] bg-indigo-500/[0.1] px-2 py-0.5 text-[0.72rem] font-semibold text-indigo-600 dark:text-indigo-400">
                    {action.outreach_data.previous_messages_count} previous
                    messages
                  </span>
                )}
                {action.outreach_data.last_sent_date && (
                  <span className="text-[0.72rem] text-muted-foreground/80">
                    · Last sent:{' '}
                    {new Date(
                      action.outreach_data.last_sent_date
                    ).toLocaleDateString()}
                  </span>
                )}
              </div>
            )}
        </div>
      </div>

      {/* AI reasoning */}
      <p className="mb-2.5 text-[0.88rem] italic leading-relaxed text-muted-foreground">
        {action.reasoning}
      </p>

      {/* Type-specific content */}
      {(action.type === 'outreach' || action.type === 'followup') &&
        action.outreach_data && (
          <OutreachCardContent data={action.outreach_data} />
        )}
      {action.type === 'meeting_prep' && action.meeting_data && (
        <MeetingPrepContent data={action.meeting_data} />
      )}
      {action.type === 'enrich' && action.enrichment_data && (
        <EnrichmentCardContent
          data={action.enrichment_data}
          contactId={action.contact.id}
        />
      )}
      {action.type === 'respond' && action.respond_data && (
        <RespondCardContent data={action.respond_data} />
      )}

      {/* Action buttons */}
      <div className="mt-3.5 flex flex-wrap gap-2">
        {(action.outreach_data?.draft_message ||
          action.respond_data?.draft_response) && (
          <button
            onClick={handleCopyDraft}
            disabled={isPending}
            className="flex items-center gap-1.5 rounded-md border border-border bg-transparent px-3 py-[5px] text-[0.78rem] font-bold text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
          >
            <Copy className="h-3.5 w-3.5" />
            Copy
          </button>
        )}
        <button
          onClick={handleMarkSent}
          disabled={isPending}
          className="flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-[5px] text-[0.78rem] font-bold text-white transition-colors hover:opacity-85 disabled:opacity-50"
        >
          <Check className="h-3.5 w-3.5" />
          Mark Sent
        </button>
        <Popover open={skipPopoverOpen} onOpenChange={setSkipPopoverOpen}>
          <PopoverTrigger asChild>
            <button
              disabled={isPending}
              className="flex items-center gap-1.5 rounded-md border border-border bg-transparent px-3 py-[5px] text-[0.78rem] font-bold text-muted-foreground/80 transition-colors hover:text-muted-foreground disabled:opacity-50"
            >
              <SkipForward className="h-3.5 w-3.5" />
              Skip
            </button>
          </PopoverTrigger>
          <PopoverContent
            className="w-64 space-y-2 border-border bg-card p-3"
            align="start"
          >
            <p className="text-xs text-muted-foreground/80">
              Reason for skipping (optional)
            </p>
            <Input
              value={skipReason}
              onChange={(e) => setSkipReason(e.target.value)}
              placeholder="e.g. Not relevant anymore"
              className="h-8 border-border bg-background text-xs text-foreground placeholder:text-muted-foreground/80"
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSkip(skipReason);
              }}
            />
            <div className="flex justify-end gap-1">
              <button
                className="rounded px-2 py-1 text-xs text-muted-foreground/80 hover:text-muted-foreground"
                onClick={() => handleSkip()}
              >
                Skip without reason
              </button>
              <button
                className="rounded bg-indigo-600 px-3 py-1 text-xs font-semibold text-white disabled:opacity-50"
                onClick={() => handleSkip(skipReason)}
                disabled={!skipReason.trim()}
              >
                Skip
              </button>
            </div>
          </PopoverContent>
        </Popover>
        <div className="relative inline-block">
          <button
            onClick={() => setSnoozeOpen(!snoozeOpen)}
            disabled={isPending}
            className="flex items-center gap-1.5 rounded-md border border-border bg-transparent px-3 py-[5px] text-[0.78rem] font-bold text-muted-foreground/80 transition-colors hover:border-violet-500/[0.25] hover:text-violet-600 dark:hover:text-violet-400 disabled:opacity-50"
          >
            <Clock className="h-3.5 w-3.5" />
            Snooze
          </button>
          {snoozeOpen && (
            <div className="absolute bottom-[calc(100%+6px)] right-0 z-50 min-w-[200px] rounded-[10px] border border-border bg-card py-1.5 shadow-lg animate-in fade-in slide-in-from-bottom-1">
              <div className="px-3.5 pb-1.5 pt-2 font-mono text-[0.65rem] font-bold uppercase tracking-[0.06em] text-muted-foreground/80">
                Snooze until
              </div>
              {[
                {
                  key: '1h' as const,
                  icon: '⏰',
                  label: '1 hour',
                  time: '+1h',
                },
                {
                  key: '3h' as const,
                  icon: '🕐',
                  label: '3 hours',
                  time: '+3h',
                },
                {
                  key: '5pm' as const,
                  icon: '🌅',
                  label: 'End of day',
                  time: '5:00 PM',
                },
                {
                  key: 'tmr' as const,
                  icon: '📅',
                  label: 'Tomorrow',
                  time: '9:00 AM',
                },
              ].map((opt) => (
                <button
                  key={opt.key}
                  onClick={() => handleSnooze(opt.key)}
                  className="flex w-full items-center gap-2.5 px-3.5 py-2.5 text-left text-[0.82rem] text-muted-foreground transition-colors hover:bg-muted"
                >
                  <span className="w-5 text-center text-base">{opt.icon}</span>
                  <span className="flex-1">{opt.label}</span>
                  <span className="font-mono text-[0.72rem] text-muted-foreground/80">
                    {opt.time}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Secondary actions (consolidated from same contact) */}
      {secondaryActions && secondaryActions.length > 0 && (
        <div className="mt-3 space-y-2 border-t border-border pt-3">
          <p className="text-xs font-medium text-muted-foreground/80">
            Also for this contact:
          </p>
          {secondaryActions.map((sa) => (
            <div
              key={sa.id}
              className="rounded-lg border border-dashed border-border p-2 text-xs"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium capitalize text-foreground">
                  {sa.type.replace('_', ' ')}
                </span>
                <span className="font-mono text-[0.65rem] text-muted-foreground/80">
                  P{sa.priority}
                </span>
              </div>
              <p className="mt-1 text-muted-foreground">{sa.reasoning}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
