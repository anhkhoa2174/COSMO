'use client';

import { MainButton } from '@/components/buttons/main-button';
import { FormSelectField } from '@/components/forms/fields/form-select-field';
import { FormSwitchField } from '@/components/forms/fields/form-switch-field';
import { FormTagsInputField } from '@/components/forms/fields/form-tags-input-field';
import { FormTextField } from '@/components/forms/fields/form-text-field';
import { Form } from '@/components/ui/form';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet';
import { cn, snakeCase } from '@/lib/utils';
import { CustomField } from '@/models/custom-field';
import {
  createCustomField,
  updateCustomField,
} from '@/network/client/custom-field';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Calendar,
  Link,
  List,
  Mail,
  Pencil,
  Plus,
  Sigma,
  Type,
} from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { z } from 'zod';

const formSchema = z.object({
  name: z.string().min(1),
  data_type: z.string().min(1),
  entity_type: z.string().min(1),
  is_required: z.boolean(),
  options: z.array(z.string()),
  sample_data: z.string().optional(),
  fallback_value: z.string().optional(),
});

type FormData = z.infer<typeof formSchema>;

interface CreateCustomFieldFormProps extends React.HTMLAttributes<HTMLFormElement> {
  onSuccess: () => void;
  isEdit?: boolean;
  data?: any;
  isFormBuilder?: boolean;
}

export function CreateCustomFieldForm({
  onSuccess,
  isEdit,
  data,
  isFormBuilder = false,
  ...props
}: CreateCustomFieldFormProps) {
  const queryClient = useQueryClient();
  const form = useForm<FormData>({
    resolver: zodResolver(formSchema),
    // The API returns null for unset sample_data / fallback_value / options,
    // which the string/array schema rejects, so an edit could never be saved.
    defaultValues: data
      ? {
          name: data.name,
          data_type: data.data_type,
          entity_type: data.entity_type,
          is_required: !!data.is_required,
          options: data.options ?? [],
          sample_data: data.sample_data ?? '',
          fallback_value: data.fallback_value ?? '',
        }
      : {
          name: '',
          // data_type: '',
          entity_type: isFormBuilder ? 'contact' : '',
          is_required: false,
          options: [],
          sample_data: '',
          fallback_value: '',
        },
  });

  const mutation = useMutation({
    mutationFn: (body: FormData) => {
      if (isEdit) {
        return updateCustomField(data.id, body as CustomField);
      }
      return createCustomField(body as CustomField);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['custom-fields'] });
      toast.success(`${isEdit ? 'Edit' : 'Add'} custom field successfully`);
      form.reset();
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message ||
          err.message ||
          `${isEdit ? 'Edit' : 'Add'} custom field failed`
      );
    },
  });

  const listType = [
    {
      icon: <Link />,
      value: 'url',
      label: 'URL',
      textOptional: '(with validation)',
    },
    {
      icon: <List />,
      value: 'select',
      label: 'Select',
      textOptional: '(dropdown with options)',
    },
    {
      icon: <Type />,
      value: 'text',
      label: 'Text',
      textOptional: '(single line, max 255 chars)',
    },
    // { icon: <AlignLeft />, value: 'textarea', label: 'Textarea', textOptional: '(multiple lines)' },
    {
      icon: <Mail />,
      value: 'email',
      label: 'Email',
      textOptional: '(with validation)',
    },
    {
      icon: <Calendar />,
      value: 'date',
      label: 'Date',
      textOptional: '(YYYY-MM-DD)',
    },
    {
      icon: <Sigma />,
      value: 'number',
      label: 'Number',
      textOptional: '(integer and decimals)',
    },
  ];

  const onSubmit = (data: FormData) => {
    mutation.mutate(data);
  };

  return (
    <Form {...form}>
      <form
        {...props}
        onSubmit={form.handleSubmit(onSubmit)}
        className={cn('space-y-6', props.className)}
      >
        <FormSelectField
          form={form}
          name="entity_type"
          label="Entity"
          placeholder="Select entity"
          options={
            isFormBuilder
              ? [{ value: 'contact', label: 'Contact' }]
              : [
                  { value: 'contact', label: 'Contact' },
                  { value: 'company', label: 'Company' },
                ]
          }
          withAsterisk
        />
        <FormTextField
          form={form}
          name="name"
          label="Field name"
          withAsterisk
          placeholder="e.g. phone number"
          description={`The field name will be used as the key in the database. The key will be in snake_case format. e.g. ${snakeCase('phone number')}`}
        />
        <FormSelectField
          form={form}
          name="data_type"
          label="Type"
          placeholder="Select type"
          options={listType}
          withAsterisk
          disabled={isEdit}
        />
        {form.watch('data_type') === 'select' && (
          <FormTagsInputField
            form={form}
            name="options"
            label="Options"
            description="Press Enter to add option"
            placeholder="Add options"
          />
        )}
        <FormTextField
          form={form}
          name="sample_data"
          label="Sample data"
          placeholder="Used to generate sample data"
        />
        <FormTextField
          form={form}
          name="fallback_value"
          label="Fallback value"
          placeholder="Used when field is empty"
        />
        <FormSwitchField
          form={form}
          name="is_required"
          label="Required"
          description="If the field is required, it will be required to fill in when creating a contact"
        />
        <MainButton
          type="submit"
          text="Save"
          icon={isEdit ? Pencil : Plus}
          loading={mutation.isPending}
          className="float-right"
        />
      </form>
    </Form>
  );
}

export function CreateCustomFieldsSheet({
  children,
  isEdit,
  item,
  open = false,
}: {
  children?: React.ReactNode;
  isEdit?: boolean;
  open?: boolean;
  item?: any;
}) {
  const [isSheetOpen, setIsSheetOpen] = useState(open);
  return (
    <Sheet open={isSheetOpen} onOpenChange={setIsSheetOpen}>
      <SheetTrigger asChild>{children}</SheetTrigger>
      <SheetContent className="overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{isEdit ? 'Edit' : 'Add'} field</SheetTitle>
          <SheetDescription>
            {isEdit
              ? 'Edit the field to update its name, type, and other details.'
              : 'Add a new field to your contact or company entity.'}
          </SheetDescription>
        </SheetHeader>
        <CreateCustomFieldForm
          data={item}
          isEdit={isEdit}
          onSuccess={() => setIsSheetOpen(false)}
        />
      </SheetContent>
    </Sheet>
  );
}
