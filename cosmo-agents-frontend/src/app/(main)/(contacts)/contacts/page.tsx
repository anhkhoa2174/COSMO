'use client';

import { MainButton } from '@/components/buttons/main-button';
import { DataTable, type ColumnDef } from '@/components/data-table/data-table';
import { CreateContactDialog } from '@/components/forms/create-contact-form';
import { ContentLayout } from '@/components/nav/content-layout';
import type { Contact } from '@/models/contact';
import { getCustomFields } from '@/network/client/custom-field';
import ContactApi from '@/network/client/contact';
import { currentPageAtom } from '@/stores/atom';
import { useQuery } from '@tanstack/react-query';
import { useAtom } from 'jotai';
import { useState } from 'react';

export default function ContactsPage() {
  const [currentPage] = useAtom(currentPageAtom);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const pageKey = 'contactsTable';
  const pageSize = 25;

  const { data: customFields } = useQuery({
    queryKey: ['customFields'],
    queryFn: () => getCustomFields(),
  });

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['contacts', searchTerm, currentPage[pageKey]],
    queryFn: () =>
      ContactApi.list(
        { filter: searchTerm ? { name: searchTerm } : {} },
        { offset: ((currentPage[pageKey] || 1) - 1) * pageSize, limit: pageSize }
      ),
  });

  const contacts = data?.data?.list?.map((item: any) => item.entity) || [];
  const total = data?.data?.total || contacts.length;

  const columns: ColumnDef<Contact>[] = [
    {
      header: 'Name',
      accessorKey: 'name',
      sortable: true,
      cell: ({ row }) => (row as any).profile?.name || row.name || '-',
    },
    {
      header: 'Company',
      accessorKey: 'company',
      sortable: true,
      cell: ({ row }) => (row as any).profile?.company || row.company || '-',
    },
    {
      header: 'Job Title',
      accessorKey: 'job_title',
      cell: ({ row }) => (row as any).profile?.job_title || row.job_title || '-',
    },
    {
      header: 'Source',
      accessorKey: 'source',
      cell: ({ row }) => row.source || '-',
    },
    {
      header: 'Status',
      accessorKey: 'status',
      cell: ({ row }) => row.status || 'pending',
    },
  ];

  return (
    <ContentLayout title="Contacts" description="Manage your contacts">
      <DataTable
        data={contacts}
        loading={isLoading}
        columns={columns}
        pageCount={total}
        pageSize={pageSize}
        pageKey={pageKey}
        showSelection={false}
        onSearchChange={(term) => setSearchTerm(term)}
        headerExtraContent={
          <MainButton
            text="Add contact"
            onClick={() => setIsDialogOpen(true)}
          />
        }
      />
      <CreateContactDialog
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        onSuccess={() => {
          refetch();
          setIsDialogOpen(false);
        }}
        customFields={customFields?.data?.list || []}
      />
    </ContentLayout>
  );
}
