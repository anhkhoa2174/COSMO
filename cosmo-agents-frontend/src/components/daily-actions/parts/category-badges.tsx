'use client';

import { useAtom } from 'jotai';
import type { CategoryBadge, CategoryId } from '@/types/daily-actions';
import { expandedCategoriesAtom } from '@/stores/daily-actions';
import { cn } from '@/lib/utils';

const emojiMap: Record<string, string> = {
  replied: '💬',
  followup: '🔄',
  new_outreach: '🚀',
  meeting_prep: '📋',
  enrichment: '⚠️',
  outreach: '🚀',
  meeting: '📋',
  blocked: '⚠️',
  enrich: '🔍',
  respond: '💬',
};

const badgeStyleMap: Record<
  string,
  { bg: string; text: string; border: string }
> = {
  new_outreach: { bg: 'bg-indigo-500/[0.1]', text: 'text-indigo-600 dark:text-indigo-400', border: 'border-indigo-500/[0.25]' },
  outreach: { bg: 'bg-indigo-500/[0.1]', text: 'text-indigo-600 dark:text-indigo-400', border: 'border-indigo-500/[0.25]' },
  followup: { bg: 'bg-orange-500/[0.1]', text: 'text-orange-600 dark:text-orange-400', border: 'border-orange-500/[0.2]' },
  replied: { bg: 'bg-emerald-500/[0.1]', text: 'text-emerald-600 dark:text-emerald-400', border: 'border-emerald-500/[0.2]' },
  respond: { bg: 'bg-emerald-500/[0.1]', text: 'text-emerald-600 dark:text-emerald-400', border: 'border-emerald-500/[0.2]' },
  meeting_prep: { bg: 'bg-violet-500/[0.1]', text: 'text-violet-600 dark:text-violet-400', border: 'border-violet-500/[0.2]' },
  meeting: { bg: 'bg-violet-500/[0.1]', text: 'text-violet-600 dark:text-violet-400', border: 'border-violet-500/[0.2]' },
  enrichment: { bg: 'bg-amber-500/[0.1]', text: 'text-amber-600 dark:text-amber-400', border: 'border-amber-500/[0.2]' },
  blocked: { bg: 'bg-amber-500/[0.1]', text: 'text-amber-600 dark:text-amber-400', border: 'border-amber-500/[0.2]' },
  enrich: { bg: 'bg-amber-500/[0.1]', text: 'text-amber-600 dark:text-amber-400', border: 'border-amber-500/[0.2]' },
};

interface CategoryBadgesProps {
  badges: CategoryBadge[];
}

export function CategoryBadges({ badges }: CategoryBadgesProps) {
  const [, setExpandedCategories] = useAtom(expandedCategoriesAtom);

  const handleBadgeClick = (categoryId: string) => {
    setExpandedCategories((prev) => {
      const next = new Set<CategoryId>(prev);
      next.add(categoryId as CategoryId);
      return next;
    });

    setTimeout(() => {
      const sectionEl = document.getElementById(`category-${categoryId}`);
      if (sectionEl) {
        sectionEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }, 50);
  };

  return (
    <div className="my-3.5 flex flex-wrap gap-2">
      {badges.map((badge) => {
        const emoji = emojiMap[badge.category] || '📋';
        const styles = badgeStyleMap[badge.category] || badgeStyleMap.outreach;

        return (
          <button
            key={badge.category}
            onClick={() => handleBadgeClick(badge.category)}
            className={cn(
              'cursor-pointer rounded-md border px-3 py-1.5 text-[0.8rem] font-bold font-mono transition-all hover:-translate-y-px',
              styles.bg,
              styles.text,
              styles.border
            )}
          >
            {emoji} {badge.count} {badge.label}
          </button>
        );
      })}
    </div>
  );
}
