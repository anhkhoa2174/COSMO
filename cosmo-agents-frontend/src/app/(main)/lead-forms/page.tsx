'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { Copy, ExternalLink } from 'lucide-react';

import { ContentLayout } from '@/components/nav/content-layout';
import { ClipboardList } from 'lucide-react';
import { PageHero } from '@/components/ui/page-hero';
import { Badge } from '@/components/ui/badge';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { MainButton } from '@/components/buttons/main-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import LeadFormApi from '@/network/client/lead-form';
import type { LeadForm } from '@/network/client/lead-form';

function getStatusBadge(status?: string) {
  const styles: Record<string, string> = {
    active: 'text-green-600 bg-green-500/10 hover:bg-green-500/20',
    inactive: 'text-gray-600 bg-gray-500/10 hover:bg-gray-500/20',
    draft: 'text-yellow-600 bg-yellow-500/10 hover:bg-yellow-500/20',
  };
  const s = status || 'draft';
  return (
    <Badge
      className={`uppercase ${styles[s] || 'bg-gray-500/10 text-gray-500'}`}
    >
      {s}
    </Badge>
  );
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
  toast.success('Copied to clipboard');
}

export default function LeadFormsPage() {
  const queryClient = useQueryClient();

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

  const forms = data?.data || [];

  return (
    <ContentLayout title="Lead Forms" section="Prospects" icon={ClipboardList}>
      <PageHero
        icon={ClipboardList}
        eyebrow="Prospects"
        accent="amber"
        title="Inbound Lead Forms"
        description="Public forms that drop submissions straight into your contact database and start the outreach sequence automatically."
      />
      {isLoading && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-[200px] rounded-lg" />
          ))}
        </div>
      )}

      {!isLoading && forms.length === 0 && (
        <div className="flex items-center justify-center py-12 text-muted-foreground">
          No lead forms found.
        </div>
      )}

      {!isLoading && forms.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {forms.map((form: LeadForm) => {
            const publicUrl = `${window.location.origin}/public/forms/${form.slug}`;
            return (
              <Card key={form.id}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="space-y-1">
                      <CardTitle className="text-lg">{form.name}</CardTitle>
                      <CardDescription className="font-mono text-xs">
                        {form.slug}
                      </CardDescription>
                    </div>
                    {getStatusBadge(form.status)}
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {form.description && (
                    <p className="line-clamp-2 text-sm text-muted-foreground">
                      {form.description}
                    </p>
                  )}

                  <div className="flex items-center gap-4 text-sm text-muted-foreground">
                    <span>{form.fields?.length || 0} fields</span>
                    <span>{form.submissions_count ?? 0} submissions</span>
                  </div>

                  <div className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs">
                    <ExternalLink className="h-3 w-3 flex-shrink-0" />
                    <span className="truncate">{publicUrl}</span>
                    <MainButton
                      size="icon"
                      variant="ghost"
                      icon={Copy}
                      className="ml-auto h-6 w-6 flex-shrink-0"
                      onClick={() => copyToClipboard(publicUrl)}
                    />
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">
                      {form.created_at
                        ? `Created ${format(new Date(form.created_at), 'MMM d, yyyy')}`
                        : 'Created —'}
                    </span>
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
    </ContentLayout>
  );
}
