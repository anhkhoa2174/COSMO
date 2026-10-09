'use client';

import { useQuery } from '@tanstack/react-query';
import { format, isToday, subDays } from 'date-fns';
import {
  AlertCircle,
  BarChart3,
  Calendar,
  CalendarRange,
  CheckSquare,
  Clock,
  Flag,
  Inbox,
  Mail,
  PauseCircle,
  PlayCircle,
  Sparkles,
  TrendingUp,
  Zap,
} from 'lucide-react';
import React from 'react';

import { PageHero } from '@/components/ui/page-hero';
import { cn } from '@/lib/utils';
import CampaignApi from '@/network/client/campaign';
import DailyActionsApi from '@/network/client/daily-actions';
import OutreachApi from '@/network/client/outreach';
import TaskApi from '@/network/client/task';
import EmailApi from '@/network/client/email';
import AgentApi from '@/network/client/agent';
import {
  AttentionCard,
  MetricCard,
} from '@/app/(main)/dashboard/_components/metric-card';
import { PerformanceBreakdown } from '@/app/(main)/dashboard/_components/performance-breakdown';
import { PipelineAnalytics } from '@/app/(main)/dashboard/_components/pipeline-analytics';
import { ActivityCharts } from '@/app/(main)/dashboard/_components/activity-charts';
import { TeamProductivity } from '@/app/(main)/dashboard/_components/team-productivity';
import TeamProductivityApi from '@/network/client/team-productivity';

/**
 * `v1/mcp/outcome-metrics` only accepts these two windows, so the period
 * selector offers exactly what the API can answer.
 */
const PERIODS = [
  { id: '7d' as const, label: 'Last 7 days', days: 7 },
  { id: '30d' as const, label: 'Last 30 days', days: 30 },
];

/** A contact counts as stalled once it has been silent this long. */
const STALLED_DAYS = 5;

function greeting(hour: number) {
  if (hour < 12) return 'Good morning';
  if (hour < 18) return 'Good afternoon';
  return 'Good evening';
}

/**
 * The personal dashboard: one representative's own pipeline, meetings, tasks
 * and results. Every query here is scoped to the caller by the backend, so a
 * member sees only their own work. An admin reaches it through the "My work"
 * view, alongside the team dashboard.
 */
export function MyDashboard() {
  const [period, setPeriod] = React.useState<'7d' | '30d'>('30d');

  // Rendered only after mount: the greeting and "today" both depend on the
  // viewer's clock, which the server cannot know.
  const [now, setNow] = React.useState<Date | null>(null);
  React.useEffect(() => setNow(new Date()), []);

  const { data: metricsRes, isLoading: loadingMetrics } = useQuery({
    queryKey: ['dashboard-outcome-metrics', period],
    queryFn: () => DailyActionsApi.getOutcomeMetrics(period),
  });

  // The wider window doubles as the baseline for the 7-day view.
  const { data: baselineRes } = useQuery({
    queryKey: ['dashboard-outcome-metrics', '30d'],
    queryFn: () => DailyActionsApi.getOutcomeMetrics('30d'),
  });

  // Counted over every contact the caller owns. The suggestion list this used
  // to read holds only contacts due an action now, so a contact who had
  // replied was never in it and "replies to answer" read zero.
  const { data: pipelineRes, isLoading: loadingPipeline } = useQuery({
    queryKey: ['dashboard-pipeline-summary'],
    queryFn: () => DailyActionsApi.getPipelineSummary(),
  });

  const { data: meetingsRes, isLoading: loadingMeetings } = useQuery({
    queryKey: ['dashboard-meetings'],
    queryFn: () => OutreachApi.getAllMeetings(),
  });

  // Both counts come from the server's total. Counting overdue rows in the
  // browser read only the first page (the API caps it at 50), and the list
  // was not limited to the caller's own campaigns.
  const { data: tasksRes, isLoading: loadingTasks } = useQuery({
    queryKey: ['dashboard-tasks-pending'],
    queryFn: () => TaskApi.list({ status: 'pending', mine: true, limit: 1 }),
  });
  const { data: overdueRes, isLoading: loadingOverdue } = useQuery({
    queryKey: ['dashboard-tasks-overdue'],
    queryFn: () =>
      TaskApi.list({ status: 'pending', mine: true, overdue: true, limit: 1 }),
  });

  const { data: campaignsRes, isLoading: loadingCampaigns } = useQuery({
    queryKey: ['dashboard-campaigns'],
    queryFn: () =>
      CampaignApi.search({ filter: { is_deleted: false } }, { limit: 200 }),
  });

  // Emails carry created_at, from_email and the classifier's intent labels —
  // enough for a real daily trend and an intent breakdown.
  const { data: emailsRes, isLoading: loadingEmails } = useQuery({
    queryKey: ['dashboard-emails'],
    queryFn: () => EmailApi.search({ filter: {} }, { offset: 0, limit: 500 }),
  });

  // Direction is decided by matching the sender against the agents' own
  // inboxes, rather than guessing from whether an intent label is present.
  const { data: agentsRes } = useQuery({
    queryKey: ['dashboard-agents'],
    queryFn: () => AgentApi.list({ filter_: {} }),
  });

  // Scope is decided server-side from the caller's role: an admin gets a row
  // per member, everyone else gets only their own.
  const { data: productivity, isLoading: loadingProductivity } = useQuery({
    queryKey: ['dashboard-productivity', period],
    queryFn: () => TeamProductivityApi.get(period),
  });

  const ownProductivity = productivity && {
    ...productivity,
    scope: 'self' as const,
    members: productivity.members.filter((m) => m.is_you),
  };

  const metrics = metricsRes?.data;
  const baseline = baselineRes?.data;
  const pipeline = pipelineRes?.data;
  const byStage = pipeline?.by_stage ?? {};
  const meetings = meetingsRes?.data ?? [];
  const campaigns = campaignsRes?.data?.list ?? [];
  const emails = (emailsRes?.data?.list ?? []).map((row: any) => row.entity);
  const agentEmails = (agentsRes?.data?.list ?? [])
    .flatMap((row: any) => row.entity)
    .map((agent: any) => agent?.email)
    .filter(Boolean) as string[];

  // Two different questions hide behind the word "unreplied". This one is the
  // actionable half: the prospect wrote back and is waiting on us.
  const repliesToAnswer = byStage.REPLIED ?? 0;

  // The other half: we wrote, they have not answered yet. Nothing to action
  // today, but it is the number people look for first.
  // NO_REPLY only: a COLD contact has not been written to yet, so there is
  // nothing for them to answer.
  const awaitingReply = byStage.NO_REPLY ?? 0;

  const meetingsToday = meetings.filter(
    (m) => m.status === 'scheduled' && m.time && isToday(new Date(m.time))
  ).length;

  const overdueTasks = overdueRes?.data?.total;

  const stalled = pipeline?.stalled ?? 0;

  const attentionTotal =
    repliesToAnswer + meetingsToday + (overdueTasks ?? 0) + stalled;

  // The search endpoint wraps each campaign in a row object; status lives on
  // the nested entity, not the row.
  const activeCampaigns = campaigns.filter(
    (c) => c.entity?.status === 'active'
  ).length;
  const pausedCampaigns = campaigns.filter(
    (c) => c.entity?.status === 'paused'
  ).length;

  const selected = PERIODS.find((p) => p.id === period)!;
  const rangeLabel = now
    ? `${format(subDays(now, selected.days - 1), 'MMM d, yyyy')} – ${format(now, 'MMM d, yyyy')}`
    : null;

  // The backend returns all-zero metrics with a `message` when the nightly
  // worker has not produced a row yet. Drawing that as a real 0 hides genuine
  // activity, so the cards say "not computed" instead.
  const metricsPending = Boolean(metrics?.message);
  // No emails sent means no rate, not a 0% one.
  const replyRate =
    metrics && !metricsPending && metrics.total_sent > 0
      ? metrics.reply_rate_overall * 100
      : null;

  // Only reply *rate* is comparable across windows — the raw counts are a
  // subset of the wider window, so differencing them would be meaningless.
  // On the 30-day view the baseline *is* the selection, so there is nothing
  // to compare and the chip is dropped entirely.
  const rateDelta =
    period === '7d' && metrics && baseline && baseline.reply_rate_overall > 0
      ? (metrics.reply_rate_overall - baseline.reply_rate_overall) * 100
      : null;
  const deltaLabel = rateDelta != null ? 'vs 30-day avg' : undefined;

  return (
    <>
      <PageHero
        icon={Sparkles}
        eyebrow="My work"
        accent="violet"
        title={now ? greeting(now.getHours()) : 'Welcome back'}
        description={now ? format(now, 'EEEE, MMMM d, yyyy') : null}
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
                ? `${replyRate.toFixed(1)}% reply rate`
                : '— reply rate'}
            </span>
          </div>
        }
      />

      <section className="space-y-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">
            What needs your attention
          </h2>
          <p className="text-[0.9rem] text-muted-foreground">
            {attentionTotal === 0
              ? 'You are all caught up'
              : `${attentionTotal} item${attentionTotal === 1 ? '' : 's'} waiting on you`}
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <AttentionCard
            icon={Inbox}
            count={repliesToAnswer}
            label="Replies to answer"
            description="Prospects wrote back and are waiting on you"
            href="/ai-inboxes"
            tone="violet"
            loading={loadingPipeline}
          />
          <AttentionCard
            icon={Calendar}
            count={meetingsToday}
            label="Meetings today"
            description="Prepare agenda and join links"
            href="/meetings"
            tone="amber"
            loading={loadingMeetings}
          />
          <AttentionCard
            icon={AlertCircle}
            count={overdueTasks}
            label="Overdue tasks"
            description="Scheduled sends that have not run"
            href="/tasks"
            tone="rose"
            loading={loadingOverdue}
          />
          <AttentionCard
            icon={Zap}
            count={stalled}
            label="Stalled follow-ups"
            description={`Contacts silent for ${STALLED_DAYS}+ days — try a new angle`}
            href="/outreach"
            tone="violet"
            loading={loadingPipeline}
          />
        </div>
      </section>

      <section className="space-y-3">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">Overview</h2>
          <p className="text-[0.9rem] text-muted-foreground">
            {rangeLabel ? `Metrics for ${rangeLabel}` : 'Loading period…'}
          </p>
          {metricsPending && (
            <p className="mt-2 inline-flex items-center gap-2 rounded-md bg-amber-50 px-3 py-1.5 text-[0.8rem] text-amber-800">
              <AlertCircle className="size-3.5 shrink-0" />
              Not computed yet — the nightly job has not produced these numbers.
              Campaign, task and team counts below are live.
            </p>
          )}
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            icon={Mail}
            label="Emails Sent"
            value={metricsPending ? '—' : (metrics?.total_sent ?? 0)}
            tone="blue"
            loading={loadingMetrics}
          />
          <MetricCard
            icon={Inbox}
            label="Replies Received"
            value={metricsPending ? '—' : (metrics?.total_replied ?? 0)}
            tone="emerald"
            loading={loadingMetrics}
          />
          <MetricCard
            icon={Clock}
            label="Awaiting Reply"
            value={awaitingReply}
            tone="amber"
            loading={loadingPipeline}
          />
          <MetricCard
            icon={TrendingUp}
            label="Reply Rate"
            value={replyRate != null ? `${replyRate.toFixed(1)}%` : '—'}
            delta={rateDelta}
            deltaLabel={deltaLabel}
            tone="violet"
            loading={loadingMetrics}
          />
          <MetricCard
            icon={Calendar}
            label="Meetings Booked"
            value={metricsPending ? '—' : (metrics?.total_meetings ?? 0)}
            tone="amber"
            loading={loadingMetrics}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            icon={PlayCircle}
            label="Active Campaigns"
            value={activeCampaigns}
            tone="emerald"
            loading={loadingCampaigns}
          />
          <MetricCard
            icon={PauseCircle}
            label="Paused Campaigns"
            value={pausedCampaigns}
            tone="amber"
            loading={loadingCampaigns}
          />
          <MetricCard
            icon={CheckSquare}
            label="Pending Tasks"
            value={tasksRes?.data?.total ?? 0}
            tone="violet"
            loading={loadingTasks}
          />
          <MetricCard
            icon={Calendar}
            label="Upcoming Meetings"
            value={meetings.filter((m) => m.status === 'scheduled').length}
            tone="rose"
            loading={loadingMeetings}
          />
        </div>
      </section>

      <TeamProductivity
        report={ownProductivity}
        loading={loadingProductivity}
        days={selected.days}
      />

      <ActivityCharts
        emails={emails}
        agentEmails={agentEmails}
        days={selected.days}
        loading={loadingEmails}
      />

      <PipelineAnalytics
        summary={pipeline}
        sent={metrics?.total_sent ?? 0}
        replied={metrics?.total_replied ?? 0}
        replyRate={metrics?.reply_rate_overall ?? 0}
        meetings={metrics?.total_meetings ?? 0}
        loading={loadingPipeline || loadingMetrics}
      />

      <PerformanceBreakdown metrics={metrics} loading={loadingMetrics} />

      <UpcomingMeetings meetings={meetings} />
    </>
  );
}

function UpcomingMeetings({
  meetings,
}: {
  meetings: Awaited<ReturnType<typeof OutreachApi.getAllMeetings>>['data'];
}) {
  const upcoming = (meetings ?? []).filter((m) => m.status === 'scheduled');
  if (upcoming.length === 0) return null;

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">
          Upcoming meetings
        </h2>
        <p className="text-[0.9rem] text-muted-foreground">
          Next {Math.min(upcoming.length, 5)} on the calendar
        </p>
      </div>
      <div className="divide-y rounded-xl border bg-card">
        {upcoming.slice(0, 5).map((meeting) => (
          <div
            key={meeting.id}
            className="flex items-center justify-between gap-4 p-4"
          >
            <div className="flex min-w-0 items-center gap-3">
              <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-amber-100 text-amber-600">
                <Flag className="size-[1.15rem]" />
              </span>
              <div className="min-w-0">
                <p className="truncate font-medium">
                  {meeting.title || 'Meeting'}
                </p>
                <p className="truncate text-[0.85rem] text-muted-foreground">
                  {format(new Date(meeting.time), 'MMM d, yyyy HH:mm')}
                  {meeting.channel && ` · ${meeting.channel}`}
                </p>
              </div>
            </div>
            {meeting.location && (
              <span className="shrink-0 text-[0.85rem] text-muted-foreground">
                {meeting.location}
              </span>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}
