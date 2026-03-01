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
  leftSection,
}: SheetHeaderProps) => {
  return (
    <div className="flex items-center justify-between gap-2 p-4">
      <p className="text-lg font-bold">{title}</p>
      <div className="flex items-center gap-2">
        {leftSection}
        {!isError && isDraft && onAssistantOpenChange && (
          <MainButton
            className="bg-gradient-to-r from-[#41AEFD] via-[#D543F5] to-[#5A57F2]"
            text="AI Writer"
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
