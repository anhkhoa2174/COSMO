'use client';

import { useCallback, useMemo, useState } from 'react';
import { useAtom } from 'jotai';
import { ChevronDown, Loader2 } from 'lucide-react';
import type {
  ActionCategory,
  CategoryId,
  DailyAction,
} from '@/types/daily-actions';
import { expandedCategoriesAtom } from '@/stores/daily-actions';
import { cn } from '@/lib/utils';
import DailyActionsApi from '@/network/client/daily-actions';
import { ActionCard } from './action-card';

/** FR-013: Consolidate multiple actions for the same contact into a single card. */
function consolidateByContact(actions: DailyAction[]) {
  const grouped = new Map<
    string,
    { primary: DailyAction; secondary: DailyAction[] }
  >();

  for (const action of actions) {
    const key = action.contact.id;
    const existing = grouped.get(key);
    if (!existing) {
      grouped.set(key, { primary: action, secondary: [] });
    } else {
      if (action.priority < existing.primary.priority) {
        existing.secondary.push(existing.primary);
        existing.primary = action;
      } else {
        existing.secondary.push(action);
      }
    }
  }

  return Array.from(grouped.values());
}

const emojiMap: Record<string, string> = {
  replied: '💬',
  followup: '⏰',
  new_outreach: '📤',
  meeting_prep: '📅',
  enrichment: '⚠️',
  outreach: '📤',
  meeting: '📅',
  blocked: '⚠️',
  enrich: '🔍',
  respond: '💬',
};

const headerStyleMap: Record<
  string,
  { bg: string; text: string; border: string; countBg: string }
> = {
  replied: {
    bg: 'bg-emerald-500/[0.1]',
    text: 'text-emerald-600 dark:text-emerald-400',
    border: 'border-emerald-500/[0.2]',
    countBg: 'bg-emerald-500/[0.15]',
  },
  new_outreach: {
    bg: 'bg-indigo-500/[0.1]',
    text: 'text-indigo-600 dark:text-indigo-400',
    border: 'border-indigo-500/[0.25]',
    countBg: 'bg-indigo-500/[0.15]',
  },
  outreach: {
    bg: 'bg-indigo-500/[0.1]',
    text: 'text-indigo-600 dark:text-indigo-400',
    border: 'border-indigo-500/[0.25]',
    countBg: 'bg-indigo-500/[0.15]',
  },
  followup: {
    bg: 'bg-orange-500/[0.1]',
    text: 'text-orange-600 dark:text-orange-400',
    border: 'border-orange-500/[0.2]',
    countBg: 'bg-orange-500/[0.15]',
  },
  meeting_prep: {
    bg: 'bg-violet-500/[0.1]',
    text: 'text-violet-600 dark:text-violet-400',
    border: 'border-violet-500/[0.2]',
    countBg: 'bg-violet-500/[0.15]',
  },
  meeting: {
    bg: 'bg-violet-500/[0.1]',
    text: 'text-violet-600 dark:text-violet-400',
    border: 'border-violet-500/[0.2]',
    countBg: 'bg-violet-500/[0.15]',
  },
  enrichment: {
    bg: 'bg-amber-500/[0.1]',
    text: 'text-amber-600 dark:text-amber-400',
    border: 'border-amber-500/[0.2]',
    countBg: 'bg-amber-500/[0.15]',
  },
  blocked: {
    bg: 'bg-amber-500/[0.1]',
    text: 'text-amber-600 dark:text-amber-400',
    border: 'border-amber-500/[0.2]',
    countBg: 'bg-amber-500/[0.15]',
  },
  enrich: {
    bg: 'bg-amber-500/[0.1]',
    text: 'text-amber-600 dark:text-amber-400',
    border: 'border-amber-500/[0.2]',
    countBg: 'bg-amber-500/[0.15]',
  },
  respond: {
    bg: 'bg-emerald-500/[0.1]',
    text: 'text-emerald-600 dark:text-emerald-400',
    border: 'border-emerald-500/[0.2]',
    countBg: 'bg-emerald-500/[0.15]',
  },
};

interface ActionCategorySectionProps {
  category: ActionCategory;
}

export function ActionCategorySection({
  category,
}: ActionCategorySectionProps) {
  const [expandedCategories, setExpandedCategories] = useAtom(
    expandedCategoriesAtom
  );
  const [extraActions, setExtraActions] = useState<DailyAction[]>([]);
  const [hasMore, setHasMore] = useState(category.has_more);
  const [nextOffset, setNextOffset] = useState(category.next_offset);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const isExpanded = expandedCategories.has(category.id);
  const emoji = emojiMap[category.id] || '📋';
  const styles = headerStyleMap[category.id] || headerStyleMap.outreach;

  const allActions = useMemo(
    () => [...category.actions, ...extraActions],
    [category.actions, extraActions]
  );

  const consolidated = useMemo(
    () => consolidateByContact(allActions),
    [allActions]
  );

  const handleLoadMore = useCallback(async () => {
    if (isLoadingMore || !hasMore) return;
    setIsLoadingMore(true);
    try {
      const offset = nextOffset ?? category.actions.length;
      const response = await DailyActionsApi.loadMoreActions(
        category.id,
        offset
      );
      const result = response.data;
      setExtraActions((prev) => [...prev, ...result.actions]);
      setHasMore(result.has_more);
      setNextOffset(result.next_offset);
    } catch {
      // Silently fail — button remains visible for retry
    } finally {
      setIsLoadingMore(false);
    }
  }, [
    isLoadingMore,
    hasMore,
    nextOffset,
    category.actions.length,
    category.id,
  ]);

  const toggleExpanded = () => {
    setExpandedCategories((prev) => {
      const next = new Set<CategoryId>(prev);
      if (next.has(category.id)) {
        next.delete(category.id);
      } else {
        next.add(category.id);
      }
      return next;
    });
  };

  const remaining = category.total_count - allActions.length;

  return (
    <div id={`category-${category.id}`} className="mt-4 overflow-hidden rounded-[10px] border border-border bg-card">
      {/* Colored header */}
      <button
        onClick={toggleExpanded}
        className={cn(
          'flex w-full items-center gap-2.5 border-b px-[14px] py-3 text-left text-[0.85rem] font-extrabold select-none transition-[filter] hover:brightness-110',
          styles.bg,
          styles.text,
          styles.border
        )}
      >
        <span className="text-[1rem]">{emoji}</span>
        <span>{category.label}</span>
        <span
          className={cn(
            'ml-1 rounded-lg px-2 py-0.5 text-[0.72rem] font-extrabold font-mono',
            styles.countBg
          )}
        >
          {category.total_count}
        </span>
        <ChevronDown
          className={cn(
            'ml-auto h-[0.85rem] w-[0.85rem] transition-transform',
            !isExpanded && '-rotate-90'
          )}
        />
      </button>

      {/* Category body */}
      <div
        className={cn(
          'overflow-hidden transition-all duration-300',
          isExpanded ? 'max-h-[20000px]' : 'max-h-0'
        )}
      >
        <div>
          {consolidated.map(({ primary, secondary }) => (
            <ActionCard
              key={primary.id}
              action={primary}
              secondaryActions={secondary.length > 0 ? secondary : undefined}
            />
          ))}
        </div>
        {hasMore && remaining > 0 && (
          <div className="border-t border-border p-3">
            <button
              onClick={handleLoadMore}
              disabled={isLoadingMore}
              className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border p-2 text-xs text-indigo-600 dark:text-indigo-400 hover:bg-muted disabled:opacity-50"
            >
              {isLoadingMore ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin" />
                  Loading...
                </>
              ) : (
                `Load more (${remaining} remaining)`
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
