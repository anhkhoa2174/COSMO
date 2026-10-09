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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Link2, Loader2, CheckCircle2, AlertCircle } from 'lucide-react';
import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import UrlExtractionApi from '@/network/client/url-extraction';
import { toast } from 'sonner';

interface ExtractFromURLSimpleDialogProps {
  contactId: string;
  trigger?: React.ReactNode;
  onSuccess?: () => void;
}

export function ExtractFromURLSimpleDialog({
  contactId,
  trigger,
  onSuccess,
}: ExtractFromURLSimpleDialogProps) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState('');
  const [extractedFields, setExtractedFields] = useState<string[]>([]);

  const extractMutation = useMutation({
    mutationFn: () => UrlExtractionApi.extractFromURL(contactId, url),
    onSuccess: (response) => {
      setExtractedFields(response.data.fields_added);
      toast.success(
        `Successfully extracted ${response.data.fields_added.length} fields from URL`
      );
      queryClient.invalidateQueries({ queryKey: ['contact', contactId] });
      if (onSuccess) {
        onSuccess();
      }
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to extract data from URL');
    },
  });

  const handleExtract = () => {
    if (!url.trim()) {
      toast.error('Please enter a URL');
      return;
    }

    // Basic URL validation
    try {
      new URL(url);
    } catch {
      toast.error('Please enter a valid URL');
      return;
    }

    extractMutation.mutate();
  };

  const handleClose = () => {
    setOpen(false);
    setUrl('');
    setExtractedFields([]);
    extractMutation.reset();
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm">
            <Link2 className="mr-2 h-4 w-4" />
            Extract from URL
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Extract Profile Data from URL</DialogTitle>
          <DialogDescription>
            Paste a profile URL (LinkedIn, Twitter, GitHub, personal website,
            etc.) and we'll automatically extract structured information using
            AI.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* URL Input */}
          <div className="space-y-2">
            <Label htmlFor="profile-url">Profile URL</Label>
            <div className="flex gap-2">
              <Input
                id="profile-url"
                placeholder="https://linkedin.com/in/johndoe"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleExtract();
                  }
                }}
                disabled={
                  extractMutation.isPending || extractMutation.isSuccess
                }
              />
              <Button
                onClick={handleExtract}
                disabled={
                  extractMutation.isPending || extractMutation.isSuccess
                }
              >
                {extractMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Extracting...
                  </>
                ) : (
                  <>
                    <Link2 className="mr-2 h-4 w-4" />
                    Extract
                  </>
                )}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Supported: LinkedIn, Twitter, GitHub, personal websites, and more
            </p>
          </div>

          {/* Loading State */}
          {extractMutation.isPending && (
            <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950">
              <div className="flex items-center gap-3">
                <Loader2 className="h-5 w-5 animate-spin text-blue-600 dark:text-blue-400" />
                <div>
                  <p className="font-medium text-blue-900 dark:text-blue-100">
                    Extracting data...
                  </p>
                  <p className="text-sm text-blue-700 dark:text-blue-300">
                    Fetching page content and using AI to extract structured
                    information
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Success State */}
          {extractMutation.isSuccess && (
            <div className="space-y-3">
              <div className="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-950">
                <div className="flex items-start gap-3">
                  <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                  <div className="flex-1">
                    <p className="font-medium text-green-900 dark:text-green-100">
                      Data extracted successfully!
                    </p>
                    <p className="text-sm text-green-700 dark:text-green-300">
                      Added {extractedFields.length} new fields to contact
                      profile. AI re-enrichment triggered automatically.
                    </p>
                  </div>
                </div>
              </div>

              {/* Show extracted fields */}
              {extractedFields.length > 0 && (
                <div className="space-y-2">
                  <Label>Fields Added:</Label>
                  <div className="flex flex-wrap gap-2">
                    {extractedFields.map((field) => (
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
          {extractMutation.isError && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-950">
              <div className="flex items-start gap-3">
                <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                <div>
                  <p className="font-medium text-red-900 dark:text-red-100">
                    Extraction failed
                  </p>
                  <p className="text-sm text-red-700 dark:text-red-300">
                    {(extractMutation.error as any)?.message ||
                      'Could not extract data from URL'}
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Info */}
          {!extractMutation.isPending && !extractMutation.isSuccess && (
            <div className="rounded-lg border bg-muted p-3">
              <p className="text-sm text-muted-foreground">
                <strong>How it works:</strong> We fetch the profile page,
                extract text content, and use AI to identify structured
                information like job title, company, skills, education,
                experience, and more. All extracted data is added to custom
                fields.
              </p>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
