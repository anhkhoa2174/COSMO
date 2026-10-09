'use client';

import { forwardRef, useRef, useImperativeHandle } from 'react';
import { useAtom } from 'jotai';
import { Paperclip, Mic } from 'lucide-react';
import { chatDraftAtom } from '@/stores/daily-actions';

interface ChatInputBarProps {
  onSend: (message: string) => void;
  isLoading?: boolean;
}

export const ChatInputBar = forwardRef<HTMLInputElement, ChatInputBarProps>(
  function ChatInputBar({ onSend, isLoading }, ref) {
  const [draft, setDraft] = useAtom(chatDraftAtom);
  const inputRef = useRef<HTMLInputElement>(null);

  useImperativeHandle(ref, () => inputRef.current!, []);

  const handleSend = () => {
    const trimmed = draft.trim();
    if (!trimmed || isLoading) return;
    onSend(trimmed);
    setDraft('');
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="shrink-0 border-t border-border bg-card px-7 pb-5 pt-4">
      <div
        className="mx-auto flex max-w-[860px] items-center gap-3 rounded-lg border-[1.5px] border-border bg-background py-1.5 pr-1.5 pl-[18px] transition-colors focus-within:border-indigo-400"
      >
        <input
          ref={inputRef}
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask the agent — try 'pipeline summary', 'list cold contacts', 'top 10 prospects'..."
          className="flex-1 border-none bg-transparent py-2.5 text-[0.9rem] text-foreground outline-none placeholder:text-muted-foreground/80"
          disabled={isLoading}
        />
        <div className="flex items-center gap-1">
          <button
            type="button"
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground/80 transition-colors hover:bg-muted hover:text-muted-foreground"
            title="Attach file"
          >
            <Paperclip className="h-[1.1rem] w-[1.1rem]" />
          </button>
          <button
            type="button"
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground/80 transition-colors hover:bg-muted hover:text-muted-foreground"
            title="Voice input"
          >
            <Mic className="h-[1.1rem] w-[1.1rem]" />
          </button>
        </div>
        <button
          onClick={handleSend}
          disabled={!draft.trim() || isLoading}
          className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-5 py-2.5 text-[0.88rem] font-bold text-white transition-colors hover:opacity-85 disabled:opacity-50"
        >
          Send &rarr;
        </button>
      </div>
    </div>
  );
});
