'use client';

import { Award, MessagesSquare } from 'lucide-react';
import React from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import type { OutcomeMetrics } from '@/types/daily-actions';

/**
 * Reply rate is the same measure in every breakdown, so each card is a single
 * series: one hue, no legend, ranked high→low. Identity is carried by the row
 * label, not by colour.
 */
/**
 * `outcome_metrics` stores these columns as `base.JSONB`, which is `[]byte` on
 * the Go side with no MarshalJSON — so the API serialises them as a base64
 * string, not as JSON. Coerce whatever arrives back into a usable value rather
 * than letting `.map` blow up the whole dashboard.
 */
function parseJsonb(value: unknown): unknown {
  if (value == null) return null;
  if (typeof value !== 'string') return value;

  const attempt = (text: string) => {
    try {
      return JSON.parse(text);
    } catch {
      return null;
    }
  };

  const direct = attempt(value);
  if (direct !== null) return direct;

  try {
    return attempt(atob(value));
  } catch {
    return null;
  }
}

function asStringArray(value: unknown): string[] {
  const parsed = parseJsonb(value);
  return Array.isArray(parsed)
    ? parsed.filter((item): item is string => typeof item === 'string')
    : [];
}

function asRateRecord(value: unknown): Record<string, number> {
  const parsed = parseJsonb(value);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
  const out: Record<string, number> = {};
  for (const [key, raw] of Object.entries(parsed as Record<string, unknown>)) {
    const num = typeof raw === 'number' ? raw : Number(raw);
    if (Number.isFinite(num)) out[key] = num;
  }
  return out;
}

function toPercent(values: Record<string, number>) {
  const entries = Object.entries(values).filter(
    ([, v]) => typeof v === 'number' && !Number.isNaN(v)
  );
  if (entries.length === 0) return [];

  // The API expresses reply_rate_overall as a 0–1 fraction; tolerate a backend
  // that switches to 0–100 rather than rendering 4300%.
  const max = Math.max(...entries.map(([, v]) => v));
  const scale = max > 1 ? 1 : 100;

  return entries
    .map(([label, value]) => ({ label, value: value * scale }))
    .sort((a, b) => b.value - a.value);
}

function BarList({
  title,
  description,
  values,
}: {
  title: string;
  description: string;
  values: Record<string, number> | undefined;
}) {
  const rows = React.useMemo(() => toPercent(values ?? {}), [values]);
  if (rows.length === 0) return null;

  // Bars are scaled against the best performer so differences stay readable
  // when every rate is small.
  const peak = Math.max(...rows.map((r) => r.value), 1);

  return (
    <div className="rounded-xl border bg-card p-5">
      <h3 className="font-semibold tracking-tight">{title}</h3>
      <p className="mt-0.5 text-[0.8rem] text-muted-foreground">
        {description}
      </p>
      <ul className="mt-4 space-y-3">
        {rows.map((row) => (
          <li
            key={row.label}
            className="grid grid-cols-[7rem_1fr_3rem] items-center gap-3"
          >
            <span
              className="truncate text-[0.85rem] capitalize"
              title={row.label}
            >
              {row.label.replace(/_/g, ' ')}
            </span>
            <span
              className="h-2 rounded-full bg-muted"
              role="img"
              aria-label={`${row.label}: ${row.value.toFixed(1)} percent reply rate`}
            >
              <span
                className="block h-2 rounded-r-[4px] bg-violet-600 dark:bg-violet-500"
                style={{ width: `${Math.max((row.value / peak) * 100, 2)}%` }}
              />
            </span>
            <span className="text-right text-[0.85rem] tabular-nums text-muted-foreground">
              {row.value.toFixed(1)}%
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function PerformanceBreakdown({
  metrics,
  loading,
}: {
  metrics: OutcomeMetrics | undefined;
  loading?: boolean;
}) {
  if (loading) {
    return (
      <section className="space-y-3">
        <h2 className="text-xl font-semibold tracking-tight">
          What is working
        </h2>
        <div className="grid gap-4 lg:grid-cols-2">
          <Skeleton className="h-52 rounded-xl" />
          <Skeleton className="h-52 rounded-xl" />
        </div>
      </section>
    );
  }

  const strategies = asStringArray(metrics?.top_performing_strategies);
  // The nightly rollup does not compute this yet, so it arrives as 0. Shown
  // only when there were meetings and a real average: "0.0 touches before a
  // prospect books" read as a measured result.
  const avg = metrics?.avg_messages_to_meeting ?? 0;
  const avgToMeeting =
    metrics && metrics.total_meetings > 0 && avg > 0 ? avg : null;

  const cards = [
    {
      title: 'Reply rate by channel',
      description: 'Where prospects actually answer',
      values: asRateRecord(metrics?.reply_rate_by_channel),
    },
    {
      title: 'Reply rate by strategy',
      description: 'Which playbook earns the response',
      values: asRateRecord(metrics?.reply_rate_by_strategy),
    },
    {
      title: 'Reply rate by industry',
      description: 'Who your message lands with',
      values: asRateRecord(metrics?.reply_rate_by_industry),
    },
    {
      title: 'Reply rate by time of day',
      description: 'When to schedule the send',
      values: asRateRecord(metrics?.reply_rate_by_time_of_day),
    },
  ].filter((card) => Object.keys(card.values ?? {}).length > 0);

  const hasHeadline = avgToMeeting != null || strategies.length > 0;
  if (cards.length === 0 && !hasHeadline) return null;

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">
          What is working
        </h2>
        <p className="text-[0.9rem] text-muted-foreground">
          Breakdowns the AI uses when it picks the next action
        </p>
      </div>

      {hasHeadline && (
        <div className="grid gap-4 sm:grid-cols-2">
          {avgToMeeting != null && (
            <div className="rounded-xl border bg-card p-5">
              <div className="flex items-start justify-between gap-3">
                <p className="text-[0.9rem] font-medium text-muted-foreground">
                  Messages to a meeting
                </p>
                <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-violet-100 text-violet-600">
                  <MessagesSquare className="size-[1.15rem]" />
                </span>
              </div>
              <p className="mt-3 text-3xl font-bold leading-none tracking-tight">
                {avgToMeeting.toFixed(1)}
              </p>
              <p className="mt-2 text-[0.8rem] text-muted-foreground">
                Average touches before a prospect books
              </p>
            </div>
          )}

          {strategies.length > 0 && (
            <div className="rounded-xl border bg-card p-5">
              <div className="flex items-start justify-between gap-3">
                <p className="text-[0.9rem] font-medium text-muted-foreground">
                  Top performing strategies
                </p>
                <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-amber-100 text-amber-600">
                  <Award className="size-[1.15rem]" />
                </span>
              </div>
              <ul className="mt-3 flex flex-wrap gap-2">
                {strategies.map((strategy) => (
                  <li
                    key={strategy}
                    className="rounded-full border bg-muted/60 px-3 py-1 text-[0.8rem] font-medium capitalize"
                  >
                    {strategy.replace(/_/g, ' ')}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {cards.length > 0 && (
        <div className="grid gap-4 lg:grid-cols-2">
          {cards.map((card) => (
            <BarList key={card.title} {...card} />
          ))}
        </div>
      )}
    </section>
  );
}
