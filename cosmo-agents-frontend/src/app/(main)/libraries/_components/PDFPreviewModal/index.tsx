import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useEffect, useState } from 'react';
import { Worker, Viewer } from '@react-pdf-viewer/core';
import '@react-pdf-viewer/core/lib/styles/index.css';

interface PDFPreviewModalProps {
  opened: boolean;
  onClose: () => void;
  file: { fileName: string; pdfBlob: Blob };
}

const PDFPreviewModal: React.FC<PDFPreviewModalProps> = ({
  opened,
  onClose,
  file,
}) => {
  const [pdfUrl, setPdfUrl] = useState<string | null>(null);

  useEffect(() => {
    const url = URL.createObjectURL(file.pdfBlob);
    setPdfUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [file.pdfBlob]);

  return (
    <Dialog open={opened} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className="max-h-[90vh] max-w-3xl"
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>{file.fileName}</DialogTitle>
          <DialogDescription />
        </DialogHeader>
        <ScrollArea className="h-[80vh] w-full">
          {pdfUrl && (
            <Worker workerUrl="https://unpkg.com/pdfjs-dist@3.11.174/build/pdf.worker.min.js">
              <Viewer fileUrl={pdfUrl} />
            </Worker>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
};

export default PDFPreviewModal;
