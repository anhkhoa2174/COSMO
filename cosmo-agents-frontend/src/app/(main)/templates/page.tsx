'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAtomValue } from 'jotai';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { useState } from 'react';
import { Eye } from 'lucide-react';

import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { LayoutTemplate } from 'lucide-react';
import { DataTable } from '@/components/data-table/data-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DeleteButton } from '@/components/buttons/delete-button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

import {
  listTemplates,
  deleteTemplate,
  getTemplate,
} from '@/network/client/template';
import { currentPageAtom } from '@/stores/atom';

import type { ColumnDef } from '@/components/data-table/data-table';
import type { GetTemplateData } from '@/network/client/template';

export default function TemplatesPage() {
  const pageKey = 'templates';
  const pageSize = 50;
  const currentPage = useAtomValue(currentPageAtom);
  const queryClient = useQueryClient();
  const [selectedTemplate, setSelectedTemplate] =
    useState<GetTemplateData | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const params = {
    offset: ((currentPage[pageKey] || 1) - 1) * pageSize,
    limit: pageSize,
  };

  const { data, isLoading } = useQuery({
    queryKey: ['templates', params],
    queryFn: () => listTemplates(params),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      toast.success('Template deleted');
    },
    onError: () => toast.error('Failed to delete template'),
  });

  const handleView = async (template: GetTemplateData) => {
    if (template.content) {
      setSelectedTemplate(template);
      return;
    }
    setDetailLoading(true);
    try {
      const res = await getTemplate(template.id);
      setSelectedTemplate(res.data);
    } catch {
      toast.error('Failed to load template');
    } finally {
      setDetailLoading(false);
    }
  };

  const columns: ColumnDef<GetTemplateData>[] = [
    {
      accessorKey: 'subject',
      header: 'Subject',
      cell: ({ row }) => (
        <button
          className="text-left font-medium text-primary hover:underline"
          onClick={() => handleView(row)}
        >
          {row.subject || 'Untitled'}
        </button>
      ),
    },
    {
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => getTypeBadge(row.type),
    },
    {
      accessorKey: 'intent_type',
      header: 'Intent',
      cell: ({ row }) =>
        row.intent_type ? (
          <Badge variant="outline">{row.intent_type}</Badge>
        ) : (
          '-'
        ),
    },
    {
      accessorKey: 'send_after',
      header: 'Send After',
      cell: ({ row }) =>
        row.send_after != null ? `${row.send_after} day(s)` : '-',
    },
    {
      accessorKey: 'created_at',
      header: 'Created',
      cell: ({ row }) =>
        row.created_at
          ? format(new Date(row.created_at), 'MMM d, HH:mm')
          : '-',
    },
    {
      accessorKey: 'actions',
      header: '',
      cell: ({ row }) => (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => handleView(row)}
            disabled={detailLoading}
          >
            <Eye className="h-4 w-4" />
          </Button>
          <DeleteButton
            size="icon"
            variant="destructive-secondary"
            onConfirm={() => deleteMutation.mutateAsync(row.id)}
            description="This will permanently delete this template."
          />
        </div>
      ),
    },
  ];

  return (
    <ContentLayout
      title="Templates"
      section="Campaigns"
      icon={LayoutTemplate}
    >
      <PageHero
        icon={LayoutTemplate}
        eyebrow="Campaigns"
        accent="blue"
        title="Templates"
        description="Email bodies with merge tags, drafted with AI and previewed against a real contact before you use them."
      />
      <DataTable
        columns={columns}
        data={data?.data?.items || []}
        loading={isLoading}
        pageCount={data?.data?.total || 0}
        pageKey={pageKey}
        pageSize={pageSize}
        showSelection={false}
      />

      <Dialog
        open={!!selectedTemplate}
        onOpenChange={() => setSelectedTemplate(null)}
      >
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {selectedTemplate?.subject || 'Untitled Template'}
            </DialogTitle>
          </DialogHeader>
          {selectedTemplate && (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                {getTypeBadge(selectedTemplate.type)}
                {selectedTemplate.intent_type && (
                  <Badge variant="outline">
                    {selectedTemplate.intent_type}
                  </Badge>
                )}
                {selectedTemplate.send_after != null && (
                  <span className="text-sm text-muted-foreground">
                    Send after {selectedTemplate.send_after} day(s)
                  </span>
                )}
              </div>
              <div
                className="prose prose-sm max-w-none rounded-lg border bg-muted/30 p-4"
                dangerouslySetInnerHTML={{
                  __html: selectedTemplate.content || '<p>No content</p>',
                }}
              />
            </div>
          )}
        </DialogContent>
      </Dialog>
    </ContentLayout>
  );
}

function getTypeBadge(type: string) {
  const styles: Record<string, string> = {
    email: 'text-blue-600 bg-blue-500/10',
    linkedin: 'text-indigo-600 bg-indigo-500/10',
    sms: 'text-green-600 bg-green-500/10',
  };
  return (
    <Badge className={`uppercase ${styles[type?.toLowerCase()] || 'text-gray-500 bg-gray-500/10'}`}>
      {type || 'Unknown'}
    </Badge>
  );
}
