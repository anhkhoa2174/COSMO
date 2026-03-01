'use client';

import { CopyButton } from '@/components/plate-ui/code-block-element';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { CheckIcon, RefreshCw, Save, Settings, X } from 'lucide-react';
import Link from 'next/link';
import { toast } from 'sonner';
import { FormState } from './types';

type BuilderHeaderProps = {
  formState: FormState;
  loadingRegenerateUrl: boolean;
  loadingSuccess: boolean;
  error: string | null;
  onPropertyChange: (property: string, value: any) => void;
  onRegenerateUrl: () => void;
  onSaveForm: () => void;
  onOpenSettings: () => void;
  closeSheet: () => void;
  formInboundSlug: string | undefined;
  previewUrl: string;
};

export default function BuilderHeader({
  formState,
  loadingRegenerateUrl,
  loadingSuccess,
  error,
  onPropertyChange,
  onRegenerateUrl,
  onSaveForm,
  onOpenSettings,
  closeSheet,
  formInboundSlug,
  previewUrl,
}: BuilderHeaderProps) {
  const origin = formState.settings?.publicUrl?.origin || '';
  const isErrorName = error?.includes('An inbound lead form with slug')

  return (
    <div className="mb-6 flex items-center justify-between">
      <div className="space-y-1">
        <h1 className="text-3xl font-bold tracking-tight">
          <div className="flex items-center gap-2">
            <div className="flex w-auto whitespace-nowrap">Form Builder -</div>
            {formInboundSlug ? (
              <div className="flex w-auto items-center gap-2">
                <Link
                  className={cn("text-[24px] font-medium text-ellipsis text-[#4F46E5] underline", !formInboundSlug && 'pointer-events-none text-gray-300')}
                  href={previewUrl}
                  target="_blank"
                >
                  {previewUrl}
                </Link>
                <CopyButton
                  size="icon"
                  variant="ghost"
                  className="size-8 gap-1 text-xs"
                  onClickCopy={() => {
                    toast.success('URL copied to clipboard');
                  }}
                  value={previewUrl}
                />
              </div>
            ) : <>
              <div className="text-[24px] font-medium text-muted-foreground">{`${origin}/${formState.settings?.publicUrl?.path || ''}/`}</div>
              <div className='relative'>
                <Input
                  className={cn('w-[200px]', error && 'border-red-500')}
                  type="text"
                  value={formState.settings?.publicUrl?.name || ''}
                  onChange={(e) =>
                    onPropertyChange('publicUrl.name', e.target.value)
                  }
                  disabled={!!formInboundSlug}
                />
                {isErrorName && (
                  <span className="absolute right-0 -bottom-[32px] text-xs text-red-500">
                    {error}
                  </span>
                )}
              </div>
              {loadingSuccess ? (
                <CheckIcon className="mr-2 h-4 w-4 text-green-500" />
              ) : (
                <RefreshCw
                  onClick={onRegenerateUrl}
                  className={
                    'mr-2 h-4 w-4 ' + (loadingRegenerateUrl ? 'animate-spin' : '')
                  }
                />
              )}
            </>}
          </div>
        </h1>
        <p className="text-muted-foreground">
          Create custom forms with drag-and-drop simplicity
        </p>
      </div>
      <div className="flex gap-2">
        <Button variant="success" onClick={onSaveForm}>
          <Save className="mr-2 h-4 w-4" />
          Save Form
        </Button>
        <Button variant="outline" onClick={onOpenSettings}>
          <Settings className="mr-2 h-4 w-4" />
          Settings
        </Button>
        <Button variant="default" onClick={closeSheet}>
          <X className="mr-2 h-4 w-4" />
          Close
        </Button>
      </div>
    </div>
  );
}
