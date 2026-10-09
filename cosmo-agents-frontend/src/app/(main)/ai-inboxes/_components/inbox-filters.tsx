'use client';

import { Search, SlidersHorizontal, X } from 'lucide-react';
import { useEffect, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { cn } from '@/lib/utils';

import {
  GROUP_COLOR_STYLES,
  useConversationGroups,
} from './conversation-groups';
import { INTENT_STYLES } from './intent-badge';

export interface InboxFilters {
  q: string;
  intent: string | null;
  hasDraft: boolean;
  sinceDays: number | null;
  /** One of the caller's own groups. */
  groupId: string | null;
}

export const EMPTY_FILTERS: InboxFilters = {
  q: '',
  intent: null,
  hasDraft: false,
  sinceDays: null,
  groupId: null,
};

export function countActiveFilters(f: InboxFilters): number {
  return (
    (f.intent ? 1 : 0) +
    (f.hasDraft ? 1 : 0) +
    (f.sinceDays ? 1 : 0) +
    (f.groupId ? 1 : 0)
  );
}

const DATE_RANGES: { label: string; days: number | null }[] = [
  { label: 'Any time', days: null },
  { label: 'Last 24 hours', days: 1 },
  { label: 'Last 7 days', days: 7 },
  { label: 'Last 30 days', days: 30 },
];

interface Props {
  value: InboxFilters;
  onChange: (next: InboxFilters) => void;
}

/**
 * Search box and filter menu for the inbox list.
 *
 * The search term is debounced before it leaves this component: the list
 * refetches on every change, and firing a query per keystroke made typing feel
 * like it was fighting the results.
 */
export function InboxFilters({ value, onChange }: Props) {
  const [draft, setDraft] = useState(value.q);
  const active = countActiveFilters(value);
  const { data: groups = [], isSuccess: groupsLoaded } =
    useConversationGroups();

  useEffect(() => setDraft(value.q), [value.q]);

  // Deleting the group being filtered on left the filter pointing at nothing:
  // an empty list, a badge counting it, and no button to turn it off.
  useEffect(() => {
    if (
      groupsLoaded &&
      value.groupId &&
      !groups.some((g) => g.id === value.groupId)
    ) {
      onChange({ ...value, groupId: null });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groupsLoaded, groups, value.groupId]);

  useEffect(() => {
    if (draft === value.q) return;
    const t = setTimeout(() => onChange({ ...value, q: draft }), 300);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft]);

  const set = (patch: Partial<InboxFilters>) =>
    onChange({ ...value, ...patch });

  return (
    <div className="flex items-center gap-2 border-b bg-background px-3 py-2">
      <div className="relative min-w-0 flex-1">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              setDraft('');
              onChange({ ...value, q: '' });
            }
          }}
          placeholder="Search sender, subject, message…"
          aria-label="Search conversations"
          className="h-8 pl-8 pr-7 text-sm"
        />
        {draft && (
          <button
            type="button"
            aria-label="Clear search"
            onClick={() => {
              setDraft('');
              onChange({ ...value, q: '' });
            }}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded text-muted-foreground hover:text-foreground"
          >
            <X className="size-3.5" />
          </button>
        )}
      </div>

      <Popover>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            aria-label="Filters"
            className={cn(
              'h-8 shrink-0 gap-1.5 px-2.5',
              active && 'border-primary'
            )}
          >
            <SlidersHorizontal className="size-3.5" />
            {active > 0 && (
              <span className="grid size-4 place-items-center rounded-full bg-primary text-[0.6rem] font-semibold text-primary-foreground">
                {active}
              </span>
            )}
          </Button>
        </PopoverTrigger>

        <PopoverContent align="end" className="w-60 p-3">
          <div className="space-y-3">
            {groups.length > 0 && (
              <div>
                <p className="mb-1.5 text-xs font-medium">Group</p>
                <div className="flex flex-wrap gap-1">
                  {groups.map((g) => (
                    <button
                      key={g.id}
                      type="button"
                      onClick={() =>
                        set({ groupId: value.groupId === g.id ? null : g.id })
                      }
                      className={cn(
                        'inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[0.7rem] transition-colors',
                        value.groupId === g.id
                          ? 'border-primary bg-primary/10 text-foreground'
                          : 'text-muted-foreground hover:bg-accent'
                      )}
                    >
                      <span
                        className={cn(
                          'size-1.5 rounded-full',
                          (
                            GROUP_COLOR_STYLES[g.color] ??
                            GROUP_COLOR_STYLES.slate
                          ).dot
                        )}
                      />
                      {g.name}
                    </button>
                  ))}
                </div>
              </div>
            )}

            <div>
              <p className="mb-1.5 text-xs font-medium">Intent</p>
              <div className="flex flex-wrap gap-1">
                {Object.entries(INTENT_STYLES).map(([key, style]) => (
                  <button
                    key={key}
                    type="button"
                    onClick={() =>
                      set({ intent: value.intent === key ? null : key })
                    }
                    className={cn(
                      'rounded-md border px-1.5 py-0.5 text-[0.7rem] transition-colors',
                      value.intent === key
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'text-muted-foreground hover:bg-accent'
                    )}
                  >
                    {style.label}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <p className="mb-1.5 text-xs font-medium">Date</p>
              <div className="flex flex-wrap gap-1">
                {DATE_RANGES.map((r) => (
                  <button
                    key={r.label}
                    type="button"
                    onClick={() => set({ sinceDays: r.days })}
                    className={cn(
                      'rounded-md border px-1.5 py-0.5 text-[0.7rem] transition-colors',
                      value.sinceDays === r.days
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'text-muted-foreground hover:bg-accent'
                    )}
                  >
                    {r.label}
                  </button>
                ))}
              </div>
            </div>

            <label className="flex cursor-pointer items-center gap-2 text-xs">
              <input
                type="checkbox"
                checked={value.hasDraft}
                onChange={(e) => set({ hasDraft: e.target.checked })}
                className="size-3.5 accent-primary"
              />
              Only threads with an AI draft waiting
            </label>

            {active > 0 && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-full text-xs"
                onClick={() => onChange({ ...EMPTY_FILTERS, q: value.q })}
              >
                Clear filters
              </Button>
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
