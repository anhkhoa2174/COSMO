'use client';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import FileDropzone from '@/components/file-dropzone';
import { Image, Loader2, CheckCircle2, AlertCircle } from 'lucide-react';
import { useCallback, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import UrlExtractionApi from '@/network/client/url-extraction';
import { toast } from 'sonner';

interface ExtractFromScreenshotDialogProps {
  contactId?: string;
  trigger?: React.ReactNode;
  onSuccess?: (fields: string[]) => void;
  onExtract?: (extractedData: Record<string, any>) => void;
}

export function ExtractFromScreenshotDialog({
  contactId,
  trigger,
  onSuccess,
  onExtract,
}: ExtractFromScreenshotDialogProps) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [imageFiles, setImageFiles] = useState<File[]>([]);
  const [imageFields, setImageFields] = useState<string[]>([]);

  const extractImageMutation = useMutation({
    mutationFn: async () => {
      if (imageFiles.length === 0) {
        throw new Error('Please upload at least one screenshot');
      }
      if (!contactId) {
        const responses = await Promise.all(
          imageFiles.map((file) =>
            UrlExtractionApi.extractFromImagePreview(file)
          )
        );
        return { mode: 'create', responses };
      }
      const responses = await Promise.all(
        imageFiles.map((file) =>
          UrlExtractionApi.extractFromImage(contactId, file)
        )
      );
      return { mode: 'edit', responses };
    },
    onSuccess: (result) => {
      if (result.mode === 'create') {
        // Create mode: pass extracted data to parent
        const extractedData = result.responses.reduce(
          (acc, response) => {
            const data = (response.data as any).extracted_data || {};
            Object.entries(data).forEach(([key, value]) => {
              if (
                acc[key] === undefined ||
                acc[key] === null ||
                acc[key] === ''
              ) {
                acc[key] = value;
              }
            });
            return acc;
          },
          {} as Record<string, any>
        );
        const fields = Object.keys(extractedData);
        setImageFields(fields);

        if (onExtract) {
          onExtract(extractedData);
        }
        toast.success(
          `Extracted ${fields.length} fields from ${imageFiles.length} screenshot(s)`
        );
        handleClose();
      } else {
        // Edit mode: normal flow
        const fields = result.responses.flatMap((response) => {
          const added = (response.data as any).fields_added || [];
          return Array.isArray(added) ? added : [];
        });
        const uniqueFields = Array.from(new Set(fields));
        setImageFields(uniqueFields);
        toast.success(
          `Successfully extracted ${uniqueFields.length} fields from ${imageFiles.length} screenshot(s)`
        );
        queryClient.invalidateQueries({ queryKey: ['contact', contactId] });
        if (onSuccess) {
          onSuccess(uniqueFields);
        }
        handleClose();
      }
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to extract data from screenshot');
    },
  });

  const handleClose = () => {
    setOpen(false);
    setImageFiles([]);
    setImageFields([]);
    extractImageMutation.reset();
  };

  const handlePaste = useCallback((event: React.ClipboardEvent) => {
    const items = event.clipboardData?.items || [];
    const newFiles: File[] = [];
    for (const item of items) {
      if (item.type.startsWith('image/')) {
        const file = item.getAsFile();
        if (file) {
          newFiles.push(file);
        }
      }
    }
    if (newFiles.length > 0) {
      setImageFiles((prev) => [...prev, ...newFiles]);
      toast.success(`Pasted ${newFiles.length} screenshot(s)`);
      event.preventDefault();
    }
  }, []);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm">
            <Image className="mr-2 h-4 w-4" />
            Extract from Screenshot
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-2xl" onPaste={handlePaste}>
        <DialogHeader>
          <DialogTitle>Extract Profile Data from Screenshot</DialogTitle>
          <DialogDescription>
            Upload or paste a screenshot of a profile page and we'll
            automatically extract structured information using AI.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Screenshot Upload */}
          <div className="space-y-2">
            <Label>Screenshot</Label>
            <FileDropzone
              multiple
              accept={{ 'image/*': [] }}
              onDrop={(files) => {
                if (files && files.length > 0) {
                  setImageFiles((prev) => [...prev, ...files]);
                }
              }}
              files={imageFiles}
              showFilesList
            />
            <p className="text-xs text-muted-foreground">
              You can paste screenshots (Ctrl+V) or drop multiple images here
            </p>
          </div>

          <div className="flex gap-2">
            <Button
              onClick={() => extractImageMutation.mutate()}
              disabled={
                imageFiles.length === 0 || extractImageMutation.isPending
              }
              className="flex-1"
            >
              {extractImageMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Extracting...
                </>
              ) : (
                <>
                  <Image className="mr-2 h-4 w-4" />
                  Extract Data ({imageFiles.length})
                </>
              )}
            </Button>
          </div>

          {/* Loading State */}
          {extractImageMutation.isPending && (
            <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950">
              <div className="flex items-center gap-3">
                <Loader2 className="h-5 w-5 animate-spin text-blue-600 dark:text-blue-400" />
                <div>
                  <p className="font-medium text-blue-900 dark:text-blue-100">
                    Analyzing screenshot...
                  </p>
                  <p className="text-sm text-blue-700 dark:text-blue-300">
                    Using AI vision to extract structured information from the
                    image
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Success State */}
          {extractImageMutation.isSuccess && (
            <div className="space-y-3">
              <div className="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-950">
                <div className="flex items-start gap-3">
                  <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                  <div className="flex-1">
                    <p className="font-medium text-green-900 dark:text-green-100">
                      Screenshot extracted successfully!
                    </p>
                    <p className="text-sm text-green-700 dark:text-green-300">
                      {contactId
                        ? `Added ${imageFields.length} new fields to contact profile.`
                        : 'Data ready to add to new contact.'}
                    </p>
                  </div>
                </div>
              </div>

              {imageFields.length > 0 && (
                <div className="space-y-2">
                  <Label>Fields Extracted:</Label>
                  <div className="flex flex-wrap gap-2">
                    {imageFields.map((field) => (
                      <Badge key={field} variant="secondary">
                        {field.replace(/_/g, ' ')}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              <div className="flex justify-end gap-2">
                <Button onClick={handleClose}>Done</Button>
              </div>
            </div>
          )}

          {/* Error State */}
          {extractImageMutation.isError && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-950">
              <div className="flex items-start gap-3">
                <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                <div>
                  <p className="font-medium text-red-900 dark:text-red-100">
                    Screenshot extraction failed
                  </p>
                  <p className="text-sm text-red-700 dark:text-red-300">
                    {(extractImageMutation.error as any)?.message ||
                      'Could not extract data from screenshot'}
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Info */}
          {!extractImageMutation.isPending &&
            !extractImageMutation.isSuccess && (
              <div className="rounded-lg border bg-muted p-3">
                <p className="text-sm text-muted-foreground">
                  <strong>How it works:</strong> Upload or paste a screenshot of
                  any profile page (LinkedIn, Twitter, resume, etc.). Our AI
                  vision model will analyze the image and extract structured
                  information like name, job title, company, skills, education,
                  and more.
                </p>
              </div>
            )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
