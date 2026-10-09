'use client';

import FormBuilder from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder';
import { MainButton } from '@/components/buttons/main-button';
import { FormDatePickerField } from '@/components/forms/fields/form-date-picker-field';
import { FormNumberField } from '@/components/forms/fields/form-number-field';
import { FormSelectField } from '@/components/forms/fields/form-select-field';
import { FormTextField } from '@/components/forms/fields/form-text-field';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Form } from '@/components/ui/form';
import { useSheet } from '@/hooks/use-sheet';
import { cn, getValuable } from '@/lib/utils';
import { Contact } from '@/models/contact';
import ContactApi from '@/network/client/contact';
import { contactListApi, InboundLeadForm } from '@/network/client/contact-list';
import { currentPageAtom, selectedIdAtom } from '@/stores/atom';
import { zodResolver } from '@hookform/resolvers/zod';
import { DialogProps } from '@radix-ui/react-dialog';
import { VisuallyHidden } from '@radix-ui/react-visually-hidden';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useAtom } from 'jotai';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { z } from 'zod';
import { ColumnDef, DataTable } from '../data-table/data-table';
import { Button } from '../ui/button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '../ui/sheet';
import { resolveContactEmail, resolveContactPhone } from '@/lib/contact-info';

interface Field {
  normalized_name: string;
  name: string;
  data_type: string;
  is_required: boolean;
  options?: string[];
}

interface CreateContactListFormProps {
  data?: any;
  isEdit?: boolean;
  onSuccess: () => void;
}

const systemFields = [
  {
    normalized_name: 'name',
    name: 'List name',
    data_type: 'text',
    is_required: true,
  },
] as Field[];

export function CreateContactListForm({
  data,
  isEdit = false,
  onSuccess,
}: CreateContactListFormProps) {
  const [filter, setFilter] = useState<Record<any, any>>({});
  const [currentPage] = useAtom(currentPageAtom);
  const [selectedId, setSelectedId] = useAtom(selectedIdAtom);
  const [inboundForms, setInboundForms] = useState<InboundLeadForm[]>([]);
  const [contactIds, setContactIds] = useState<string[]>([]);
  const pageKey = 'contactsTable';
  const pageSize = 10;

  const {
    open: openSheet,
    close: closeSheet,
    opened: sheetOpened,
    size: sheetSize,
    content: sheetContent,
  } = useSheet();

  useEffect(() => {
    if (isEdit) {
      (async () => {
        setSelectedId({});
        const contactList = await contactListApi.detail(data.id);
        const ids = contactList.data.contacts.map((contact) => contact.id);
        setContactIds(ids);
        setSelectedId({ contactsTable: ids });
        setInboundForms(contactList.data?.inbound_lead_forms || []);
      })();
    }
  }, [isEdit, data]);

  const {
    data: contacts,
    isLoading,
    refetch: refetchContacts,
    error,
  } = useQuery({
    queryKey: [
      'contacts',
      { filter },
      { offset: ((currentPage[pageKey] || 1) - 1) * pageSize, limit: pageSize },
    ],
    queryFn: () =>
      ContactApi.list(
        { filter },
        {
          offset: ((currentPage[pageKey] || 1) - 1) * pageSize,
          limit: pageSize,
        }
      ),
  });

  const { data: fieldData, refetch: refetchFieldsValue } = useQuery({
    queryKey: ['contacts', 'fieldValue'],
    queryFn: () =>
      ContactApi.getValue({
        fields: ['company', 'job_title'],
        offset: 0,
        limit: 100,
      }),
  });
  const companyValue =
    fieldData?.data?.list?.find((item) => item.company)?.company?.list || [];
  const jobTitleValue =
    fieldData?.data?.list?.find((item) => item.job_title)?.job_title?.list ||
    [];
  const _fieldsValue = {
    company: companyValue.filter(
      (item: unknown) =>
        !((typeof item === 'string' && !item.length) || item === null)
    ),
    job_title: jobTitleValue.filter(
      (item: unknown) =>
        !((typeof item === 'string' && !item.length) || item === null)
    ),
  };

  const columns: ColumnDef<Contact>[] = [
    {
      accessorKey: 'name',
      sortable: true,
      header: 'Name',
      cell: ({ row }) => <span className="font-semibold">{row.name}</span>,
    },
    {
      accessorKey: 'email',
      sortable: true,
      header: 'Email',
      cell: ({ row }) => {
        const email = resolveContactEmail(row);
        return email ? (
          <span className="text-sm">{email}</span>
        ) : (
          <span className="text-muted-foreground">-</span>
        );
      },
    },
    {
      accessorKey: 'phone',
      sortable: true,
      header: 'Phone',
      cell: ({ row }) => {
        const phone = resolveContactPhone(row);
        return phone ? (
          <span className="text-sm">{phone}</span>
        ) : (
          <span className="text-muted-foreground">-</span>
        );
      },
    },
    {
      accessorKey: 'address',
      sortable: true,
      header: 'Address',
    },
    {
      accessorKey: 'city',
      sortable: true,
      header: 'City',
    },
    {
      accessorKey: 'country',
      sortable: true,
      header: 'Country',
    },
    {
      accessorKey: 'job_title',
      sortable: true,
      header: 'Job title',
    },
    {
      accessorKey: 'company',
      sortable: true,
      header: 'Company',
    },
  ];

  const FormSchema = z.object({
    id: z.string().optional(),
    ...systemFields.reduce((acc, field) => {
      const schema = (() => {
        if (field.data_type === 'email') {
          return field.is_required
            ? z.string().email({ message: 'Invalid email address' })
            : z.string().email({ message: 'Invalid email address' }).optional();
        }
        if (field.data_type === 'url') {
          return field.is_required
            ? z.string().url({ message: 'Invalid URL' })
            : z.string().url({ message: 'Invalid URL' }).optional();
        }
        return field.is_required
          ? z.string().nonempty()
          : z.string().optional();
      })();
      return { ...acc, [field.normalized_name]: schema };
    }, {}),
  });

  const form = useForm<any>({
    resolver: zodResolver(FormSchema),
    defaultValues: data ? { ...getValuable(data) } : undefined,
  });
  const mutation = useMutation({
    mutationFn: (dataForm: any) =>
      isEdit
        ? contactListApi.update(data.id, {
            ...dataForm,
            ...(inboundForms.length > 0
              ? { inbound_lead_form_ids: inboundForms.map((form) => form.id) }
              : {}),
            contact_ids:
              JSON.stringify(contactIds) !==
              JSON.stringify(selectedId?.contactsTable)
                ? selectedId?.contactsTable || []
                : contactIds,
          })
        : contactListApi.create({
            ...dataForm,
            ...(inboundForms.length > 0
              ? { inbound_lead_form_ids: inboundForms.map((form) => form.id) }
              : {}),
            contact_ids: selectedId?.contactsTable || [],
          }),
    onSuccess: () => {
      toast.success(
        isEdit
          ? 'Contact list has been updated'
          : 'Contact list has been created'
      );
      form.reset();
      onSuccess();
      setSelectedId({});
      refetchContacts();
      refetchFieldsValue();
      setInboundForms([]);
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message ||
          err.message ||
          (isEdit
            ? 'Contact list has not been updated'
            : 'Contact list has not been created')
      );
    },
  });

  const onSubmit = (data: FormData) => {
    mutation.mutate(getValuable(data));
  };

  const renderFields = (fields: Field[]) => {
    return fields.map(
      ({ normalized_name, name, data_type, is_required, options }) => {
        if (['text', 'email', 'url'].includes(data_type)) {
          return (
            <FormTextField
              key={normalized_name}
              form={form}
              name={normalized_name}
              label={name}
              placeholder={`Enter ${name}`}
              withAsterisk={is_required}
              type={data_type}
            />
          );
        }
        if (['select'].includes(data_type)) {
          return (
            <FormSelectField
              key={normalized_name}
              form={form}
              name={normalized_name}
              label={name}
              placeholder={`Select ${name}`}
              options={
                options?.map((option) => ({
                  value: option,
                  label: option,
                })) ?? []
              }
              withAsterisk={is_required}
            />
          );
        }
        if (['number'].includes(data_type)) {
          return (
            <FormNumberField
              key={normalized_name}
              form={form}
              name={normalized_name}
              label={name}
              placeholder={`Enter ${name}`}
              withAsterisk={is_required}
            />
          );
        }
        if (['date'].includes(data_type)) {
          return (
            <FormDatePickerField
              key={normalized_name}
              form={form}
              name={normalized_name}
              label={name}
              placeholder={`Pick a ${name} date`}
              withAsterisk={is_required}
            />
          );
        }

        return null;
      }
    );
  };

  const handleCloseSheet = () => {
    closeSheet();
  };

  const dataContacts =
    contacts?.data?.list.flatMap((item: { entity: any }) => item.entity) || [];

  const pageCountContacts = contacts?.data?.total || 0;

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4">
            {renderFields(systemFields)}
          </div>
        </div>
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">Inbound Form</p>
          </div>
          <div className="grid w-full grid-cols-1 gap-4">
            {inboundForms.map((form) => (
              <Button
                key={form.id}
                type="button"
                variant="outline"
                onClick={() => {
                  openSheet({
                    size: '2xl',
                    content: (
                      <FormBuilder
                        formInboundSlug={form.slug}
                        closeSheet={closeSheet}
                      />
                    ),
                  });
                }}
              >
                {form.slug}
              </Button>
            ))}
            <Button
              type="button"
              variant="outline"
              className="border-dashed"
              onClick={() =>
                openSheet({
                  size: '2xl',
                  content: (
                    <FormBuilder
                      closeSheet={closeSheet}
                      onCreateSuccess={(_, data) => {
                        setInboundForms((prevForms) => [
                          ...new Set([...prevForms, data]),
                        ]);
                      }}
                    />
                  ),
                })
              }
            >
              + Create Inbound Form
            </Button>
          </div>
        </div>
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">Contacts</p>
          </div>
          <div className="grid w-full grid-cols-1 gap-4">
            {error ? (
              <p className="p-2 text-center text-destructive">
                Error loading contacts
              </p>
            ) : (
              <DataTable
                data={dataContacts}
                loading={isLoading}
                columns={columns}
                pageCount={pageCountContacts}
                pageSize={25}
                pageKey={pageKey}
                filterIdButton="contacts-table-filter-button"
                onFilterChange={setFilter}
                filterFieldSelections={_fieldsValue}
                className="h-[calc(100vh-40rem)] overflow-auto"
                isScrollViewport
              />
            )}
          </div>
        </div>

        <Sheet
          open={sheetOpened}
          onOpenChange={(open) => !open && handleCloseSheet()}
        >
          <SheetContent
            withCloseButton={false}
            className={cn('p-0', sheetSize)}
          >
            <VisuallyHidden>
              <SheetHeader>
                <SheetTitle />
                <SheetDescription />
              </SheetHeader>
            </VisuallyHidden>
            {sheetContent}
          </SheetContent>
        </Sheet>
        <MainButton
          text="Submit"
          className="w-full"
          loading={mutation.isPending}
        />
      </form>
    </Form>
  );
}

interface CreateContactListDialogProps
  extends CreateContactListFormProps, DialogProps {}

export function CreateContactListDialog({
  data,
  isEdit = false,
  onSuccess,
  ...props
}: CreateContactListDialogProps) {
  return (
    <Dialog {...props}>
      {/* <DialogTrigger asChild>{children}</DialogTrigger> */}
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-screen-sm">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? 'Edit Contact List' : 'Add a New Contact List'}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Manually edit a contact list from your contacts.'
              : 'Manually add a new contact list from your contacts.'}
          </DialogDescription>
        </DialogHeader>
        <CreateContactListForm
          onSuccess={onSuccess}
          data={data}
          isEdit={isEdit}
        />
      </DialogContent>
    </Dialog>
  );
}
