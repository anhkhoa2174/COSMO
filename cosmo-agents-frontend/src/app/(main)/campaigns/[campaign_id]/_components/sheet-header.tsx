import { Sparkles } from 'lucide-react';
import { MainButton } from '@/components/buttons/main-button';
import React from 'react';

interface SheetHeaderProps {
  title?: string;
  isError?: boolean;
  isDraft?: boolean;
  hasChanged: boolean;
  onSubmit: () => void;
  onClose: () => void;
  onAssistantOpenChange?: () => void;
  /** Lets the trigger show whether the panel is currently open. */
  assistantOpen?: boolean;
  leftSection?: React.ReactNode;
}

const SheetHeader = ({
  title = '',
  isError = false,
  isDraft = true,
  hasChanged,
  onSubmit,
  onClose,
  onAssistantOpenChange,
  assistantOpen = false,
  leftSection,
}: SheetHeaderProps) => {
  return (
    <div className="flex items-center justify-between gap-2 p-4">
      <div className="flex min-w-0 items-center gap-3">
        <p className="truncate text-lg font-bold">{title}</p>
        {/* The Save button appearing is a subtle cue; say it outright. */}
        {!isError && hasChanged && (
          <span className="flex shrink-0 items-center gap-1.5 rounded-full bg-amber-50 px-2.5 py-1 text-[0.75rem] font-medium text-amber-700">
            <span className="size-1.5 rounded-full bg-amber-500" />
            Unsaved changes
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        {leftSection}
        {!isError && isDraft && onAssistantOpenChange && (
          <MainButton
            className={
              assistantOpen
                ? 'border border-violet-300 bg-violet-50 text-violet-700 hover:bg-violet-100'
                : 'bg-gradient-to-r from-[#41AEFD] via-[#D543F5] to-[#5A57F2]'
            }
            text={assistantOpen ? 'Hide AI Writer' : 'AI Writer'}
            icon={Sparkles}
            onClick={onAssistantOpenChange}
          />
        )}
        {!isError && hasChanged && (
          <>
            <MainButton onClick={onSubmit} text="Save" />
            <MainButton onClick={onClose} variant="ghost" text="Discard" />
          </>
        )}
        {(isError || !hasChanged) && (
          <MainButton onClick={onClose} text="Close" />
        )}
      </div>
    </div>
  );
};

export { SheetHeader };
