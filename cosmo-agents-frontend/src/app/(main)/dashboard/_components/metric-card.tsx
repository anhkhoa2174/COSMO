'use client';

import { ArrowRight, MoveRight, TrendingDown, TrendingUp } from 'lucide-react';
import Link from 'next/link';
import React from 'react';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

const TONE = {
  violet: 'bg-violet-100 text-violet-600',
  blue: 'bg-blue-100 text-blue-600',
  emerald: 'bg-emerald-100 text-emerald-600',
  amber: 'bg-amber-100 text-amber-600',
  rose: 'bg-rose-100 text-rose-600',
  slate: 'bg-slate-100 text-slate-600',
} as const;

export type MetricTone = keyof typeof TONE;

/**
 * A card in the "What needs your attention" row: count, label, one-line reason,
 * and an arrow through to the page that resolves it.
 */
export function AttentionCard({
  icon: Icon,
  count,
  label,
  description,
  href,
  tone = 'violet',
  loading,
}: {
  icon: React.ComponentType<{ className?: string }>;
  count: number | undefined;
  label: string;
  description: string;
  href: string;
  tone?: MetricTone;
  loading?: boolean;
}) {
  return (
    <Link
      href={href}
      className="group flex flex-col gap-3 rounded-xl border bg-card p-4 transition-shadow hover:shadow-md"
    >
      <div className="flex items-center gap-3">
        <span
          className={cn(
            'grid size-9 shrink-0 place-items-center rounded-lg',
            TONE[tone]
          )}
        >
          <Icon className="size-[1.15rem]" />
        </span>
        {loading ? (
          <Skeleton className="h-7 w-8" />
        ) : (
          <span className="text-2xl font-bold leading-none">{count ?? 0}</span>
        )}
        <span className="min-w-0 flex-1 truncate text-[0.95rem] font-medium">
          {label}
        </span>
        <ArrowRight className="size-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
      </div>
      <p className="text-[0.8rem] leading-snug text-muted-foreground">
        {description}
      </p>
    </Link>
  );
}

/**
 * A card in the "Overview" grid: label, value, and a delta chip.
 *
 * `delta` is null whenever no comparable prior window exists — the metrics
 * endpoint only returns the selected window, so we say so rather than
 * inventing a baseline.
 */
export function MetricCard({
  icon: Icon,
  label,
  value,
  delta,
  deltaLabel,
  tone = 'violet',
  loading,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string | number;
  delta?: number | null;
  deltaLabel?: string;
  tone?: MetricTone;
  loading?: boolean;
}) {
  const up = delta != null && delta > 0;
  const down = delta != null && delta < 0;
  const DeltaIcon = up ? TrendingUp : down ? TrendingDown : MoveRight;

  return (
    <div className="flex flex-col gap-3 rounded-xl border bg-card p-5">
      <div className="flex items-start justify-between gap-3">
        <p className="text-[0.9rem] font-medium text-muted-foreground">
          {label}
        </p>
        <span
          className={cn(
            'grid size-9 shrink-0 place-items-center rounded-lg',
            TONE[tone]
          )}
        >
          <Icon className="size-[1.15rem]" />
        </span>
      </div>
      {loading ? (
        <Skeleton className="h-9 w-24" />
      ) : (
        <p className="text-3xl font-bold leading-none tracking-tight">
          {value}
        </p>
      )}
      {!loading && deltaLabel && (
        <span
          className={cn(
            'inline-flex w-fit items-center gap-1.5 rounded-md px-2 py-1 text-[0.75rem] font-medium',
            up && 'bg-emerald-50 text-emerald-700',
            down && 'bg-rose-50 text-rose-700',
            !up && !down && 'bg-muted text-muted-foreground'
          )}
        >
          <DeltaIcon className="size-3.5" />
          {delta != null
            ? `${delta > 0 ? '+' : ''}${delta.toFixed(delta % 1 === 0 ? 0 : 1)}% ${deltaLabel}`
            : deltaLabel}
        </span>
      )}
    </div>
  );
}
