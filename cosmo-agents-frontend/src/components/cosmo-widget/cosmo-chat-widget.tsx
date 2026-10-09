'use client';

/**
 * Ask COSMO — floating chat widget.
 *
 * Launcher sits bottom-right at z-40: above page content (sidebar is z-10/z-20)
 * but below Radix dialogs/sheets (z-50) and sonner toasts, so nothing important
 * ever gets covered.
 *
 * Below the `sm` breakpoint the panel becomes a full-screen sheet.
 */

import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { usePathname } from 'next/navigation';
import {
  AlertTriangle,
  ArrowUp,
  History,
  Paperclip,
  Plus,
  RotateCcw,
  Square,
  Trash2,
  X,
} from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import { cn } from '@/lib/utils';
import {
  ATTACHMENT_ACCEPT,
  composeMessage,
  readAttachment,
  type Attachment,
} from '@/lib/import/attachment';

import { useCosmoChat } from './cosmo-chat-provider';
import { CosmoMessage } from './cosmo-message';

/**
 * Every suggestion below maps onto tools that really exist in
 * cosmo-agents-sdk / src/lib/ai/bd-agent-tools.ts:
 *   create_campaign + list_campaigns, get_pipeline_summary,
 *   search_contacts / hybrid_search_contacts, suggest_outreach +
 *   generate_outreach_draft.
 */
const EXAMPLE_PROMPTS = [
  'Give me a pipeline summary',
  'Create a campaign to warm up my cold leads',
  'Search my contacts for fintech founders',
  'Suggest outreach and draft a follow-up for my top prospect',
] as const;

function useIsMac() {
  const [isMac, setIsMac] = useState(false);
  useEffect(() => {
    setIsMac(
      /Mac|iPhone|iPad/i.test(navigator.platform || navigator.userAgent)
    );
  }, []);
  return isMac;
}

export function CosmoChatWidget() {
  const [showHistory, setShowHistory] = useState(false);

  const {
    isOpen,
    open,
    close,
    reset,
    messages,
    input,
    setInput,
    append,
    reload,
    stop,
    isLoading,
    error,
    sessions,
    activeSessionId,
    openSession,
    deleteSession,
  } = useCosmoChat();

  const isMac = useIsMac();
  const pathname = usePathname();
  // Daily Actions owns the bottom strip with its own composer — lift the
  // launcher above it so the Send button is never covered.
  const liftLauncher = pathname?.startsWith('/daily-actions') ?? false;
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const [attachment, setAttachment] = useState<Attachment | null>(null);
  const [reading, setReading] = useState(false);

  // Escape closes the panel (unless a modal is on top of it).
  useEffect(() => {
    if (!isOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      if (document.querySelector('[role="dialog"],[role="alertdialog"]'))
        return;
      close();
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [isOpen, close]);

  useEffect(() => {
    if (isOpen) inputRef.current?.focus();
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    bottomRef.current?.scrollIntoView({ block: 'end' });
  }, [isOpen, messages, isLoading, error]);

  const send = () => {
    const text = input.trim();
    // A file on its own is a valid message: "here, look at this".
    if ((!text && !attachment) || isLoading) return;
    setInput('');
    setAttachment(null);
    void append({
      role: 'user',
      content: composeMessage(text || 'Please look at this file.', attachment),
    });
  };

  const pickFile = async (file: File | undefined) => {
    if (!file) return;
    setReading(true);
    try {
      setAttachment(await readAttachment(file));
    } catch (e: any) {
      toast.error(e?.message ?? 'Could not read that file');
    } finally {
      setReading(false);
      // Cleared so choosing the same file twice in a row still fires onChange.
      if (fileRef.current) fileRef.current.value = '';
    }
  };

  if (!isOpen) {
    return (
      <button
        type="button"
        onClick={open}
        aria-label="Open Ask COSMO"
        title={`Ask COSMO (${isMac ? '⌘' : 'Ctrl'}+/)`}
        className={cn(
          // Pill that expands on hover: the label tells a first-time user what
          // the button does, while the collapsed state stays out of the way.
          'group fixed right-5 z-40 flex h-14 items-center gap-2.5 overflow-hidden rounded-full bg-gradient-to-br from-violet-600 to-indigo-700 pl-3.5 pr-3.5 text-white shadow-xl shadow-violet-600/30 ring-1 ring-white/20 transition-all duration-300 hover:pr-5 hover:shadow-2xl hover:shadow-violet-600/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-400 focus-visible:ring-offset-2',
          liftLauncher ? 'bottom-[6.5rem]' : 'bottom-5'
        )}
      >
        <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-white shadow-sm">
          <CosmoMark size={22} id="cosmo-widget-launcher" />
        </span>
        <span className="max-w-0 whitespace-nowrap text-sm font-semibold opacity-0 transition-all duration-300 group-hover:max-w-[7rem] group-hover:opacity-100">
          Ask COSMO
        </span>
      </button>
    );
  }

  return (
    <div
      role="complementary"
      aria-label="Ask COSMO"
      className={cn(
        // mobile: full-screen sheet
        'fixed inset-0 z-40 flex flex-col overflow-hidden bg-background',
        // sm+: anchored floating panel
        'sm:inset-auto sm:bottom-5 sm:right-5 sm:h-[600px] sm:max-h-[calc(100dvh-2.5rem)] sm:w-[400px] sm:resize sm:overflow-hidden sm:rounded-xl sm:border sm:shadow-2xl sm:shadow-violet-950/10 dark:sm:shadow-black/40'
      )}
    >
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2.5 border-b bg-gradient-to-r from-violet-500 to-indigo-600 px-3.5 py-3 text-white">
        <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-white shadow-sm">
          <CosmoMark size={20} id="cosmo-widget-header" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold leading-tight">
            Ask COSMO
          </p>
          <p className="truncate text-[0.7rem] leading-tight text-white/70">
            Your BD agent — it can act, not just answer
          </p>
        </div>
        {sessions.length > 0 && (
          <button
            type="button"
            onClick={() => setShowHistory((v) => !v)}
            title="Past conversations"
            aria-label="Past conversations"
            aria-expanded={showHistory}
            className="grid size-7 shrink-0 place-items-center rounded-md text-white/80 transition-colors hover:bg-white/15 hover:text-white"
          >
            <History className="size-4" />
          </button>
        )}
        {messages.length > 0 && (
          <button
            type="button"
            onClick={reset}
            title="New conversation"
            aria-label="New conversation"
            className="grid size-7 shrink-0 place-items-center rounded-md text-white/80 transition-colors hover:bg-white/15 hover:text-white"
          >
            {/* A plus, not a circular arrow. Beside the history icon a ↺ reads
                as "reload this conversation", which is close enough to the
                opposite of what it does that people did not find it. */}
            <Plus className="size-4" />
          </button>
        )}
        <button
          type="button"
          onClick={close}
          title="Close (Esc)"
          aria-label="Close Ask COSMO"
          className="grid size-7 shrink-0 place-items-center rounded-md text-white/80 transition-colors hover:bg-white/15 hover:text-white"
        >
          <X className="size-4" />
        </button>
      </div>

      {showHistory && (
        <div className="max-h-56 shrink-0 overflow-y-auto border-b bg-muted/30">
          <p className="px-3.5 pt-2.5 text-[0.7rem] font-medium uppercase tracking-wide text-muted-foreground">
            Past conversations
          </p>
          <ul className="p-1.5">
            {sessions.map((session) => (
              <li key={session.id} className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => {
                    void openSession(session.id);
                    setShowHistory(false);
                  }}
                  className={cn(
                    'min-w-0 flex-1 truncate rounded-md px-2 py-1.5 text-left text-xs transition-colors hover:bg-accent',
                    session.id === activeSessionId &&
                      'bg-violet-500/10 font-medium text-violet-700 dark:text-violet-300'
                  )}
                >
                  {session.title || 'Untitled conversation'}
                </button>
                <button
                  type="button"
                  onClick={() => void deleteSession(session.id)}
                  title="Delete conversation"
                  aria-label={`Delete ${session.title}`}
                  className="grid size-7 shrink-0 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Thread */}
      <div className="cosmo-thin-scrollbar min-h-0 flex-1 space-y-3 overflow-y-auto px-3.5 py-4">
        {messages.length === 0 && !error && (
          <div className="flex h-full flex-col justify-center gap-4 px-1">
            <div className="flex flex-col items-center gap-2 text-center">
              <span className="grid size-12 place-items-center rounded-xl border border-border bg-white shadow-sm">
                <CosmoMark size={26} id="cosmo-widget-empty" />
              </span>
              <p className="text-sm font-semibold text-foreground">
                What should COSMO do next?
              </p>
              <p className="text-xs text-muted-foreground">
                Ask in plain language — COSMO uses your CRM tools to get it
                done.
              </p>
            </div>
            <div className="flex flex-col gap-1.5">
              {EXAMPLE_PROMPTS.map((prompt) => (
                <button
                  key={prompt}
                  type="button"
                  onClick={() => {
                    setInput(prompt);
                    inputRef.current?.focus();
                  }}
                  className="rounded-xl border bg-card px-3 py-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:border-violet-300 hover:bg-violet-500/10 hover:text-violet-700 dark:hover:border-violet-700 dark:hover:text-violet-300"
                >
                  {prompt}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map((message) => (
          <CosmoMessage key={message.id} message={message} />
        ))}

        {isLoading && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="flex gap-1">
              <span className="size-1.5 animate-bounce rounded-full bg-violet-500 [animation-delay:-0.3s] dark:bg-violet-400" />
              <span className="size-1.5 animate-bounce rounded-full bg-violet-500 [animation-delay:-0.15s] dark:bg-violet-400" />
              <span className="size-1.5 animate-bounce rounded-full bg-violet-500 dark:bg-violet-400" />
            </span>
            COSMO is working…
          </div>
        )}

        {error && (
          <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-3">
            <div className="flex items-start gap-2">
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-destructive" />
              <div className="min-w-0 flex-1">
                <p className="text-xs font-semibold text-destructive">
                  Request failed
                </p>
                <p className="mt-0.5 break-words text-xs text-muted-foreground">
                  {error.message || 'Unknown error'}
                </p>
              </div>
            </div>
            <Button
              size="sm"
              variant="outline"
              className="mt-2 h-7 w-full text-xs"
              onClick={() => void reload()}
            >
              <RotateCcw className="size-3" />
              Retry
            </Button>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Composer */}
      <div className="shrink-0 border-t bg-card px-3 py-2.5">
        {attachment && (
          <div className="mb-1.5 flex items-center gap-2 rounded-lg border bg-muted/50 px-2.5 py-1.5 text-xs">
            <Paperclip className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="min-w-0 flex-1 truncate font-medium">
              {attachment.name}
            </span>
            {/* Truncation is stated here, not only inside the message, so the
                limit is visible before sending rather than inferred from a
                partial answer afterwards. */}
            {attachment.truncated && (
              <span className="shrink-0 text-[0.7rem] text-amber-600">
                first {(attachment.text.length / 1000).toFixed(0)}k chars
              </span>
            )}
            <button
              type="button"
              onClick={() => setAttachment(null)}
              aria-label="Remove attachment"
              className="shrink-0 rounded text-muted-foreground hover:text-foreground"
            >
              <X className="size-3.5" />
            </button>
          </div>
        )}
        <input
          ref={fileRef}
          type="file"
          accept={ATTACHMENT_ACCEPT}
          hidden
          onChange={(e) => void pickFile(e.target.files?.[0])}
        />
        <div className="flex items-end gap-2 rounded-xl border bg-background px-2.5 py-1.5 focus-within:border-violet-400 focus-within:ring-1 focus-within:ring-violet-400/40">
          <Button
            size="icon"
            variant="ghost"
            className="size-8 shrink-0 rounded-lg"
            onClick={() => fileRef.current?.click()}
            disabled={reading || isLoading}
            title="Attach a text file or spreadsheet"
            aria-label="Attach a file"
          >
            <Paperclip className="size-4" />
          </Button>
          <Textarea
            ref={inputRef}
            value={input}
            onChange={(event) => setInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                send();
              }
            }}
            rows={1}
            placeholder="Ask COSMO to do something…"
            className="max-h-32 min-h-[32px] resize-none border-0 bg-transparent p-0 py-1 text-sm shadow-none focus-visible:ring-0"
          />
          {isLoading ? (
            <Button
              size="icon"
              variant="ghost"
              className="size-8 shrink-0 rounded-lg"
              onClick={() => stop()}
              title="Stop"
              aria-label="Stop generating"
            >
              <Square className="size-3.5 fill-current" />
            </Button>
          ) : (
            <Button
              size="icon"
              className="size-8 shrink-0 rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 hover:opacity-90"
              onClick={send}
              disabled={!input.trim() && !attachment}
              title="Send"
              aria-label="Send message"
            >
              <ArrowUp className="size-4" />
            </Button>
          )}
        </div>
        <p className="mt-1.5 text-center text-[0.65rem] text-muted-foreground">
          {isMac ? '⌘' : 'Ctrl'}+/ to toggle · Esc to close
        </p>
      </div>
    </div>
  );
}
