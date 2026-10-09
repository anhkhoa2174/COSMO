'use client';

const QUICK_COMMANDS = [
  'Pipeline summary',
  'List contacts COLD',
  'Show FU#1',
  'Contacts bị block',
  'Contacts nguy cơ drop',
  'List 50 contacts outreach',
  'Top 10 prospects',
  'Show meeting pipeline',
] as const;

interface QuickCommandChipsProps {
  onSendCommand: (command: string) => void;
  disabled?: boolean;
}

export function QuickCommandChips({
  onSendCommand,
  disabled,
}: QuickCommandChipsProps) {
  return (
    <div className="mt-4 flex flex-wrap gap-2">
      {QUICK_COMMANDS.map((cmd) => (
        <button
          key={cmd}
          disabled={disabled}
          onClick={() => onSendCommand(cmd)}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-[0.8rem] font-semibold text-muted-foreground transition-colors hover:border-indigo-500/[0.25] hover:text-indigo-600 dark:hover:text-indigo-400 disabled:opacity-50"
        >
          {cmd}
        </button>
      ))}
    </div>
  );
}
