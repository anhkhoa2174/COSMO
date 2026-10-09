'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { MainButton } from '@/components/buttons/main-button';
import FileDropzone from '@/components/file-dropzone';
import {
  KNOWLEDGE_TYPES,
  KnowledgeApiV2,
  type KnowledgeType,
} from '@/network/client/knowledge';
import { cn } from '@/lib/utils';

interface UploadPDFModalProps {
  opened: boolean;
  onClose: () => void;
  handleConfirm: (files: File[]) => void;
}

const UploadPDFModal: React.FC<UploadPDFModalProps> = ({
  opened,
  onClose,
  handleConfirm,
}) => {
  const [files, setFiles] = useState<File[]>([]);
  const [type, setType] = useState<KnowledgeType>('other');
  const [isLoading, setIsLoading] = useState(false);

  const handleConfirmClick = async () => {
    if (files.length > 0) {
      setIsLoading(true);
      try {
        await KnowledgeApiV2.upload(files, type);
        handleConfirm(files);
        setFiles([]);
        setType('other');
      } catch (error: any) {
        toast.error(
          error.error?.message || error.message || 'Oops! Something went wrong'
        );
      } finally {
        setIsLoading(false);
      }
    }
  };

  const handleCancel = () => {
    setFiles([]);
    setType('other');
    onClose();
  };

  return (
    <Dialog open={opened} onOpenChange={(open) => !open && handleCancel()}>
      <DialogContent
        className="max-w-3xl"
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>PDF Upload</DialogTitle>
          <DialogDescription>
            Drag and drop your documents here or click to select files.
            Supported formats: PDF, Word (.docx), TXT, CSV, Markdown, JSON,
            HTML, and XML. You can upload multiple files at once. The maximum
            file size is 30MB.
          </DialogDescription>
        </DialogHeader>
        <FileDropzone
          onDrop={setFiles}
          maxSize={30 * 1024 * 1024}
          // Mirrors the backend allow-list in knowledge_service.go; keep the
          // two in sync or the dropzone silently rejects a supported format.
          accept={{
            'application/pdf': ['.pdf'],
            'application/vnd.openxmlformats-officedocument.wordprocessingml.document':
              ['.docx'],
            'text/plain': ['.txt'],
            'text/csv': ['.csv'],
            'text/markdown': ['.md'],
            'application/json': ['.json'],
            'text/html': ['.html'],
            'application/xml': ['.xml'],
          }}
        />
        <div>
          <p className="mb-1.5 text-sm font-medium">
            What are these documents?
          </p>
          <div className="flex flex-wrap gap-1.5">
            {KNOWLEDGE_TYPES.map((t) => (
              <button
                key={t.value}
                type="button"
                onClick={() => setType(t.value)}
                className={cn(
                  'rounded-md border px-2 py-1 text-xs transition-colors',
                  type === t.value
                    ? 'border-primary bg-primary/10 text-foreground'
                    : 'text-muted-foreground hover:bg-accent'
                )}
              >
                {t.label}
              </button>
            ))}
          </div>
          <p className="mt-1.5 text-xs text-muted-foreground">
            AI replies search the matching documents first — a pricing question
            is answered from your pricing documents.
          </p>
        </div>
        <div className="mt-2 flex justify-end gap-2">
          <MainButton variant="outline" text="Cancel" onClick={handleCancel} />
          <MainButton
            text="Upload"
            onClick={handleConfirmClick}
            disabled={!files.length}
            loading={isLoading}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default UploadPDFModal;
