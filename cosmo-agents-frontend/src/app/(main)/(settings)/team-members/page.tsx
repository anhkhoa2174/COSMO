'use client';

import { AddButton } from '@/components/buttons/add-button';
import { MainButton } from '@/components/buttons/main-button';
import { ColumnDef, DataTable } from '@/components/data-table/data-table';
import { CreateMemberDialog } from '@/components/forms/create-member-form';
import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { Users2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { MemberSearchResponseData } from '@/models/organization';
import OrganizationApi from '@/network/client/organization';
import { useUser } from '@/hooks/use-user';
import { useQuery } from '@tanstack/react-query';
import { MoreVertical, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

export default function TeamMembersPage() {
  const { user } = useUser();
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['members'],
    queryFn: () => OrganizationApi.searchMember('me', { filter: {} }),
  });

  const memberData = data?.data.list || [];
  const total = data?.data.total || 0;

  const handleDelete = async (member: MemberSearchResponseData) => {
    const myPromise = async () => {
      return await OrganizationApi.deleteMember('me', {
        member_ids: [member.entity.id],
      });
    };

    toast.promise(myPromise, {
      loading: 'Deleting...',
      success: () => {
        refetch();
        return `${member.entity.name} has been deleted`;
      },
      error: 'Failed to delete member',
    });
  };

  const columns: ColumnDef<MemberSearchResponseData>[] = [
    { accessorKey: 'name', header: 'Name', cell: ({ row }) => row.entity.name },
    {
      accessorKey: 'email',
      header: 'Email',
      cell: ({ row }) => row.entity.email,
    },
    {
      accessorKey: 'role',
      header: 'Role',
      cell: ({ row }) => (
        <Badge
          variant={
            row.role.name === 'admin'
              ? 'destructive-secondary'
              : 'info-secondary'
          }
          className="capitalize"
        >
          {row.role.name}
        </Badge>
      ),
    },
    {
      accessorKey: 'job_title',
      header: 'Job title',
      cell: ({ row }) => row.role.job_title,
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => (
        <Badge
          variant={row.role.status === 'active' ? 'success' : 'warning'}
          className="capitalize"
        >
          {row.role.status}
        </Badge>
      ),
    },
    {
      accessorKey: 'action',
      header: 'Action',
      cell: ({ row }) =>
        user?.roles[0].name === 'admin' && row.entity.id !== user?.id ? (
          <DropdownMenu>
            <DropdownMenuTrigger>
              <MainButton variant="ghost" size="icon" icon={MoreVertical} />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem
                className="text-destructive focus:bg-destructive-foreground focus:text-destructive"
                onClick={() => handleDelete(row)}
              >
                <Trash2 /> Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <></>
        ),
    },
  ];

  const rightSection = (
    <CreateMemberDialog>
      <AddButton text="Invite member" />
    </CreateMemberDialog>
  );

  return (
    <ContentLayout title="Team Members" section="Settings" icon={Users2}>
      <PageHero
        icon={Users2}
        eyebrow="Settings"
        accent="blue"
        title="Team Members"
        description="Everyone in your organization, and what each of them is allowed to do."
        actions={rightSection}
      />
      <DataTable
        columns={columns}
        data={memberData}
        loading={isLoading}
        pageCount={total}
        pageKey="members"
        showSelection={false}
      />
    </ContentLayout>
  );
}
