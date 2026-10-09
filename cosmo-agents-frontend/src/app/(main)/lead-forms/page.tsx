'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { VisuallyHidden } from '@radix-ui/react-visually-hidden';
import { Copy, ExternalLink, Pencil, Plus } from 'lucide-react';

import FormBuilder from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder';
import { ContentLayout } from '@/components/nav/content-layout';
import { ClipboardList } from 'lucide-react';
import { PageHero } from '@/components/ui/page-hero';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Skeleton } from '@/components/ui/skeleton';
import { MainButton } from '@/components/buttons/main-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import { useSheet } from '@/hooks/use-sheet';
import { cn } from '@/lib/utils';
import LeadFormApi from '@/network/client/lead-form';
import type { LeadForm } from '@/network/client/lead-form';

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.success('Copied to clipboard');
}

export default function LeadFormsPage() {
  const queryClient = useQueryClient();
  const {
    open: openSheet,
    close: closeSheet,
    opened: sheetOpened,
    size: sheetSize,
    content: sheetContent,
  } = useSheet();

  const { data, isLoading } = useQuery({
    queryKey: ['lead-forms'],
    queryFn: () => LeadFormApi.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: (identifier: string) => LeadFormApi.delete(identifier),
    onSuccess: () => {
      toast.success('Form deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['lead-forms'] });
    },
    onError: () => {
      toast.error('Failed to delete form');
    },
  });

  // The builder saves through the same inbound-lead-form API; refresh the list
  // and any cached form whenever it closes, whether after a save or not.
  const handleCloseSheet = () => {
    closeSheet();
    queryClient.invalidateQueries({ queryKey: ['lead-forms'] });
    queryClient.invalidateQueries({ queryKey: ['form-inbound-slug'] });
  };

  const openBuilder = (slug?: string) => {
    openSheet({
      size: '2xl',
      content: (
        <FormBuilder
          key={slug ?? 'new'}
          formInboundSlug={slug}
          closeSheet={handleCloseSheet}
          onCreateSuccess={() =>
            queryClient.invalidateQueries({ queryKey: ['lead-forms'] })
          }
        />
      ),
    });
  };

  const forms = data?.data || [];

  return (
    <ContentLayout title="Lead Forms" section="Prospects" icon={ClipboardList}>
      <PageHero
        icon={ClipboardList}
        eyebrow="Prospects"
        accent="amber"
        title="Inbound Lead Forms"
        description="Public forms that drop submissions straight into your contact database and start the outreach sequence automatically."
        actions={
          <MainButton
            text="New form"
            icon={Plus}
            onClick={() => openBuilder()}
          />
        }
      />
      {isLoading && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-[160px] rounded-lg" />
          ))}
        </div>
      )}

      {!isLoading && forms.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
          <p>No lead forms yet.</p>
          <MainButton
            variant="outline"
            text="Create your first form"
            icon={Plus}
            onClick={() => openBuilder()}
          />
        </div>
      )}

      {!isLoading && forms.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {forms.map((form: LeadForm) => {
            const publicUrl = `${window.location.origin}/public/forms/${form.slug}`;
            return (
              <Card
                key={form.id}
                role="button"
                tabIndex={0}
                onClick={() => openBuilder(form.slug)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') openBuilder(form.slug);
                }}
                className="cursor-pointer transition-colors hover:border-amber-400"
              >
                <CardHeader className="pb-3">
                  <CardTitle className="text-lg">{form.name}</CardTitle>
                  <CardDescription className="font-mono text-xs">
                    {form.slug}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div
                    className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <ExternalLink className="h-3 w-3 flex-shrink-0" />
                    <a
                      href={publicUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="truncate hover:underline"
                    >
                      {publicUrl}
                    </a>
                    <MainButton
                      size="icon"
                      variant="ghost"
                      icon={Copy}
                      className="ml-auto h-6 w-6 flex-shrink-0"
                      onClick={() => copyToClipboard(publicUrl)}
                    />
                  </div>

                  <div
                    className="flex items-center justify-between"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <MainButton
                      size="sm"
                      variant="outline"
                      text="Edit form"
                      icon={Pencil}
                      onClick={() => openBuilder(form.slug)}
                    />
                    <DeleteButton
                      size="icon"
                      variant="destructive-secondary"
                      onConfirm={() => deleteMutation.mutateAsync(form.slug)}
                      description="This action cannot be undone. This will permanently delete this lead form."
                      loading={deleteMutation.isPending}
                    />
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      <Sheet
        open={sheetOpened}
        onOpenChange={(open) => !open && handleCloseSheet()}
      >
        <SheetContent withCloseButton={false} className={cn('p-0', sheetSize)}>
          <VisuallyHidden>
            <SheetHeader>
              <SheetTitle>Lead form builder</SheetTitle>
              <SheetDescription>
                Create or edit an inbound lead form
              </SheetDescription>
            </SheetHeader>
          </VisuallyHidden>
          {sheetContent}
        </SheetContent>
      </Sheet>
    </ContentLayout>
  );
}
