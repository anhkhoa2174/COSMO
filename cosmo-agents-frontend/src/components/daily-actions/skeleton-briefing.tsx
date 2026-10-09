'use client';

import { Skeleton } from '@/components/ui/skeleton';

export function SkeletonBriefing() {
  return (
    <div className="space-y-5">
      {/* Greeting line */}
      <Skeleton className="h-5 w-3/4 bg-muted" />

      {/* Strategic reasoning block */}
      <div className="space-y-2 border-l-4 border-border pl-4">
        <Skeleton className="h-4 w-full bg-muted" />
        <Skeleton className="h-4 w-5/6 bg-muted" />
        <Skeleton className="h-4 w-2/3 bg-muted" />
      </div>

      {/* Category badge chips row */}
      <div className="flex gap-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-28 rounded-full bg-muted" />
        ))}
      </div>

      {/* Category section headers */}
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="flex items-center gap-2.5 px-[18px] py-3.5">
            <Skeleton className="h-5 w-5 rounded bg-muted-foreground/15" />
            <Skeleton className="h-4 w-40 bg-muted-foreground/15" />
            <Skeleton className="ml-auto h-4 w-4 bg-muted-foreground/15" />
          </div>
        </div>
      ))}
    </div>
  );
}
