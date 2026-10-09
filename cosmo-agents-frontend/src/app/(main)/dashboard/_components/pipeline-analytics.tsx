'use client';

import React from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { PipelineSummary } from '@/network/client/daily-actions';

/**
 * Categorical slots 1–5 of the validated default palette, in fixed order.
 * Verified with the palette validator (light surface): lightness band, chroma
 * floor, CVD separation and normal-vision floor all pass. Three slots sit below
 * 3:1 against the surface, so every segment carries a visible label and value —
 * identity is never colour alone.
 */
const STATE_STYLES: { key: string; label: string; hex: string }[] = [
  { key: 'COLD', label: 'Cold', hex: '#2a78d6' },
  { key: 'NO_REPLY', label: 'Awaiting reply', hex: '#eb6834' },
  { key: 'REPLIED', label: 'Replied', hex: '#1baf7a' },
  { key: 'POST_MEETING', label: 'Post-meeting', hex: '#eda100' },
  { key: 'DROPPED', label: 'Dropped', hex: '#e87ba4' },
];

/** Ordinal steps of the sequential blue ramp — nothing lighter than step 250. */
const FUNNEL_STEPS = ['#86b6ef', '#3987e5', '#1c5cab'];

function Donut({
  slices,
  total,
}: {
  slices: { label: string; value: number; hex: string }[];
  total: number;
}) {
  const radius = 60;
  const circumference = 2 * Math.PI * radius;
  // A 2px gap of surface between neighbouring arcs, expressed in path units.
  const gap = total > 0 ? 2 : 0;

  let offset = 0;

  return (
    <svg viewBox="0 0 160 160" className="size-40 shrink-0 -rotate-90">
      <circle
        cx="80"
        cy="80"
        r={radius}
        fill="none"
        stroke="currentColor"
        strokeWidth="18"
        className="text-muted"
      />
      {total > 0 &&
        slices.map((slice) => {
          const length = (slice.value / total) * circumference;
          const dash = Math.max(length - gap, 0);
          const el = (
            <circle
              key={slice.label}
              cx="80"
              cy="80"
              r={radius}
              fill="none"
              stroke={slice.hex}
              strokeWidth="18"
              strokeDasharray={`${dash} ${circumference - dash}`}
              strokeDashoffset={-offset}
            >
              <title>{`${slice.label}: ${slice.value}`}</title>
            </circle>
          );
          offset += length;
          return el;
        })}
    </svg>
  );
}

function StateBreakdown({ summary }: { summary?: PipelineSummary }) {
  const counts = new Map(Object.entries(summary?.by_stage ?? {}));

  // Fixed order — a slot's hue follows the state, never its rank, so a state
  // dropping to zero must not repaint the others.
  const slices = STATE_STYLES.map((style) => ({
    label: style.label,
    hex: style.hex,
    value: counts.get(style.key) ?? 0,
  }));
  const total = slices.reduce((sum, slice) => sum + slice.value, 0);

  return (
    <div className="rounded-xl border bg-card p-5">
      <h3 className="font-semibold tracking-tight">Pipeline by state</h3>
      <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
        Where your {total} contact{total === 1 ? '' : 's'} currently sit
      </p>

      <div className="mt-5 flex flex-wrap items-center gap-6">
        <Donut slices={slices} total={total} />

        <ul className="min-w-[11rem] flex-1 space-y-2">
          {slices.map((slice) => {
            const share = total > 0 ? (slice.value / total) * 100 : 0;
            return (
              <li
                key={slice.label}
                className="flex items-center gap-2.5 text-[0.85rem]"
              >
                <span
                  aria-hidden="true"
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: slice.hex }}
                />
                <span className="flex-1 truncate">{slice.label}</span>
                <span className="tabular-nums text-muted-foreground">
                  {slice.value}
                </span>
                <span className="w-11 text-right tabular-nums text-muted-foreground">
                  {share.toFixed(0)}%
                </span>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}

function Funnel({
  sent,
  replied,
  replyRate,
  meetings,
}: {
  sent: number;
  replied: number;
  /** Share of threads written to that got an answer (0–1), from the rollup. */
  replyRate: number;
  meetings: number;
}) {
  const stages = [
    { label: 'Emails sent', value: sent, hex: FUNNEL_STEPS[0] },
    { label: 'Replies received', value: replied, hex: FUNNEL_STEPS[1] },
    { label: 'Meetings booked', value: meetings, hex: FUNNEL_STEPS[2] },
  ];
  const peak = Math.max(sent, 1);

  return (
    <div className="rounded-xl border bg-card p-5">
      <h3 className="font-semibold tracking-tight">Conversion funnel</h3>
      <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
        How far the selected period carried each contact
      </p>

      <ul className="mt-5 space-y-4">
        {stages.map((stage, index) => {
          const previous = index === 0 ? null : stages[index - 1].value;
          // Replies are counted per thread while sends count every follow-up
          // and a thread can answer more than once, so replies ÷ sends is not
          // a rate (it read 200%). The rollup's thread-level reply rate is.
          let rate: number | null = null;
          let rateLabel = '';
          if (index === 1) {
            rate = sent > 0 ? replyRate * 100 : null;
            rateLabel = 'of threads emailed got a reply';
          } else if (previous && previous > 0) {
            rate = (stage.value / previous) * 100;
            rateLabel = `of ${stages[index - 1].label.toLowerCase()}`;
          }

          return (
            <li key={stage.label}>
              <div className="flex items-baseline justify-between gap-3">
                <span className="text-[0.85rem]">{stage.label}</span>
                <span className="text-[0.95rem] font-semibold tabular-nums">
                  {stage.value}
                </span>
              </div>
              <div className="mt-1.5 h-2.5 rounded-full bg-muted">
                <div
                  className="h-2.5 rounded-r-[4px]"
                  style={{
                    width: `${Math.min(100, Math.max((stage.value / peak) * 100, stage.value > 0 ? 2 : 0))}%`,
                    backgroundColor: stage.hex,
                  }}
                  title={`${stage.label}: ${stage.value}`}
                />
              </div>
              {rate !== null && (
                <p className="mt-1 text-[0.75rem] text-muted-foreground">
                  {rate.toFixed(1)}% {rateLabel}
                </p>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function FollowUpDepth({ summary }: { summary?: PipelineSummary }) {
  const depth = summary?.followup_depth ?? [0, 0, 0, 0];
  const buckets = [
    'No follow-up yet',
    '1 follow-up',
    '2 follow-ups',
    '3 or more',
  ].map((label, i) => ({ label, value: depth[i] ?? 0 }));

  const peak = Math.max(...buckets.map((b) => b.value), 1);

  return (
    <div className="rounded-xl border bg-card p-5">
      <h3 className="font-semibold tracking-tight">Follow-up depth</h3>
      <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
        How hard each contact has been worked
      </p>
      <ul className="mt-5 space-y-3">
        {buckets.map((bucket) => (
          <li
            key={bucket.label}
            className="grid grid-cols-[8rem_1fr_2.5rem] items-center gap-3"
          >
            <span className="truncate text-[0.85rem]">{bucket.label}</span>
            <span className="h-2 rounded-full bg-muted">
              <span
                className="block h-2 rounded-r-[4px] bg-violet-600"
                style={{
                  width: `${Math.max((bucket.value / peak) * 100, bucket.value > 0 ? 2 : 0)}%`,
                }}
                title={`${bucket.label}: ${bucket.value}`}
              />
            </span>
            <span className="text-right text-[0.85rem] tabular-nums text-muted-foreground">
              {bucket.value}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function PipelineAnalytics({
  summary,
  sent,
  replied,
  replyRate,
  meetings,
  loading,
}: {
  summary?: PipelineSummary;
  sent: number;
  replied: number;
  replyRate: number;
  meetings: number;
  loading?: boolean;
}) {
  if (loading) {
    return (
      <section className="space-y-3">
        <h2 className="text-xl font-semibold tracking-tight">Analytics</h2>
        <div className="grid gap-4 lg:grid-cols-3">
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
          <Skeleton className="h-64 rounded-xl" />
        </div>
      </section>
    );
  }

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">Analytics</h2>
        <p className="text-[0.9rem] text-muted-foreground">
          Where contacts sit, how far they convert, and how hard they are worked
        </p>
      </div>
      <div className={cn('grid gap-4 lg:grid-cols-3')}>
        <StateBreakdown summary={summary} />
        <Funnel
          sent={sent}
          replied={replied}
          replyRate={replyRate}
          meetings={meetings}
        />
        <FollowUpDepth summary={summary} />
      </div>
    </section>
  );
}
