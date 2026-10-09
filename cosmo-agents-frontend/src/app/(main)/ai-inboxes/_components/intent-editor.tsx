'use client';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import type { ConversationIntentCorrection } from '@/models/conversation';
import ConversationApi from '@/network/client/conversation';
import { useMutation } from '@tanstack/react-query';
import { Check, ChevronDown, Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { INTENT_STYLES, IntentBadge, normalizeIntent } from './intent-badge';

/**
 * The labels a human may pick, in the order the classifier's taxonomy defines
 * them. Sent as the backend's display strings so the value the rep chose is
 * the value that gets stored.
 */
const SELECTABLE_INTENTS = [
  'Interested',
  'Request for pricing',
  'Request for information',
  'Referral',
  'Nurture',
  'Out of office',
  'Not interested',
  'Do not contact',
  'Unknown intent',
];

interface IntentEditorProps {
  conversationId: string;
  /** The label currently stored for this reply. */
  intent: string;
  /** True once a human has already overridden the classifier here. */
  corrected?: boolean;
  size?: 'sm' | 'md';
  /** Called after a successful change so the thread can refresh. */
  onCorrected?: (result: ConversationIntentCorrection) => void;
  /**
   * Invoked when the correction leaves an AI draft that was written for the
   * previous intent. The draft is never rewritten automatically — it may hold
   * the rep's own edits — so the caller decides what to offer.
   */
  onDraftStale?: () => void;
}

/**
 * The intent badge, made editable.
 *
 * The classifier reads short or context-free replies badly, and its label is
 * not cosmetic: it chose the handler that ran and, for "Do not contact", it
 * suppressed the contact. So the rep needs a way to overrule it, and the
 * override has to travel to the backend rather than just repaint the chip.
 */
export function IntentEditor({
  conversationId,
  intent,
  corrected,
  size = 'md',
  onCorrected,
  onDraftStale,
}: IntentEditorProps) {
  const currentKey = normalizeIntent(intent);

  const mutation = useMutation({
    mutationFn: (next: string) =>
      ConversationApi.correctIntent(conversationId, { intent: next }),
    onSuccess: ({ data }) => {
      if (!data.changed) return;

      // Say what actually happened to the contact. Silently lifting a
      // suppression is exactly the kind of change a rep must not discover
      // later from a send they did not expect.
      const suppression =
        data.suppression_changed === 'unsuppressed'
          ? ' The contact is no longer suppressed.'
          : data.suppression_changed === 'suppressed'
            ? ' The contact is now suppressed and will not be emailed again.'
            : '';
      toast.success(`Intent set to "${data.intent}".${suppression}`);

      onCorrected?.(data);
      if (data.draft_stale) onDraftStale?.();
    },
    onError: (err: any) => {
      toast.error(
        err?.error?.message || err?.message || 'Could not update the intent'
      );
    },
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild disabled={mutation.isPending}>
        <button
          type="button"
          className={cn(
            'inline-flex items-center gap-1 rounded-full transition-opacity',
            'hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            mutation.isPending && 'cursor-wait opacity-60'
          )}
          title={
            corrected
              ? 'Intent corrected by a person. Click to change.'
              : 'Classified by AI. Click to correct.'
          }
        >
          <IntentBadge intent={intent} size={size} />
          {mutation.isPending ? (
            <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
          ) : (
            <ChevronDown className="h-3 w-3 text-muted-foreground" />
          )}
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
          {corrected ? 'Corrected by a person' : 'Classified by AI'}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {SELECTABLE_INTENTS.map((label) => {
          const key = normalizeIntent(label);
          const isCurrent = key === currentKey;
          return (
            <DropdownMenuItem
              key={label}
              disabled={isCurrent || mutation.isPending}
              onSelect={() => mutation.mutate(label)}
              className="flex items-center justify-between gap-2"
            >
              <span
                className={cn(
                  'text-sm',
                  INTENT_STYLES[key]?.textClass ?? 'text-foreground'
                )}
              >
                {label}
              </span>
              {isCurrent && <Check className="h-3.5 w-3.5 shrink-0" />}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
