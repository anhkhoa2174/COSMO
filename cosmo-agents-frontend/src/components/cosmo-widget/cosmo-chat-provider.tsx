'use client';

/**
 * Ask COSMO — global chat widget provider.
 *
 * Mounted once in the authenticated `(main)` layout so both the open/closed
 * state and the whole `useChat` message thread survive client-side navigation
 * (Next.js keeps layouts mounted while only the page segment swaps).
 *
 * It deliberately points at `/api/ai/daily-actions` — the exact same route the
 * Daily Actions page uses — so the widget gets the identical BD Agent system
 * prompt and the full cosmo-agents-sdk tool registry with zero duplication.
 *
 * Conversations are persisted server-side, per user. The thread used to live
 * only in this component's state, so a refresh lost it; now the most recent
 * conversation is restored on mount and every completed turn is appended.
 *
 * The write happens here rather than in the API route because that route is
 * shared with the Daily Actions page: persisting there would file that page's
 * chat into Ask COSMO's history too. The cost is that a turn abandoned
 * mid-stream is not stored, which is the right trade — a half-finished answer
 * is not worth restoring.
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PropsWithChildren,
} from 'react';
import { useChat, type UseChatHelpers } from 'ai/react';

import ChatSessionsApi, {
  type ChatSession,
} from '@/network/client/chat-sessions';

/** Ctrl+/ (Cmd+/ on mac). Ctrl+K is already taken by the Daily Actions chat
 *  input focus shortcut and Ctrl+B by the sidebar toggle, so `/` is the free
 *  slot that is also not reserved by Chrome/Firefox/Safari. */
export const COSMO_WIDGET_SHORTCUT_LABEL = '/';

export type CosmoChatContextValue = UseChatHelpers & {
  isOpen: boolean;
  open: () => void;
  close: () => void;
  toggle: () => void;
  /** Wipes the thread so the empty state (example prompts) comes back. */
  reset: () => void;

  /** Past conversations, most recently used first. */
  sessions: ChatSession[];
  activeSessionId: string | null;
  /** Loads a past conversation into the thread. */
  openSession: (id: string) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
  historyLoading: boolean;
};

const CosmoChatContext = createContext<CosmoChatContextValue | null>(null);

export function CosmoChatProvider({ children }: PropsWithChildren) {
  const [isOpen, setIsOpen] = useState(false);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);

  // The id is mirrored into a ref because onFinish runs inside the streaming
  // callback, which closes over the value from the render that started the
  // request — by then a freshly created session would not be visible.
  const sessionRef = useRef<string | null>(null);
  const pendingQuestion = useRef<string>('');

  const chat = useChat({
    id: 'cosmo-widget',
    api: '/api/ai/daily-actions',
    onError: (err) => {
      // The route surfaces real messages via getErrorMessage — keep them in the
      // console too, the thread renders them inline (no silent failures).
      console.error('[cosmo-widget] chat error:', err);
    },
    onFinish: async (message) => {
      const question = pendingQuestion.current;
      pendingQuestion.current = '';
      if (!message.content) return;

      try {
        let id = sessionRef.current;
        if (!id) {
          // The conversation is created on the first completed turn, not when
          // the widget opens: opening and closing it without asking anything
          // should not leave an empty row in the history.
          const created = await ChatSessionsApi.create(question);
          id = created.id;
          sessionRef.current = id;
          setActiveSessionId(id);
          setSessions((prev) => [created, ...prev]);
        }
        await ChatSessionsApi.append(id, [
          ...(question ? [{ role: 'user' as const, content: question }] : []),
          { role: 'assistant' as const, content: message.content },
        ]);
      } catch (err) {
        // Losing the transcript must not break the conversation in progress.
        console.error('[cosmo-widget] could not save the turn:', err);
      }
    },
  });

  const { setMessages, setInput, messages } = chat;

  // Remember the question that is in flight, so onFinish can store the pair
  // together. Reading it back off the thread would be fragile: the assistant
  // message is appended before the callback runs.
  useEffect(() => {
    const last = messages[messages.length - 1];
    if (last?.role === 'user') {
      pendingQuestion.current = last.content;
    }
  }, [messages]);

  const open = useCallback(() => setIsOpen(true), []);
  const close = useCallback(() => setIsOpen(false), []);
  const toggle = useCallback(() => setIsOpen((v) => !v), []);

  const reset = useCallback(() => {
    setMessages([]);
    setInput('');
    // Starting a new thread detaches from the stored conversation; the next
    // completed turn opens a fresh one.
    sessionRef.current = null;
    setActiveSessionId(null);
  }, [setMessages, setInput]);

  const openSession = useCallback(
    async (id: string) => {
      setHistoryLoading(true);
      try {
        const { messages: stored } = await ChatSessionsApi.messages(id);
        setMessages(
          stored.map((m, i) => ({
            id: `${id}-${i}`,
            role: m.role,
            content: m.content,
          }))
        );
        sessionRef.current = id;
        setActiveSessionId(id);
      } catch (err) {
        console.error('[cosmo-widget] could not open the conversation:', err);
      } finally {
        setHistoryLoading(false);
      }
    },
    [setMessages]
  );

  const deleteSession = useCallback(
    async (id: string) => {
      try {
        await ChatSessionsApi.remove(id);
      } catch (err) {
        console.error('[cosmo-widget] could not delete the conversation:', err);
        return;
      }
      setSessions((prev) => prev.filter((s) => s.id !== id));
      if (sessionRef.current === id) {
        setMessages([]);
        sessionRef.current = null;
        setActiveSessionId(null);
      }
    },
    [setMessages]
  );

  // Restore on mount: the session list, and the most recent conversation.
  // A failure here is silent on purpose — the widget still works without its
  // history, and a signed-out or offline user would otherwise see an error for
  // something they did not ask for.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const list = await ChatSessionsApi.list();
        if (cancelled) return;
        setSessions(list);
        if (list[0]) await openSession(list[0].id);
      } catch {
        /* no history is a valid state */
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Global toggle shortcut: Ctrl+/ (Cmd+/ on mac).
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key === '/') {
        event.preventDefault();
        toggle();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [toggle]);

  const value = useMemo<CosmoChatContextValue>(
    () => ({
      ...chat,
      isOpen,
      open,
      close,
      toggle,
      reset,
      sessions,
      activeSessionId,
      openSession,
      deleteSession,
      historyLoading,
    }),
    [
      chat,
      isOpen,
      open,
      close,
      toggle,
      reset,
      sessions,
      activeSessionId,
      openSession,
      deleteSession,
      historyLoading,
    ]
  );

  return (
    <CosmoChatContext.Provider value={value}>
      {children}
    </CosmoChatContext.Provider>
  );
}

export function useCosmoChat(): CosmoChatContextValue {
  const ctx = useContext(CosmoChatContext);
  if (!ctx) {
    throw new Error('useCosmoChat must be used inside <CosmoChatProvider>');
  }
  return ctx;
}
