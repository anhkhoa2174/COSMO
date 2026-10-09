'use client';

import { useCallback, useState } from 'react';
import { useSetAtom } from 'jotai';
import { Check, ChevronDown, Copy, Eye, Send, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { activePanelAtom } from '@/stores/daily-actions';
import { ContactAvatar } from './contact-avatar';
import type { ContactListData, ContactListItem } from '@/types/daily-actions';

const PAGE_SIZE = 10;

const stageBadgeColor: Record<string, string> = {
  COLD: 'bg-indigo-500/[0.12] text-indigo-600 dark:text-indigo-400 border border-indigo-500/[0.25]',
  NO_REPLY: 'bg-orange-500/[0.1] text-orange-600 dark:text-orange-400 border border-orange-500/[0.2]',
  REPLIED: 'bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400 border border-emerald-500/[0.2]',
  POST_MEETING: 'bg-violet-500/[0.1] text-violet-600 dark:text-violet-400 border border-violet-500/[0.2]',
  DROPPED: 'bg-destructive/[0.08] text-destructive border border-destructive/[0.2]',
};

interface ContactListResultProps {
  data: ContactListData;
}

export function ContactListResult({ data }: ContactListResultProps) {
  const [expanded, setExpanded] = useState(true);
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);

  const visibleContacts = data.contacts.slice(0, visibleCount);
  const remaining = data.contacts.length - visibleCount;

  const handleLoadMore = () => {
    setVisibleCount((prev) => Math.min(prev + PAGE_SIZE, data.contacts.length));
  };

  return (
    <div className="overflow-hidden rounded-[10px] border border-border bg-card">
      {/* Header */}
      <button
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2.5 bg-indigo-500/[0.1] px-[14px] py-3 text-left text-[0.85rem] font-extrabold text-indigo-600 dark:text-indigo-400 transition-colors hover:brightness-110"
      >
        {data.icon && <span className="text-[1rem]">{data.icon}</span>}
        <span>{data.title}</span>
        <span className="ml-1 rounded-lg bg-indigo-500/[0.15] px-2 py-0.5 text-[0.72rem] font-extrabold font-mono">
          {data.total_count}
        </span>
        <ChevronDown
          className={cn(
            'ml-auto h-[0.85rem] w-[0.85rem] text-muted-foreground/80 transition-transform',
            !expanded && '-rotate-90'
          )}
        />
      </button>

      {/* Contact rows */}
      <div
        className={cn(
          'overflow-hidden transition-all duration-300',
          expanded ? 'max-h-[10000px]' : 'max-h-0'
        )}
      >
        {visibleContacts.map((contact) => (
          <ContactRow key={contact.id} contact={contact} />
        ))}

        {/* Load more */}
        {remaining > 0 && (
          <div className="border-t border-border p-3">
            <button
              onClick={handleLoadMore}
              className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border p-2 text-xs text-indigo-600 dark:text-indigo-400 hover:bg-muted"
            >
              Load {Math.min(PAGE_SIZE, remaining)} more — {remaining} remaining
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function ContactRow({ contact }: { contact: ContactListItem }) {
  const setActivePanel = useSetAtom(activePanelAtom);
  const [genMsgExpanded, setGenMsgExpanded] = useState(false);
  const [copyState, setCopyState] = useState<'idle' | 'copied'>('idle');
  const [sentState, setSentState] = useState<'idle' | 'sent'>('idle');

  const handleCopy = useCallback(async () => {
    if (!contact.generated_message) return;
    await navigator.clipboard.writeText(contact.generated_message);
    setCopyState('copied');
    setTimeout(() => setCopyState('idle'), 2000);
  }, [contact.generated_message]);

  const handleMarkSent = useCallback(() => {
    setSentState('sent');
  }, []);

  const stageBadge = contact.outreach_stage
    ? stageBadgeColor[contact.outreach_stage] ?? 'bg-muted text-muted-foreground/80 border border-border'
    : null;

  return (
    <div className="border-t border-border">
      {/* Main row */}
      <div className="flex items-center gap-3 px-[14px] py-2.5 transition-colors hover:bg-muted">
        <ContactAvatar name={contact.name} size={30} />

        {/* Info */}
        <div className="min-w-0 flex-1">
          <button
            onClick={() => setActivePanel({ contactId: contact.id })}
            className="border-b-[1.5px] border-dashed border-indigo-500/[0.35] pb-px text-[0.85rem] font-bold text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
          >
            {contact.name}
          </button>
          <p className="truncate text-[0.72rem] text-muted-foreground/80">
            {[contact.job_title, contact.company]
              .filter(Boolean)
              .join(' · ')}
            {contact.days_since_last_interaction != null &&
              ` · Day ${contact.days_since_last_interaction}`}
          </p>
        </div>

        {/* Badges */}
        <div className="hidden items-center gap-1.5 sm:flex">
          {stageBadge && (
            <span
              className={cn(
                'rounded px-[7px] py-[1px] font-mono text-[0.62rem] font-bold uppercase tracking-[0.06em]',
                stageBadge
              )}
            >
              {contact.outreach_stage}
            </span>
          )}
          {contact.status && (
            <span
              className={cn(
                'rounded px-[7px] py-[1px] font-mono text-[0.62rem] font-bold uppercase tracking-[0.06em]',
                contact.status === 'ready'
                  ? 'bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400 border border-emerald-500/[0.2]'
                  : 'bg-amber-500/[0.1] text-amber-600 dark:text-amber-400 border border-amber-500/[0.2]'
              )}
            >
              {contact.status === 'ready' ? 'Approved' : 'Pending'}
            </span>
          )}
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-1">
          <button
            onClick={() => setActivePanel({ contactId: contact.id })}
            className="flex h-7 w-7 items-center justify-center rounded text-muted-foreground/80 hover:bg-muted hover:text-muted-foreground"
            title="Detail"
          >
            <Eye className="h-3.5 w-3.5" />
          </button>
          {contact.generated_message && (
            <button
              onClick={() => setGenMsgExpanded((v) => !v)}
              className={cn(
                'flex h-7 items-center gap-1 rounded px-2 text-[0.68rem] font-bold',
                genMsgExpanded
                  ? 'bg-indigo-600 text-white'
                  : 'text-indigo-600 dark:text-indigo-400 hover:bg-indigo-500/[0.1]'
              )}
              title="Generate / view message"
            >
              <Send className="h-3 w-3" />
              Gen Msg
            </button>
          )}
        </div>
      </div>

      {/* Gen Msg inline expand */}
      {genMsgExpanded && contact.generated_message && (
        <div className="mx-[14px] mb-3 rounded-lg border border-border bg-muted/50 p-4">
          <pre className="whitespace-pre-wrap text-[0.82rem] leading-relaxed text-muted-foreground">
            {contact.generated_message}
          </pre>
          <div className="mt-3 flex items-center gap-2">
            <button
              onClick={handleCopy}
              disabled={copyState === 'copied'}
              className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-[0.72rem] font-bold text-muted-foreground hover:text-foreground disabled:opacity-70"
            >
              {copyState === 'copied' ? (
                <>
                  <Check className="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  Copy
                </>
              )}
            </button>
            <button
              onClick={handleMarkSent}
              disabled={sentState === 'sent'}
              className={cn(
                'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[0.72rem] font-bold disabled:opacity-70',
                sentState === 'sent'
                  ? 'bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400'
                  : 'bg-indigo-600 text-white'
              )}
            >
              {sentState === 'sent' ? (
                <>
                  <Check className="h-3 w-3" />
                  Sent
                </>
              ) : (
                <>
                  <Send className="h-3 w-3" />
                  Mark Sent
                </>
              )}
            </button>
            <button
              onClick={() => setGenMsgExpanded(false)}
              className="ml-auto flex items-center gap-1 rounded-md px-2 py-1.5 text-[0.72rem] text-muted-foreground/80 hover:bg-muted"
            >
              <X className="h-3 w-3" />
              Close
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
