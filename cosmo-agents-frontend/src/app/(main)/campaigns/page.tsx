'use client';

import { useQuery } from '@tanstack/react-query';
import { useAtomValue } from 'jotai';
import { MoreVertical, Trash2 } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';

import { AddButton } from '@/components/buttons/add-button';
import { MainButton } from '@/components/buttons/main-button';
import { DataTable } from '@/components/data-table/data-table';
import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { Flag } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

import CampaignApi from '@/network/client/campaign';
import { currentPageAtom } from '@/stores/atom';

import type { ColumnDef } from '@/components/data-table/data-table';
import {
  createTourCustomContent,
  UpdateOnboarding,
  useTour,
} from '@/hooks/use-tour';
import { useUser } from '@/hooks/use-user';
import Link from 'next/link';
import { useEffect } from 'react';
import DescriptionText from '@/components/tour/DescriptionText';

export default function CampaignsPage() {
  const { user, invalidate } = useUser();
  const pageKey = 'campaigns';
  const pageSize = 25;
  const currentPage = useAtomValue(currentPageAtom);
  const params = {
    offset: ((currentPage[pageKey] || 1) - 1) * pageSize,
    limit: pageSize,
  };
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['campaigns', params],
    queryFn: () =>
      CampaignApi.search({ filter: { is_deleted: false } }, params),
  });

  const router = useRouter();

  const handleDelete = async (campaignId: string) => {
    const myPromise = async () => {
      return await CampaignApi.delete(campaignId);
    };

    toast.promise(myPromise, {
      loading: 'Deleting...',
      success: () => {
        refetch();
        return 'Campaign has been deleted';
      },
      error: 'Failed to delete campaign',
    });
  };

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: 'name',
      header: 'Campaign name',
      cell: ({ row }) => (
        <Link
          href={`/campaigns/${row.entity.id}`}
          className="font-medium text-blue-500 hover:underline hover:underline-offset-2"
        >
          {row.entity.name}
        </Link>
      ),
    },
    {
      accessorKey: 'agent',
      header: 'Agent',
      cell: ({ row }) => row.entity.agent?.name || '-',
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => getStatusLabel(row.entity.status),
    },
    {
      accessorKey: 'sent',
      header: 'Sent',
      cell: ({ row }) => row.sent,
    },
    {
      accessorKey: 'replies',
      header: 'Replies',
      cell: ({ row }) => row.reply,
    },
    {
      accessorKey: 'reply_rate',
      header: 'Reply rate',
      cell: ({ row }) => `${(row.reply_rate * 100).toFixed(2)}%`,
    },
    {
      accessorKey: 'interested',
      header: 'Interested',
      cell: ({ row }) => row.interested,
    },
    {
      accessorKey: 'interest_rate',
      header: 'Interest rate',
      cell: ({ row }) => `${(row.interest_rate * 100).toFixed(2)}%`,
    },
    {
      accessorKey: 'creator',
      header: 'Creator',
      cell: ({ row }) => row.creator,
    },
    {
      accessorKey: 'action',
      header: 'Action',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <MainButton size="icon" icon={MoreVertical} variant="ghost" />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              className="text-destructive focus:bg-destructive-secondary focus:text-destructive"
              onClick={() => handleDelete(row.entity.id)}
            >
              <Trash2 />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  const rightSection = (
    <AddButton
      id="campaigns-create-button"
      text="Create campaign"
      onClick={() => router.push('/campaigns/playbook')}
    />
  );

  const { start, reset } = useTour([
    {
      element: '#campaigns-create-button',
      popover: {
        title: '',
        side: 'left',
        description: '',
        customContent: createTourCustomContent({
          isNextSubmit: true,
          title: 'Create campaign',
          description: (
            <DescriptionText
              title="Start your outreach!"
              description="Create a campaign to begin engaging leads with AI-powered messaging."
            />
          ),
          imageSrc: '',
          user,
          onNext: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onPrev: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onFinish: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'completed',
                },
              },
              user
            );
          },
          onSkipAll: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'skipped',
                },
              },
              user
            );
          },
          onNavigate: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          update: invalidate,
        }),
      },
    },
  ]);

  useEffect(() => {
    if (
      user?.ui_metadata?.campaigns?.onboarding &&
      data?.data.list.length === 0
    ) {
      start();
    }
    return () => {
      reset();
    };
  }, [user, data]);

  return (
    <ContentLayout title="Campaigns" section="Campaigns" icon={Flag}>
      <PageHero
        icon={Flag}
        eyebrow="Campaigns"
        accent="violet"
        title="Campaigns"
        description="Multi-step outreach sequences with follow-up delays and intent-based reply routing."
        actions={rightSection}
      />
      <DataTable
        columns={columns}
        data={data?.data.list || []}
        loading={isLoading}
        pageCount={data?.data.total || 0}
        pageKey={pageKey}
        showSelection={false}
      />
    </ContentLayout>
  );
}

function getStatusLabel(status: string) {
  let className = '';
  switch (status) {
    case 'active':
      className = 'text-green-500 bg-green-500/10 hover:bg-green-500/20';
      break;
    case 'paused':
      className = 'text-blue-500 bg-blue-500/10 hover:bg-blue-500/20';
      break;
    case 'ended':
      className = 'text-yellow-500 bg-yellow-500/10 hover:bg-yellow-500/20';
      break;
    case 'scheduled':
      className = 'text-indigo-500 bg-indigo-500/10 hover:bg-indigo-500/20';
      break;
    default:
      className = 'text-gray-500 bg-gray-500/10 hover:bg-gray-500/20';
  }

  return <Badge className={`uppercase ${className}`}>{status}</Badge>;
}
