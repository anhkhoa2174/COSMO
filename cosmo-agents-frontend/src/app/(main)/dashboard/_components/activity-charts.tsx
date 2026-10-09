'use client';

import { format, startOfDay, subDays } from 'date-fns';
import React from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import type { Email } from '@/models/email';

/**
 * Categorical slots 1 and 2 of the validated palette. Verified with the palette
 * validator on the light surface: every check passes, including contrast — so
 * these two carry identity on their own, alongside the legend.
 */
const SERIES = {
  sent: { label: 'Sent', hex: '#2a78d6' },
  received: { label: 'Replies', hex: '#eb6834' },
};

type DayBucket = { date: Date; sent: number; received: number };

function buildBuckets(emails: Email[], agentEmails: Set<string>, days: number) {
  const today = startOfDay(new Date());
  const buckets: DayBucket[] = [];
  const index = new Map<string, DayBucket>();

  for (let i = days - 1; i >= 0; i--) {
    const date = subDays(today, i);
    const bucket: DayBucket = { date, sent: 0, received: 0 };
    buckets.push(bucket);
    index.set(format(date, 'yyyy-MM-dd'), bucket);
  }

  for (const email of emails) {
    if (!email.created_at) continue;
    const bucket = index.get(format(new Date(email.created_at), 'yyyy-MM-dd'));
    if (!bucket) continue;
    // Direction comes from the agent's own addresses rather than being inferred
    // from the presence of an intent label, which only inbound mail carries.
    if (agentEmails.has((email.from_email || '').toLowerCase())) {
      bucket.sent += 1;
    } else {
      bucket.received += 1;
    }
  }

  return buckets;
}

function ActivityChart({ buckets }: { buckets: DayBucket[] }) {
  const width = 720;
  const height = 200;
  const padding = { top: 12, right: 12, bottom: 26, left: 30 };
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;

  const peak = Math.max(...buckets.flatMap((b) => [b.sent, b.received]), 1);

  const x = (i: number) =>
    padding.left +
    (buckets.length === 1
      ? plotWidth / 2
      : (i / (buckets.length - 1)) * plotWidth);
  const y = (value: number) =>
    padding.top + plotHeight - (value / peak) * plotHeight;

  const line = (key: 'sent' | 'received') =>
    buckets
      .map((b, i) => `${i === 0 ? 'M' : 'L'} ${x(i)} ${y(b[key])}`)
      .join(' ');

  // Four ticks keeps the axis readable without crowding a 30-day window.
  const tickIndexes = Array.from(
    new Set([
      0,
      Math.floor(buckets.length / 3),
      Math.floor((2 * buckets.length) / 3),
      buckets.length - 1,
    ])
  ).filter((i) => i >= 0 && i < buckets.length);

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className="h-52 w-full"
      role="img"
      aria-label="Emails sent and replies received per day"
    >
      {/* Recessive grid: hairlines one shade off the surface, never dashed. */}
      {[0, 0.5, 1].map((fraction) => (
        <line
          key={fraction}
          x1={padding.left}
          x2={width - padding.right}
          y1={padding.top + plotHeight * fraction}
          y2={padding.top + plotHeight * fraction}
          stroke="currentColor"
          strokeWidth="1"
          className="text-border"
        />
      ))}

      {/* The midpoint label is dropped when it would repeat an end: with a
          peak of 1, Math.round(0.5) printed "1" twice. */}
      {[peak, Math.round(peak / 2), 0].map((value, i) =>
        i === 1 && (value === peak || value === 0) ? null : (
          <text
            key={value + '-' + i}
            x={padding.left - 6}
            y={padding.top + plotHeight * (i / 2) + 4}
            textAnchor="end"
            className="fill-muted-foreground text-[10px] tabular-nums"
          >
            {value}
          </text>
        )
      )}

      {tickIndexes.map((i) => (
        <text
          key={i}
          x={x(i)}
          y={height - 6}
          textAnchor={
            i === 0 ? 'start' : i === buckets.length - 1 ? 'end' : 'middle'
          }
          className="fill-muted-foreground text-[10px]"
        >
          {format(buckets[i].date, 'MMM d')}
        </text>
      ))}

      {(['sent', 'received'] as const).map((key) => (
        <path
          key={key}
          d={line(key)}
          fill="none"
          stroke={SERIES[key].hex}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      ))}

      {/* Per-point hover targets, wider than the marks themselves. */}
      {buckets.map((bucket, i) => (
        <rect
          key={i}
          x={x(i) - plotWidth / buckets.length / 2}
          y={padding.top}
          width={Math.max(plotWidth / buckets.length, 6)}
          height={plotHeight}
          fill="transparent"
        >
          <title>
            {`${format(bucket.date, 'MMM d')} — ${bucket.sent} sent, ${bucket.received} replies`}
          </title>
        </rect>
      ))}
    </svg>
  );
}

function IntentBreakdown({ emails, days }: { emails: Email[]; days: number }) {
  // Same window as the chart beside it: the card says "in this window", and
  // counting every email ever labelled made 5 of the 2 replies the chart drew.
  const since = subDays(startOfDay(new Date()), days - 1);
  const counts = new Map<string, number>();
  for (const email of emails) {
    if (!email.created_at || new Date(email.created_at) < since) continue;
    for (const intent of email.intents ?? []) {
      if (!intent) continue;
      counts.set(intent, (counts.get(intent) ?? 0) + 1);
    }
  }

  const rows = [...counts.entries()]
    .map(([label, value]) => ({ label, value }))
    .sort((a, b) => b.value - a.value);

  if (rows.length === 0) {
    return (
      <div className="rounded-xl border bg-card p-5">
        <h3 className="font-semibold tracking-tight">
          What prospects say back
        </h3>
        <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
          Intent labels the classifier assigned to inbound replies
        </p>
        <p className="mt-6 text-[0.9rem] text-muted-foreground">
          No classified replies in this window yet.
        </p>
      </div>
    );
  }

  const total = rows.reduce((sum, row) => sum + row.value, 0);
  const peak = Math.max(...rows.map((r) => r.value), 1);

  return (
    <div className="rounded-xl border bg-card p-5">
      <h3 className="font-semibold tracking-tight">What prospects say back</h3>
      <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
        {total} classified repl{total === 1 ? 'y' : 'ies'} in this window
      </p>
      <ul className="mt-5 space-y-3">
        {rows.map((row) => (
          <li
            key={row.label}
            className="grid grid-cols-[9rem_1fr_3.5rem] items-center gap-3"
          >
            <span className="truncate text-[0.85rem]" title={row.label}>
              {row.label}
            </span>
            <span className="h-2 rounded-full bg-muted">
              <span
                className="block h-2 rounded-r-[4px] bg-violet-600"
                style={{ width: `${Math.max((row.value / peak) * 100, 2)}%` }}
                title={`${row.label}: ${row.value}`}
              />
            </span>
            <span className="text-right text-[0.85rem] tabular-nums text-muted-foreground">
              {row.value}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function ActivityCharts({
  emails,
  agentEmails,
  days,
  loading,
}: {
  emails: Email[];
  agentEmails: string[];
  days: number;
  loading?: boolean;
}) {
  const buckets = React.useMemo(
    () =>
      buildBuckets(
        emails,
        new Set(agentEmails.map((e) => e.toLowerCase())),
        days
      ),
    [emails, agentEmails, days]
  );

  if (loading) {
    return (
      <section className="space-y-3">
        <h2 className="text-xl font-semibold tracking-tight">Activity</h2>
        <div className="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
          <Skeleton className="h-72 rounded-xl" />
          <Skeleton className="h-72 rounded-xl" />
        </div>
      </section>
    );
  }

  return (
    <section className="min-w-0 space-y-3">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">Activity</h2>
        <p className="text-[0.9rem] text-muted-foreground">
          Daily volume, and what came back
        </p>
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="min-w-0 rounded-xl border bg-card p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 className="font-semibold tracking-tight">
                Sent vs replies, per day
              </h3>
              <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
                Last {days} days
              </p>
            </div>
            {/* Two series always carry a legend — identity is never colour alone. */}
            <ul className="flex items-center gap-4">
              {(['sent', 'received'] as const).map((key) => (
                <li
                  key={key}
                  className="flex items-center gap-1.5 text-[0.8rem]"
                >
                  <span
                    aria-hidden="true"
                    className="h-0.5 w-4 rounded-full"
                    style={{ backgroundColor: SERIES[key].hex }}
                  />
                  <span className="text-muted-foreground">
                    {SERIES[key].label}
                  </span>
                </li>
              ))}
            </ul>
          </div>
          <div className="mt-4">
            <ActivityChart buckets={buckets} />
          </div>
        </div>

        <IntentBreakdown emails={emails} days={days} />
      </div>
    </section>
  );
}
