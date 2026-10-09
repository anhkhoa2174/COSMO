'use client';

import { useQuery } from '@tanstack/react-query';
import { format, subDays } from 'date-fns';
import {
  AlertCircle,
  BarChart3,
  Calendar,
  CalendarRange,
  CheckSquare,
  Inbox,
  Mail,
  SkipForward,
  TrendingUp,
  UserX,
  Users,
  UsersRound,
} from 'lucide-react';
import React from 'react';

import { PageHero } from '@/components/ui/page-hero';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import {
  AttentionCard,
  MetricCard,
} from '@/app/(main)/dashboard/_components/metric-card';
import { TeamProductivity } from '@/app/(main)/dashboard/_components/team-productivity';
import TeamProductivityApi, {
  type ProductivityMember,
} from '@/network/client/team-productivity';

const PERIODS = [
  { id: '7d' as const, label: 'Last 7 days', days: 7 },
  { id: '30d' as const, label: 'Last 30 days', days: 30 },
];

/** Skipping more than this share of actions is worth a conversation. */
const HIGH_SKIP_RATE = 0.5;

/**
 * The admin dashboard: how the team is doing, not how the admin is doing.
 *
 * Every figure is summed from the team-productivity report, which the server
 * scopes by role — an admin receives a row per member, anyone else receives
 * only their own. The page is only rendered for admins, but even if it were
 * reached by a member it could show nothing beyond their own row.
 */
export function TeamDashboard() {
  const [period, setPeriod] = React.useState<'7d' | '30d'>('30d');
  const [now, setNow] = React.useState<Date | null>(null);
  React.useEffect(() => setNow(new Date()), []);

  const { data: report, isLoading } = useQuery({
    queryKey: ['dashboard-productivity', period],
    queryFn: () => TeamProductivityApi.get(period),
  });

  const members = report?.members ?? [];
  const selected = PERIODS.find((p) => p.id === period)!;
  const rangeLabel = now
    ? `${format(subDays(now, selected.days - 1), 'MMM d, yyyy')} – ${format(now, 'MMM d, yyyy')}`
    : null;

  // Reply figures come from the nightly rollup, which may not have reached
  // every member yet. Only members it has reached are summed, and the cards
  // say so rather than presenting a partial total as the whole team's.
  const withMetrics = members.filter((m) => m.metrics);
  const sent = withMetrics.reduce((n, m) => n + m.metrics!.emails_sent, 0);
  const replies = withMetrics.reduce((n, m) => n + m.metrics!.replies, 0);
  // Each member's rate is "threads answered ÷ threads written to". Dividing
  // the reply total by the send total instead counted every follow-up as a
  // chance to reply and every extra answer as a reply, and read 200% on a
  // thread that answered twice. The team rate is the members' rates weighted
  // by how much each one sent.
  const replyRate =
    sent > 0
      ? (withMetrics.reduce(
          (n, m) => n + m.metrics!.reply_rate * m.metrics!.emails_sent,
          0,
        ) /
          sent) *
        100
      : null;
  const metricsPartial =
    members.length > 0 && withMetrics.length < members.length;

  const completed = members.reduce((n, m) => n + m.actions_completed, 0);
  const skipped = members.reduce((n, m) => n + m.actions_skipped, 0);
  const meetings = members.reduce((n, m) => n + m.meetings_booked, 0);
  // Someone who ran a campaign has worked, even with no daily action or
  // meeting logged: emails sent count as activity, or an admin who only sent
  // campaign mail was flagged "inactive" on their own dashboard.
  const isActive = (m: (typeof members)[number]) =>
    m.actions_completed > 0 ||
    m.meetings_booked > 0 ||
    (m.metrics?.emails_sent ?? 0) > 0;
  const active = members.filter(isActive);
  const inactive = members.filter((m) => !isActive(m));
  const highSkip = members.filter((m) => {
    const handled = m.actions_completed + m.actions_skipped;
    return handled > 0 && m.actions_skipped / handled > HIGH_SKIP_RATE;
  });
  const noReplies = withMetrics.filter(
    (m) => m.metrics!.emails_sent > 0 && m.metrics!.replies === 0
  );

  const needsAttention = inactive.length + highSkip.length + noReplies.length;

  return (
    <>
      <PageHero
        icon={UsersRound}
        eyebrow="Team overview"
        accent="violet"
        title="Team dashboard"
        description={
          rangeLabel
            ? `${members.length} member${members.length === 1 ? '' : 's'} · ${rangeLabel}`
            : null
        }
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex items-center gap-1 rounded-full border bg-background/80 p-1">
              {PERIODS.map((p) => (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => setPeriod(p.id)}
                  className={cn(
                    'rounded-full px-3.5 py-1.5 text-[0.8rem] font-medium transition-colors',
                    period === p.id
                      ? 'bg-violet-600 text-white'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
            {rangeLabel && (
              <span className="inline-flex h-9 items-center gap-2 rounded-full border bg-background/80 px-3.5 text-[0.8rem] font-medium">
                <CalendarRange className="size-4 text-violet-600" />
                {rangeLabel}
              </span>
            )}
            <span className="inline-flex h-9 items-center gap-2 rounded-full border bg-background/80 px-3.5 text-[0.8rem] font-medium">
              <BarChart3 className="size-4 text-violet-600" />
              {replyRate != null
                ? `${replyRate.toFixed(1)}% team reply rate`
                : '— team reply rate'}
            </span>
          </div>
        }
      />

      <section className="space-y-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">
            Who needs your attention
          </h2>
          <p className="text-[0.9rem] text-muted-foreground">
            {needsAttention === 0
              ? 'Everyone on the team is active'
              : `${needsAttention} flag${needsAttention === 1 ? '' : 's'} across the team`}
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <AttentionCard
            icon={UserX}
            count={inactive.length}
            label="Inactive members"
            description={
              inactive.length
                ? `No actions, emails or meetings in ${selected.days} days: ${names(inactive)}`
                : `Everyone did work in the last ${selected.days} days`
            }
            href="/team-members"
            tone="rose"
            loading={isLoading}
          />
          <AttentionCard
            icon={SkipForward}
            count={highSkip.length}
            label="Skipping most actions"
            description={
              highSkip.length
                ? `Skipped over half of their suggested actions: ${names(highSkip)}`
                : 'Nobody is skipping most of their suggested actions'
            }
            href="/team-members"
            tone="amber"
            loading={isLoading}
          />
          <AttentionCard
            icon={Inbox}
            count={noReplies.length}
            label="Sending without replies"
            description={
              noReplies.length
                ? `Emails sent but no replies yet: ${names(noReplies)}`
                : withMetrics.some((m) => m.metrics!.emails_sent > 0)
                  ? 'Everyone who sent email has received a reply'
                  : `Nobody sent email in the last ${selected.days} days`
            }
            href="/team-members"
            tone="violet"
            loading={isLoading}
          />
        </div>
      </section>

      <section className="space-y-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">
            Team overview
          </h2>
          <p className="text-[0.9rem] text-muted-foreground">
            {rangeLabel ? `Totals for ${rangeLabel}` : 'Loading period…'}
          </p>
          {(report?.metrics_pending || metricsPartial) && (
            <p className="mt-2 inline-flex items-center gap-2 rounded-md bg-amber-50 px-3 py-1.5 text-[0.8rem] text-amber-800">
              <AlertCircle className="size-3.5 shrink-0" />
              Email and reply totals cover {withMetrics.length} of{' '}
              {members.length} members — the nightly job has not reached the
              rest. Action and meeting counts are live.
            </p>
          )}
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <MetricCard
            icon={Mail}
            label="Emails Sent"
            value={withMetrics.length ? sent : '—'}
            tone="blue"
            loading={isLoading}
          />
          <MetricCard
            icon={Inbox}
            label="Replies Received"
            value={withMetrics.length ? replies : '—'}
            tone="emerald"
            loading={isLoading}
          />
          <MetricCard
            icon={TrendingUp}
            label="Team Reply Rate"
            value={replyRate != null ? `${replyRate.toFixed(1)}%` : '—'}
            tone="violet"
            loading={isLoading}
          />
          <MetricCard
            icon={CheckSquare}
            label="Actions Completed"
            value={completed}
            tone="emerald"
            loading={isLoading}
          />
          <MetricCard
            icon={Calendar}
            label="Meetings Booked"
            value={meetings}
            tone="amber"
            loading={isLoading}
          />
          <MetricCard
            icon={Users}
            label="Active Members"
            value={`${active.length} / ${members.length}`}
            tone="rose"
            loading={isLoading}
          />
        </div>
      </section>

      <MemberComparison
        members={members}
        loading={isLoading}
        skipped={skipped}
      />

      <TeamProductivity
        report={report}
        loading={isLoading}
        days={selected.days}
      />
    </>
  );
}

function names(members: ProductivityMember[]) {
  const shown = members.slice(0, 3).map((m) => m.name || m.email);
  const rest = members.length - shown.length;
  return rest > 0 ? `${shown.join(', ')} and ${rest} more` : shown.join(', ');
}

/**
 * Actions completed per member, drawn as bars against the busiest member so
 * the spread across the team is visible at a glance.
 */
function MemberComparison({
  members,
  loading,
  skipped,
}: {
  members: ProductivityMember[];
  loading?: boolean;
  skipped: number;
}) {
  if (loading) {
    return (
      <section className="space-y-3">
        <Skeleton className="h-7 w-56" />
        <Skeleton className="h-40" />
      </section>
    );
  }
  if (members.length < 2) return null;

  const max = Math.max(1, ...members.map((m) => m.actions_completed));
  const ranked = [...members].sort(
    (a, b) => b.actions_completed - a.actions_completed
  );

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">
          Workload across the team
        </h2>
        <p className="text-[0.9rem] text-muted-foreground">
          Actions completed per member · {skipped} skipped in total
        </p>
      </div>
      <div className="space-y-3 rounded-xl border bg-card p-5">
        {ranked.map((m) => (
          <div key={m.user_id} className="flex items-center gap-3">
            <span className="w-40 shrink-0 truncate text-sm font-medium">
              {m.name || m.email}
              {m.is_you && (
                <span className="ml-1.5 text-[0.7rem] font-semibold uppercase text-violet-600">
                  You
                </span>
              )}
            </span>
            <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-violet-500"
                style={{ width: `${(m.actions_completed / max) * 100}%` }}
              />
            </div>
            <span className="w-10 shrink-0 text-right text-sm font-semibold tabular-nums">
              {m.actions_completed}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
