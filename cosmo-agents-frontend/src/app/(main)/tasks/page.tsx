'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAtomValue } from 'jotai';
import { format } from 'date-fns';
import { toast } from 'sonner';

import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { CheckSquare } from 'lucide-react';
import { DataTable } from '@/components/data-table/data-table';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

import TaskApi from '@/network/client/task';
import { currentPageAtom } from '@/stores/atom';

import type { ColumnDef } from '@/components/data-table/data-table';
import type { TaskStatus } from '@/models/task';
import { useState } from 'react';

export default function TasksPage() {
  const pageKey = 'tasks';
  const pageSize = 50;
  const currentPage = useAtomValue(currentPageAtom);
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const params = {
    offset: ((currentPage[pageKey] || 1) - 1) * pageSize,
    limit: pageSize,
    ...(statusFilter !== 'all' ? { status: statusFilter as TaskStatus } : {}),
  };

  const { data, isLoading } = useQuery({
    queryKey: ['tasks', params],
    queryFn: () => TaskApi.list(params),
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => TaskApi.update(id, { status: 'cancelled' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      toast.success('Task cancelled');
    },
    onError: () => toast.error('Failed to cancel task'),
  });

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => getStatusBadge(row.status),
    },
    {
      accessorKey: 'priority',
      header: 'Priority',
      cell: ({ row }) => getPriorityBadge(row.priority),
    },
    {
      accessorKey: 'campaign_id',
      header: 'Campaign',
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground">
          {row.campaign_id?.slice(0, 8)}...
        </span>
      ),
    },
    {
      accessorKey: 'contact_id',
      header: 'Contact',
      cell: ({ row }) => (
        <span className="font-mono text-xs text-muted-foreground">
          {row.contact_id?.slice(0, 8)}...
        </span>
      ),
    },
    {
      accessorKey: 'schedule_at',
      header: 'Scheduled',
      cell: ({ row }) =>
        row.schedule_at
          ? format(new Date(row.schedule_at), 'MMM d, HH:mm')
          : '-',
    },
    {
      accessorKey: 'done_at',
      header: 'Completed',
      cell: ({ row }) =>
        row.done_at ? format(new Date(row.done_at), 'MMM d, HH:mm') : '-',
    },
    {
      accessorKey: 'error',
      header: 'Error',
      cell: ({ row }) =>
        row.error ? (
          <span className="text-xs text-destructive">{row.error}</span>
        ) : (
          '-'
        ),
    },
    {
      accessorKey: 'created_at',
      header: 'Created',
      cell: ({ row }) => format(new Date(row.created_at), 'MMM d, HH:mm'),
    },
  ];

  const leftSection = (
    <Select value={statusFilter} onValueChange={setStatusFilter}>
      <SelectTrigger className="w-[140px]">
        <SelectValue placeholder="Filter status" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">All</SelectItem>
        <SelectItem value="pending">Pending</SelectItem>
        <SelectItem value="running">Running</SelectItem>
        <SelectItem value="done">Done</SelectItem>
        <SelectItem value="failed">Failed</SelectItem>
        <SelectItem value="cancelled">Cancelled</SelectItem>
      </SelectContent>
    </Select>
  );

  return (
    <ContentLayout
      title="Tasks"
      section="Pipeline"
      icon={CheckSquare}
      leftSection={leftSection}
    >
      <PageHero
        icon={CheckSquare}
        eyebrow="Pipeline"
        accent="violet"
        title="Tasks"
        description="Scheduled sends and follow-ups queued by the worker — what is due, what ran, and what failed."
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
    </ContentLayout>
  );
}

function getStatusBadge(status: string) {
  const styles: Record<string, string> = {
    pending: 'text-yellow-600 bg-yellow-500/10',
    running: 'text-blue-600 bg-blue-500/10',
    done: 'text-green-600 bg-green-500/10',
    failed: 'text-red-600 bg-red-500/10',
    cancelled: 'text-gray-500 bg-gray-500/10',
  };
  return (
    <Badge className={`uppercase ${styles[status] || styles.cancelled}`}>
      {status}
    </Badge>
  );
}

function getPriorityBadge(priority: string) {
  const styles: Record<string, string> = {
    critical: 'text-red-600 bg-red-500/10',
    high: 'text-orange-600 bg-orange-500/10',
    medium: 'text-blue-600 bg-blue-500/10',
    low: 'text-gray-500 bg-gray-500/10',
  };
  return (
    <Badge variant="outline" className={styles[priority] || styles.low}>
      {priority}
    </Badge>
  );
}
