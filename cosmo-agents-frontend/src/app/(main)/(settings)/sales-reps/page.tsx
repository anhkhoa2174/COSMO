'use client';

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { EllipsisVertical } from 'lucide-react';
import { toast } from 'sonner';
import { AddButton } from '@/components/buttons/add-button';
import { MainButton } from '@/components/buttons/main-button';
import { CreateSalesRepDialog } from '@/components/forms/create-sales-rep-form';
import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { UserRound } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import SalesRepApi from '@/network/client/sales-rep';
import { ColumnDef, DataTable } from '@/components/data-table/data-table';
import { useSearchParams } from 'next/navigation';
import { useRouter } from 'next/navigation';

export default function SalesRepsPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const isModalOpen = searchParams.get('isModalOpen');
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['salesReps'],
    queryFn: () => SalesRepApi.search({ filter: {} }),
  });

  const salesRepData = data?.data.list || [];
  const total = data?.data.total || 0;

  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [selectedSalesRep, setSelectedSalesRep] = useState<any>(null);

  const handleDelete = async (salesRep: any) => {
    const myPromise = async () => {
      return await SalesRepApi.delete(salesRep.entity.id);
    };

    toast.promise(myPromise, {
      loading: 'Deleting...',
      success: () => {
        refetch();
        return `${salesRep.entity.first_name} ${salesRep.entity.last_name} has been deleted`;
      },
      error: 'Failed to delete sales rep',
    });
  };

  const columns: ColumnDef[] = [
    {
      accessorKey: 'first_name',
      header: 'First Name',
      cell: ({ row }) => row.entity.first_name,
    },
    {
      accessorKey: 'last_name',
      header: 'Last Name',
      cell: ({ row }) => row.entity.last_name,
    },
    {
      accessorKey: 'email',
      header: 'Email',
      cell: ({ row }) => row.entity.email,
    },
    {
      accessorKey: 'calendar_link',
      header: 'Calendar Link',
      cell: ({ row }) => (
        <a href={row.entity.calendar_link || ''} target="_blank">
          {row.entity.calendar_link || '-'}
        </a>
      ),
    },
    {
      accessorKey: 'action',
      header: 'Action',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <MainButton size="icon" icon={EllipsisVertical} variant="ghost" />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              onSelect={() => {
                setSelectedSalesRep(row);
                setIsDialogOpen(true);
              }}
            >
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => handleDelete(row)}
              className="text-destructive focus:bg-destructive-foreground focus:text-destructive"
            >
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  const rightSection = (
    <AddButton text="Add sales rep" onClick={() => setIsDialogOpen(true)} />
  );

  useEffect(() => {
    if (isModalOpen === 'true') {
      setIsDialogOpen(true);
    }
  }, [isModalOpen]);

  return (
    <ContentLayout title="Sales Reps" section="Settings" icon={UserRound}>
      <PageHero
        icon={UserRound}
        eyebrow="Settings"
        accent="emerald"
        title="Sales Reps"
        description="The people your campaigns send on behalf of — their name and signature appear on outgoing mail."
        actions={rightSection}
      />
      <DataTable
        columns={columns}
        data={salesRepData}
        loading={isLoading}
        pageCount={total}
        pageKey="salesReps"
        showSelection={false}
      />
      <CreateSalesRepDialog
        isEdit={selectedSalesRep !== null}
        data={selectedSalesRep?.entity}
        open={isDialogOpen}
        onOpenChange={(open) => {
          setIsDialogOpen(open);
          setSelectedSalesRep(null);
          if (!open) {
            router.push('/sales-reps');
          }
        }}
      />
    </ContentLayout>
  );
}
