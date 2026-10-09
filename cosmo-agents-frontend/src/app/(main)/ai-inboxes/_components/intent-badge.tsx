import { cn } from '@/lib/utils';

export interface IntentStyle {
  label: string;
  /** Classes for the colored chip (background + text). */
  chipClass: string;
  /** Text-only color, used to highlight the intent sentence in the email body. */
  textClass: string;
}

/**
 * Canonical intent → visual style. Keys are SCREAMING_SNAKE_CASE; the backend
 * stores display strings like "Interested" / "Out of office", so always go
 * through normalizeIntent()/getIntentStyle() rather than indexing directly.
 */
export const INTENT_STYLES: Record<string, IntentStyle> = {
  INTERESTED: {
    label: 'Interested',
    chipClass:
      'bg-violet-100 text-violet-700 dark:bg-violet-950 dark:text-violet-300',
    textClass: 'text-violet-700 dark:text-violet-300',
  },
  NOT_INTERESTED: {
    label: 'Not interested',
    chipClass: 'bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300',
    textClass: 'text-rose-700 dark:text-rose-300',
  },
  REQUEST_FOR_PRICING: {
    label: 'Request for pricing',
    chipClass: 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300',
    textClass: 'text-blue-700 dark:text-blue-300',
  },
  REQUEST_FOR_INFORMATION: {
    label: 'Request for information',
    chipClass:
      'bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300',
    textClass: 'text-amber-700 dark:text-amber-300',
  },
  REFERRAL: {
    label: 'Referral',
    chipClass: 'bg-teal-100 text-teal-700 dark:bg-teal-950 dark:text-teal-300',
    textClass: 'text-teal-700 dark:text-teal-300',
  },
  NURTURE: {
    label: 'Nurture',
    chipClass:
      'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300',
    textClass: 'text-emerald-700 dark:text-emerald-300',
  },
  OUT_OF_OFFICE: {
    label: 'Out of office',
    chipClass:
      'bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300',
    textClass: 'text-orange-700 dark:text-orange-300',
  },
  DO_NOT_CONTACT: {
    label: 'Do not contact',
    chipClass: 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300',
    textClass: 'text-red-700 dark:text-red-300',
  },
  OTHER: {
    label: 'Other',
    chipClass: 'bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300',
    textClass: 'text-zinc-700 dark:text-zinc-300',
  },
};

/**
 * Maps any stored intent value ("Interested", "OUT_OF_OFFICE", "out of
 * office", …) to a canonical INTENT_STYLES key, falling back to OTHER.
 */
/**
 * The backend stores display strings ("Do not contact", "Unknown intent"), so
 * normalisation upper-cases and underscores them. ALIASES cover the labels
 * whose wording differs from the canonical key.
 */
const ALIASES: Record<string, string> = {
  UNKNOWN_INTENT: 'OTHER',
  UNKNOWN: 'OTHER',
};

export function normalizeIntent(intent: string | null | undefined): string {
  if (!intent) return 'OTHER';
  const key = intent.trim().toUpperCase().replace(/[\s-]+/g, '_');
  if (INTENT_STYLES[key]) return key;
  if (ALIASES[key]) return ALIASES[key];
  return 'OTHER';
}

export function getIntentStyle(intent: string | null | undefined): IntentStyle {
  return INTENT_STYLES[normalizeIntent(intent)];
}

interface IntentBadgeProps {
  intent: string;
  size?: 'sm' | 'md';
  className?: string;
}

export function IntentBadge({ intent, size = 'sm', className }: IntentBadgeProps) {
  const key = normalizeIntent(intent);
  const style = INTENT_STYLES[key];
  // Unknown values fall back to zinc but keep their own text so nothing is
  // silently relabeled "Other" (e.g. legacy "Unknown intent").
  const label = key === 'OTHER' && intent?.trim() ? intent.trim() : style.label;

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center whitespace-nowrap rounded-full font-medium',
        size === 'sm' ? 'px-2 py-0.5 text-[11px]' : 'px-2.5 py-0.5 text-xs',
        style.chipClass,
        className
      )}
    >
      {label}
    </span>
  );
}
