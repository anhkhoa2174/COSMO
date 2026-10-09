'use client';

import { format } from 'date-fns';
import { Bell, ChevronRight, HelpCircle, Search, Sparkle } from 'lucide-react';
import React from 'react';
import { cn } from '@/lib/utils';

/**
 * The date pill only renders once mounted — formatting "today" on the server
 * and again on the client is a guaranteed hydration mismatch across midnight
 * and across timezones.
 */
function TodayPill() {
  const [today, setToday] = React.useState<string | null>(null);

  React.useEffect(() => {
    setToday(format(new Date(), 'EEE, MMM d'));
  }, []);

  return (
    <span className="hidden h-8 min-w-[7.5rem] items-center justify-center gap-1.5 rounded-full border border-border bg-background px-3 text-[0.8rem] font-medium text-foreground sm:inline-flex">
      <Sparkle className="size-3.5 fill-violet-500 text-violet-500" />
      {today ?? ' '}
    </span>
  );
}

export function TopBar({
  title,
  section,
  icon: Icon,
  leftSection,
  rightSection,
  hasNotifications = true,
  utilities = true,
  onSearch,
}: {
  title: string | JSX.Element;
  section?: string;
  icon?: React.ComponentType<{ className?: string }>;
  leftSection?: React.ReactNode;
  rightSection?: React.ReactNode;
  hasNotifications?: boolean;
  /** Editor screens (campaign builder, template editor) hide the utility
   *  cluster so the bar carries only the document's own actions. */
  utilities?: boolean;
  onSearch?: () => void;
}) {
  return (
    <header className="sticky top-0 z-50 flex h-16 w-full shrink-0 items-center justify-between gap-2 border-b bg-background px-4 md:px-6">
      <div className="flex min-w-0 items-center gap-2.5">
        {leftSection}
        {Icon && (
          <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-violet-100 text-violet-600">
            <Icon className="size-4" />
          </span>
        )}
        {section && (
          <>
            <span className="hidden truncate text-[0.95rem] text-muted-foreground sm:inline">
              {section}
            </span>
            <ChevronRight className="hidden size-4 shrink-0 text-muted-foreground sm:inline" />
          </>
        )}
        {typeof title === 'string' ? (
          <p className="truncate text-[0.95rem] font-semibold">{title}</p>
        ) : (
          title
        )}
      </div>

      <div className="flex shrink-0 items-center gap-2">
        {rightSection}
        {!utilities ? null : (
          <>
            <TodayPill />
            <button
              type="button"
              onClick={onSearch}
              className="hidden h-8 items-center gap-2 rounded-full px-2.5 text-[0.85rem] text-muted-foreground transition-colors hover:text-foreground md:inline-flex"
            >
              <Search className="size-4" />
              Search
              <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 font-sans text-[0.7rem] text-muted-foreground">
                ⌘K
              </kbd>
            </button>
            <button
              type="button"
              aria-label="Notifications"
              className="relative grid size-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <Bell className="size-[1.15rem]" />
              {hasNotifications && (
                <span
                  className={cn(
                    'absolute right-1.5 top-1.5 size-2 rounded-full bg-red-500',
                    'ring-2 ring-background'
                  )}
                />
              )}
            </button>
            <button
              type="button"
              aria-label="Help"
              className="grid size-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <HelpCircle className="size-[1.15rem]" />
            </button>
          </>
        )}
      </div>
    </header>
  );
}
