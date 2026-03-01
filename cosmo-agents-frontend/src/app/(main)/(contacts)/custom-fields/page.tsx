'use client';

import { format } from 'date-fns';
import { useQuery } from '@tanstack/react-query';
import { Edit2 } from 'lucide-react';
import { toast } from 'sonner';
import { AddButton } from '@/components/buttons/add-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import { MainButton } from '@/components/buttons/main-button';
import { DataTable, type ColumnDef } from '@/components/data-table/data-table';
import { CreateCustomFieldsSheet } from '@/components/forms/create-custom-fields-form';
import { ContentLayout } from '@/components/nav/content-layout';
import { Badge } from '@/components/ui/badge';
import { CustomField } from '@/models/custom-field';
import {
  deleteCustomField,
  getCustomFields,
} from '@/network/client/custom-field';

export default function ContactsCustomFieldsPage() {
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['custom-fields'],
    queryFn: getCustomFields,
  });

  const pageKey = 'customFieldsTable';

  const defaultColumns: ColumnDef<CustomField>[] = [
    { header: 'Field Name', accessorKey: 'name', sortable: true },
    {
      header: 'Data Type',
      accessorKey: 'data_type',
      sortable: true,
      cell: ({ row }) => getDataTypeLabel(row.data_type),
    },
    {
      header: 'Entity Type',
      accessorKey: 'entity_type',
      sortable: true,
      cell: ({ row }) => <span className="capitalize">{row.entity_type}</span>,
    },
    {
      header: 'Required',
      accessorKey: 'is_required',
      sortable: true,
      cell: ({ row }) => <span>{row.is_required ? 'Yes' : 'No'}</span>,
    },
    {
      header: 'Created At',
      accessorKey: 'created_at',
      sortable: true,
      cell: ({ row }) => <span>{format(new Date(row.created_at), 'Pp')}</span>,
    },
    {
      header: 'Actions',
      accessorKey: 'actions',
      cell: ({ row }) => (
        <div className="flex gap-2">
          <CreateCustomFieldsSheet isEdit item={row}>
            <MainButton size="icon" icon={Edit2} variant="ghost" />
          </CreateCustomFieldsSheet>
          <DeleteButton
            onConfirm={() => handleDeleteField(row)}
            description="This action cannot be undone. This will permanently delete your custom field and remove
your data from our servers."
            variant="destructive-secondary"
          />
        </div>
      ),
    },
  ];

  const handleDeleteField = async (data: CustomField) => {
    const myPromise = async () => {
      return await deleteCustomField(data.id);
    };

    toast.promise(myPromise, {
      loading: 'Deleting...',
      success: () => {
        refetch();
        return `${data.name} field has been deleted`;
      },
      error: 'Error',
    });
  };

  const rightSection = (
    <CreateCustomFieldsSheet open={false}>
      <AddButton text="Add field" />
    </CreateCustomFieldsSheet>
  );

  return (
    <ContentLayout title="Custom Fields" rightSection={rightSection}>
      <DataTable
        data={data?.data.list || []}
        loading={isLoading}
        columns={defaultColumns}
        pageCount={data?.data.total || 0}
        pageSize={25}
        pageKey={pageKey}
        showSelection={false}
      />
    </ContentLayout>
  );
}

function getDataTypeLabel(dataType: string) {
  let className = '';
  switch (dataType) {
    case 'text':
      className = 'text-blue-500 bg-blue-500/10 hover:bg-blue-500/20';
      break;
    case 'select':
      className = 'text-green-500 bg-green-500/10 hover:bg-green-500/20';
      break;
    case 'number':
      className = 'text-yellow-500 bg-yellow-500/10 hover:bg-yellow-500/20';
      break;
    case 'date':
      className = 'text-purple-500 bg-purple-500/10 hover:bg-purple-500/20';
      break;
    case 'url':
      className = 'text-red-500 bg-red-500/10 hover:bg-red-500/20';
      break;
    case 'email':
      className = 'text-orange-500 bg-orange-500/10 hover:bg-orange-500/20';
      break;
    default:
      className = 'text-gray-500 bg-gray-500/10 hover:bg-gray-500/20';
  }

  return <Badge className={`uppercase ${className}`}>{dataType}</Badge>;
}
