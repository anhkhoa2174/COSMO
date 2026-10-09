'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import {
  Check,
  ChevronDown,
  ChevronRight,
  Compass,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  describeNextAction,
  nextActionColors,
  nextActionLabels,
} from '@/lib/next-step';
import { cn } from '@/lib/utils';
import OutreachApi, {
  type NextAction,
  type NextStepDecision,
} from '@/network/client/outreach';

const selectedByText: Record<NextStepDecision['selected_by'], string> = {
  rule: 'Only one action was allowed',
  model: 'Chosen by AI among the allowed actions',
  fallback: 'AI unavailable — default cadence used',
};

const triggerText: Record<NextStepDecision['trigger'], string> = {
  reply: 'after a reply',
  timer: 'when a wait ended',
  cadence: 'when the cadence moved on',
  manual: 'on request',
};

const approvalText: Record<NextStepDecision['approval'], string> = {
  automatic: 'Runs automatically',
  approve: 'You approve the email before it is sent',
  task: 'Becomes a task for you',
  confirm: 'Needs your confirmation',
  decide: 'Handed to you to decide',
};

function ActionBadge({ action }: { action: NextAction }) {
  return (
    <Badge className={cn('text-white', nextActionColors[action])}>
      {nextActionLabels[action] ?? action}
    </Badge>
  );
}

/** Why the engine took a decision: what it was allowed, and what it was not. */
function DecisionWhy({ d }: { d: NextStepDecision }) {
  const rules = (d.rules_fired ?? []).filter((r) => r.rule > 0);
  return (
    <div className="space-y-3 rounded-md border bg-muted/30 p-3 text-xs">
      <div>
        <p className="mb-1 font-medium">Allowed actions</p>
        <div className="flex flex-wrap gap-1">
          {(d.eligible ?? []).map((a) => (
            <Badge
              key={a}
              variant={a === d.action ? 'default' : 'outline'}
              className="text-[10px]"
            >
              {nextActionLabels[a] ?? a}
            </Badge>
          ))}
        </div>
      </div>
      {rules.length > 0 && (
        <div>
          <p className="mb-1 font-medium">Rules that applied</p>
          <ul className="space-y-1">
            {rules.map((r) => (
              <li key={r.rule}>
                <span className="font-mono text-muted-foreground">
                  Rule {r.rule}
                </span>{' '}
                {r.effect}
                {r.removed && r.removed.length > 0 && (
                  <span className="text-muted-foreground">
                    {' '}
                    — removed{' '}
                    {r.removed.map((a) => nextActionLabels[a] ?? a).join(', ')}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      {d.situation?.details && (
        <div>
          <p className="mb-1 font-medium">Read from the reply</p>
          <p className="text-muted-foreground">
            {Object.entries(d.situation.details)
              .filter(([, v]) => v !== '' && v !== false && v != null)
              .map(([k, v]) => `${k.replace('_', ' ')}: ${v}`)
              .join(' · ') || 'nothing specific'}
          </p>
        </div>
      )}
      <p className="text-muted-foreground">
        {selectedByText[d.selected_by]}
        {d.fallback_cause ? ` (${d.fallback_cause})` : ''}
        {d.model ? ` · ${d.model}` : ''}
      </p>
    </div>
  );
}

export function NextStepPanel({ contactId }: { contactId: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['next-step', contactId],
    queryFn: () => OutreachApi.getNextStep(contactId),
    enabled: !!contactId,
  });

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['next-step', contactId] });
    queryClient.invalidateQueries({ queryKey: ['contacts-for-outreach'] });
  };

  const decide = useMutation({
    mutationFn: () => OutreachApi.decideNextStep(contactId),
    onSuccess: (res) => {
      refresh();
      const d = res.data;
      toast.success(
        d ? describeNextAction(d.action, d.args) : 'Next step decided'
      );
    },
    onError: (e: any) =>
      toast.error('Could not decide the next step', {
        description: e?.message,
      }),
  });

  const review = useMutation({
    mutationFn: (v: { id: string; approve: boolean }) =>
      OutreachApi.reviewNextStep(v.id, v.approve),
    onSuccess: refresh,
  });

  const view = data?.data;
  const current = view?.current;
  const latest = view?.decisions?.[0];

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center justify-between text-base">
          <span className="flex items-center gap-2">
            <Compass className="h-4 w-4 text-violet-600" />
            Next step
          </span>
          <Button
            size="sm"
            variant="outline"
            className="gap-1.5"
            disabled={decide.isPending}
            onClick={() => decide.mutate()}
          >
            {decide.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" />
            )}
            Decide now
          </Button>
        </CardTitle>
        <CardDescription>
          {view && !view.enabled
            ? 'The engine is off for your organisation (Settings → Outreach Timing). Decide now still works for this contact.'
            : 'Chosen from the situation after rules remove what must not be done.'}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        ) : !current ? (
          <p className="text-sm text-muted-foreground">
            No decision yet. Press “Decide now”, or wait for the next reply.
          </p>
        ) : (
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <ActionBadge action={current.action} />
              <span className="text-sm font-medium">
                {describeNextAction(current.action, current.args, current.due_at)}
              </span>
            </div>
            {current.reason && (
              <p className="text-sm text-muted-foreground">
                {current.reason}
              </p>
            )}
            {latest && latest.id === current.decision_id && (
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span>{approvalText[latest.approval]}</span>
                {latest.status === 'pending_review' &&
                  (latest.approval === 'confirm' ||
                    latest.approval === 'decide' ||
                    latest.approval === 'task') && (
                    <span className="flex gap-1">
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-6 gap-1 px-2 text-xs"
                        onClick={() =>
                          review.mutate({ id: latest.id, approve: true })
                        }
                      >
                        <Check className="h-3 w-3" /> Confirm
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 gap-1 px-2 text-xs"
                        onClick={() =>
                          review.mutate({ id: latest.id, approve: false })
                        }
                      >
                        <X className="h-3 w-3" /> Reject
                      </Button>
                    </span>
                  )}
                {latest.status === 'approved' && (
                  <Badge variant="outline">Confirmed</Badge>
                )}
                {latest.status === 'rejected' && (
                  <Badge variant="outline">Rejected</Badge>
                )}
              </div>
            )}
          </div>
        )}

        {view?.decisions && view.decisions.length > 0 && (
          <div className="space-y-1 border-t pt-3">
            <p className="text-xs font-medium text-muted-foreground">
              Decision history
            </p>
            {view.decisions.slice(0, 6).map((d) => (
              <div key={d.id} className="text-sm">
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded px-1 py-1 text-left hover:bg-accent"
                  onClick={() => setOpen(open === d.id ? null : d.id)}
                >
                  {open === d.id ? (
                    <ChevronDown className="h-3.5 w-3.5 shrink-0" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 shrink-0" />
                  )}
                  <ActionBadge action={d.action} />
                  <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                    {triggerText[d.trigger]} ·{' '}
                    {formatDistanceToNow(new Date(d.created_at), {
                      addSuffix: true,
                    })}{' '}
                    · {d.reason}
                  </span>
                </button>
                {open === d.id && (
                  <div className="ml-6 mt-1">
                    <DecisionWhy d={d} />
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
