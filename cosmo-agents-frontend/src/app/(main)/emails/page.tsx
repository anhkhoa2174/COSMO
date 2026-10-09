'use client';

import { useQuery } from '@tanstack/react-query';
import { useAtomValue } from 'jotai';
import { format } from 'date-fns';
import { useState } from 'react';

import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { Mail } from 'lucide-react';
import { DataTable } from '@/components/data-table/data-table';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

import EmailApi from '@/network/client/email';
import { currentPageAtom } from '@/stores/atom';
import type { ColumnDef } from '@/components/data-table/data-table';

export default function EmailsPage() {
  const pageKey = 'emails';
  const pageSize = 50;
  const currentPage = useAtomValue(currentPageAtom);
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const params = {
    offset: ((currentPage[pageKey] || 1) - 1) * pageSize,
    limit: pageSize,
  };

  const { data, isLoading } = useQuery({
    queryKey: ['emails', params, statusFilter],
    queryFn: () =>
      EmailApi.search(
        {
          filter: statusFilter !== 'all' ? { status: statusFilter } : {},
        },
        params
      ),
  });

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: 'entity.from_email',
      header: 'From',
      cell: ({ row }) => (
        <span className="max-w-[180px] truncate text-sm">
          {row.entity?.from_email || '-'}
        </span>
      ),
    },
    {
      accessorKey: 'entity.to_email',
      header: 'To',
      cell: ({ row }) => (
        <span className="max-w-[180px] truncate text-sm">
          {row.entity?.to_email || '-'}
        </span>
      ),
    },
    {
      accessorKey: 'entity.subject',
      header: 'Subject',
      cell: ({ row }) => (
        <span className="max-w-[300px] truncate font-medium">
          {row.entity?.subject || '(no subject)'}
        </span>
      ),
    },
    {
      accessorKey: 'entity.intents',
      header: 'Intent',
      cell: ({ row }) => {
        const intents = row.entity?.intents;
        if (!intents || intents.length === 0) return '-';
        return (
          <Badge variant="outline" className="text-xs">
            {intents[0]}
          </Badge>
        );
      },
    },
    {
      accessorKey: 'entity.status',
      header: 'Status',
      cell: ({ row }) => {
        const status = row.entity?.status || 'unknown';
        const styles: Record<string, string> = {
          draft: 'text-gray-600 bg-gray-500/10',
          sending: 'text-blue-600 bg-blue-500/10',
          sent: 'text-green-600 bg-green-500/10',
          inbox: 'text-purple-600 bg-purple-500/10',
        };
        return (
          <Badge className={`uppercase ${styles[status] || 'text-gray-500 bg-gray-500/10'}`}>
            {status}
          </Badge>
        );
      },
    },
    {
      accessorKey: 'entity.created_at',
      header: 'Date',
      cell: ({ row }) =>
        row.entity?.created_at
          ? format(new Date(row.entity.created_at), 'MMM d, HH:mm')
          : '-',
    },
  ];

  const leftSection = (
    <Select value={statusFilter} onValueChange={setStatusFilter}>
      <SelectTrigger className="w-[140px]">
        <SelectValue placeholder="Filter status" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">All</SelectItem>
        <SelectItem value="draft">Draft</SelectItem>
        <SelectItem value="sending">Sending</SelectItem>
        <SelectItem value="sent">Sent</SelectItem>
        <SelectItem value="inbox">Inbox</SelectItem>
      </SelectContent>
    </Select>
  );

  return (
    <ContentLayout
      title="Emails"
      section="Inbox"
      icon={Mail}
      leftSection={leftSection}
    >
      <PageHero
        icon={Mail}
        eyebrow="Inbox"
        accent="blue"
        title="Emails"
        description="Every message sent and received across your campaigns, with delivery and reply status on each one."
      />
      <DataTable
        columns={columns}
        data={data?.data?.list || []}
        loading={isLoading}
        pageCount={data?.data?.total || 0}
        pageKey={pageKey}
        pageSize={pageSize}
        showSelection={false}
      />
    </ContentLayout>
  );
}
