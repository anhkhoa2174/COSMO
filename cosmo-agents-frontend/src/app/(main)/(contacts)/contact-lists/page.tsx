'use client';

import { ImageSearch } from '@/assets/icons';
import { AddButton } from '@/components/buttons/add-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import { MainButton } from '@/components/buttons/main-button';
import { DataTable, type ColumnDef } from '@/components/data-table/data-table';
import { CreateContactListDialog } from '@/components/forms/create-contact-list-form';
import { ContentLayout } from '@/components/nav/content-layout';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  contactListApi,
  GetContactListData,
} from '@/network/client/contact-list';
import { currentPageAtom, selectedIdAtom } from '@/stores/atom';
import { useQuery } from '@tanstack/react-query';
import { cn } from '@udecode/cn';
import { format } from 'date-fns';
import { useAtom } from 'jotai';
import { Edit2, MoreVertical } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { toast } from 'sonner';

export default function ContactListsPage() {
  const router = useRouter();
  const [searchTerm, setSearchTerm] = useState('');
  const [rowsPerPage] = useState(25);
  const [loadingDelete, setLoadingDelete] = useState(false);
  const [selectedId, setSelectedId] = useAtom(selectedIdAtom);
  const [currentPage] = useAtom(currentPageAtom);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [selectedRow, setSelectedRow] = useState<any>(null);
  const pageKey = 'listsTable';

  const {
    data: lists,
    isLoading,
    refetch,
    error,
  } = useQuery({
    queryKey: ['contacts-lists', currentPage[pageKey], rowsPerPage, searchTerm],
    queryFn: () =>
      contactListApi.search(
        {
          filter_: {
            filter_: [{ name: searchTerm }],
          },
        },
        { offset: ((currentPage[pageKey] || 1) - 1) * rowsPerPage, limit: rowsPerPage }
      ),
  });
  const listData =
    lists?.data?.list.map((list) => ({
      ...list.entity,
      creator: list.creator,
      size: list.size,
      number_of_inbound_forms: list.number_of_inbound_forms,
      created_at: format(new Date(list.entity.created_at), 'dd/MM/yyyy HH:mm'),
      updated_at: format(new Date(list.entity.updated_at), 'dd/MM/yyyy HH:mm'),
    })) || [];

  const total = lists?.data?.total || 1;

  const handleSearchChange = (term: string) => {
    setSearchTerm(term);
    refetch();
  };

  const handleDelete = async () => {
    if (!selectedId[pageKey] || selectedId[pageKey].length === 0) {
      toast.error('No contact selected');
      return;
    }

    setLoadingDelete(true);
    try {
      const idsToDelete = { ids: selectedId[pageKey] };

      await contactListApi.delete(idsToDelete);

      toast.success('Contact deleted successfully');

      setSelectedId({});
      refetch();
      router.refresh();
    } catch (err: any) {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    } finally {
      setLoadingDelete(false);
    }
  };

  const rowSelectorExtraContent = (
    <DeleteButton
      text="Delete"
      onConfirm={handleDelete}
      description="This action cannot be undone. This will permanently delete your contact lists and remove
your data from our servers."
      loading={loadingDelete}
    />
  );

  const columns: ColumnDef<GetContactListData>[] = [
    {
      header: 'List name',
      accessorKey: 'name',
      sortable: true,
    },
    {
      header: 'Source',
      accessorKey: 'source',
      cell: ({ row }) => {
        let className = '';
        if (row.source.includes('apollo')) {
          className = 'bg-yellow-300';
        }
        if (row.source.includes('hubspot')) {
          className = 'bg-orange-300';
        }
        if (row?.source?.includes('cosmo-agents')) {
          className = 'bg-gradient-to-r from-blue-300 to-purple-300';
          return (
            <Badge variant="secondary" className={cn(className, 'capitalize')}>
              cosmo agents
            </Badge>
          );
        }
        if (row.source.includes('csv')) {
          className = 'bg-green-300';
        }
        return (
          <Badge variant="secondary" className={cn(className, 'capitalize')}>
            {row.source}
          </Badge>
        );
      },
    },
    {
      header: 'Inbound forms',
      accessorKey: 'number_of_inbound_forms',
      sortable: true,
    },
    { header: 'Size', accessorKey: 'size', sortable: true },
    { header: 'Last updated', accessorKey: 'updated_at', sortable: true },
    { header: 'Created', accessorKey: 'created_at', sortable: true },
    {
      header: 'Creator',
      accessorKey: 'creator',
    },
    {
      accessorKey: 'action',
      header: 'Action',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <MainButton icon={MoreVertical} size="icon" variant="ghost" />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              onClick={() => {
                setSelectedRow(row);
                setIsDialogOpen(true);
              }}
            >
              <Edit2 /> Edit
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  const headerExtraContent = (
    <>
      <AddButton
        variant="secondary"
        text="Create"
        onClick={() => {
          setIsDialogOpen(true);
          setSelectedRow(null);
          setSelectedId({});
        }}
      />
    </>
  );

  return (
    <ContentLayout title="Contact Lists" className="md:w-[calc(100vw-260px)]">
      <div>
        {error ? (
          <div className="flex items-center justify-center">
            <ImageSearch />
            <p className="text-destructive">Error loading contacts</p>
          </div>
        ) : (
          <DataTable
            data={listData}
            columns={columns}
            pageSize={25}
            pageCount={total}
            // rowSelectorExtraContent={rowSelectorExtraContent}
            showSelection={false}
            pageKey={pageKey}
            loading={isLoading}
            onSearchChange={handleSearchChange}
            headerExtraContent={headerExtraContent}
          />
        )}
        <CreateContactListDialog
          onSuccess={() => {
            refetch();
            setIsDialogOpen(false);
          }}
          isEdit={!!selectedRow}
          data={selectedRow}
          open={isDialogOpen}
          onOpenChange={setIsDialogOpen}
        />
      </div>
    </ContentLayout>
  );
}
