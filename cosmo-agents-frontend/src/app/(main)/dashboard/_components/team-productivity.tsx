'use client';

import { AlertCircle, CalendarCheck, Trophy, Users } from 'lucide-react';

import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type {
  ProductivityMember,
  ProductivityReport,
} from '@/network/client/team-productivity';

/**
 * Per-member productivity.
 *
 * The same component renders both roles. An admin's report carries a row per
 * member and reads as a team table; a member's report carries exactly one row —
 * their own — and reads as a personal summary. The server decides which,
 * so there is no client-side branch that could be flipped to see a peer's
 * numbers.
 */
export function TeamProductivity({
  report,
  loading,
  days,
}: {
  report?: ProductivityReport;
  loading?: boolean;
  days: number;
}) {
  if (loading) {
    return (
      <section className="space-y-3">
        <Skeleton className="h-7 w-56" />
        <Skeleton className="h-48" />
      </section>
    );
  }

  if (!report || report.members.length === 0) return null;

  const isTeam = report.scope === 'team';
  const members = report.members;
  const totals = members.reduce(
    (acc, m) => ({
      completed: acc.completed + m.actions_completed,
      meetings: acc.meetings + m.meetings_booked,
    }),
    { completed: 0, meetings: 0 }
  );
  const nobodyActive = totals.completed === 0 && totals.meetings === 0;

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">
            {isTeam ? 'Team productivity' : 'Your productivity'}
          </h2>
          <p className="text-[0.9rem] text-muted-foreground">
            {isTeam
              ? `${members.length} member${members.length === 1 ? '' : 's'} · last ${days} days`
              : `What you got done in the last ${days} days`}
          </p>
        </div>
        {isTeam && (
          <span className="inline-flex items-center gap-2 rounded-full border bg-background px-3 py-1.5 text-[0.8rem] font-medium">
            <CalendarCheck className="size-3.5 text-violet-600" />
            {totals.completed} action{totals.completed === 1 ? '' : 's'} ·{' '}
            {totals.meetings} meeting{totals.meetings === 1 ? '' : 's'}
          </span>
        )}
      </div>

      {report.metrics_pending && (
        <p className="inline-flex items-center gap-2 rounded-md bg-amber-50 px-3 py-1.5 text-[0.8rem] text-amber-800">
          <AlertCircle className="size-3.5 shrink-0" />
          Reply rates are not computed yet — the nightly job has not run. The
          action and meeting counts below are live.
        </p>
      )}

      <div className="overflow-x-auto rounded-xl border bg-card">
        <table className="w-full min-w-[46rem] text-sm">
          <thead>
            <tr className="border-b text-left text-[0.8rem] text-muted-foreground">
              <th className="px-4 py-3 font-medium">
                {isTeam ? 'Member' : 'You'}
              </th>
              <th className="px-4 py-3 text-right font-medium">Completed</th>
              <th className="px-4 py-3 text-right font-medium">Skipped</th>
              <th className="px-4 py-3 text-right font-medium">Active days</th>
              <th className="px-4 py-3 text-right font-medium">Meetings</th>
              <th className="px-4 py-3 text-right font-medium">Emails sent</th>
              <th className="px-4 py-3 text-right font-medium">Reply rate</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {members.map((m, i) => (
              <MemberRow
                key={m.user_id}
                member={m}
                // Only worth calling out a leader when there is a team to lead
                // and somebody actually did something.
                leader={isTeam && i === 0 && m.actions_completed > 0}
              />
            ))}
          </tbody>
        </table>
      </div>

      {nobodyActive && (
        <p className="text-[0.85rem] text-muted-foreground">
          No actions were completed in this window.
        </p>
      )}
    </section>
  );
}

function MemberRow({
  member,
  leader,
}: {
  member: ProductivityMember;
  leader: boolean;
}) {
  // Nothing sent means no rate, not a 0% one, as on the summary cards.
  const rate =
    member.metrics && member.metrics.emails_sent > 0
      ? `${(member.metrics.reply_rate * 100).toFixed(1)}%`
      : '—';

  return (
    <tr className={cn(member.is_you && 'bg-violet-50/60')}>
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <Avatar member={member} />
          <div className="min-w-0">
            <p className="flex items-center gap-1.5 truncate font-medium">
              {member.name || member.email}
              {member.is_you && (
                <span className="rounded bg-violet-100 px-1.5 py-0.5 text-[0.65rem] font-semibold uppercase tracking-wide text-violet-700">
                  You
                </span>
              )}
              {leader && (
                <Trophy className="size-3.5 shrink-0 text-amber-500" />
              )}
            </p>
            <p className="truncate text-[0.8rem] capitalize text-muted-foreground">
              {member.role}
            </p>
          </div>
        </div>
      </td>
      <td className="px-4 py-3 text-right font-semibold tabular-nums">
        {member.actions_completed}
      </td>
      <td className="px-4 py-3 text-right tabular-nums text-muted-foreground">
        {member.actions_skipped}
      </td>
      <td className="px-4 py-3 text-right tabular-nums text-muted-foreground">
        {member.active_days}
      </td>
      <td className="px-4 py-3 text-right tabular-nums">
        {member.meetings_booked}
      </td>
      <td className="px-4 py-3 text-right tabular-nums text-muted-foreground">
        {member.metrics ? member.metrics.emails_sent : '—'}
      </td>
      <td className="px-4 py-3 text-right tabular-nums">{rate}</td>
    </tr>
  );
}

function Avatar({ member }: { member: ProductivityMember }) {
  const initials = (member.name || member.email)
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase();

  if (member.picture) {
    // eslint-disable-next-line @next/next/no-img-element
    return (
      <img
        src={member.picture}
        alt=""
        className="size-9 shrink-0 rounded-full object-cover"
      />
    );
  }

  return (
    <span className="grid size-9 shrink-0 place-items-center rounded-full bg-muted text-[0.75rem] font-semibold text-muted-foreground">
      {initials || <Users className="size-4" />}
    </span>
  );
}
