import React from 'react';
import { cn } from '@/lib/utils';

/**
 * A consistent bar at the top of every pane in the composer.
 *
 * Without it the three panes read as three unlabelled boxes and nothing tells
 * the reader that typing in one drives another. The `hint` slot carries that
 * relationship in words.
 */
export function PaneHeader({
  icon: Icon,
  title,
  hint,
  actions,
  className,
}: {
  icon?: React.ComponentType<{ className?: string }>;
  title: string;
  /** One short line saying what this pane does for the others. */
  hint?: string;
  actions?: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex h-12 shrink-0 items-center gap-3 border-b bg-background px-4',
        className
      )}
    >
      {Icon && (
        <span className="grid size-7 shrink-0 place-items-center rounded-lg bg-violet-100 text-violet-600">
          <Icon className="size-4" />
        </span>
      )}
      <div className="flex min-w-0 flex-1 items-baseline gap-2">
        <p className="shrink-0 text-[0.9rem] font-semibold">{title}</p>
        {hint && (
          <p className="truncate text-[0.78rem] text-muted-foreground">
            {hint}
          </p>
        )}
      </div>
      {actions && (
        <div className="flex shrink-0 items-center gap-2">{actions}</div>
      )}
    </div>
  );
}
