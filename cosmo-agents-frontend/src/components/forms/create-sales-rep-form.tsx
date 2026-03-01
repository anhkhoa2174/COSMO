'use client';

import { PropsWithChildren } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { z } from 'zod';
import { MainButton } from '@/components/buttons/main-button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Form } from '@/components/ui/form';
import { getValuable, validURL } from '@/lib/utils';
import SalesRepApi from '@/network/client/sales-rep';
import { FormTextField } from './fields/form-text-field';

const FormSchema = z.object({
  id: z.string().optional(),
  first_name: z.string().nonempty(),
  last_name: z.string().nonempty(),
  email: z.string().email({
    message: 'Invalid email address',
  }),
  calendar_link: z
    .string()
    .refine((value) => !value || validURL(value), {
      message: 'Invalid url',
    })
    .optional(),
});

type FormData = z.infer<typeof FormSchema>;

interface CreateSalesRepFormProps {
  onSuccess: () => void;
  isEdit?: boolean;
  data?: FormData;
}
export function CreateSalesRepForm({ onSuccess, isEdit = false, data }: CreateSalesRepFormProps) {
  const queryClient = useQueryClient();
  const form = useForm<FormData>({
    resolver: zodResolver(FormSchema),
    defaultValues: data ? { ...data, calendar_link: '' } : undefined,
  });
  const mutation = useMutation({
    mutationFn: isEdit
      ? (_data: FormData) => SalesRepApi.update(data?.id as string, _data)
      : SalesRepApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['salesReps'] });
      toast.success(isEdit ? 'Sales rep has been updated' : 'Sales rep has been created');
      form.reset();
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || isEdit
          ? 'Failed to update sales rep'
          : 'Failed to create sales rep'
      );
    },
  });

  const onSubmit = (data: FormData) => {
    mutation.mutate(getValuable(data));
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <div className="grid grid-cols-2 gap-4">
          <FormTextField
            form={form}
            name="first_name"
            label="First name"
            withAsterisk
            placeholder="John"
          />
          <FormTextField
            form={form}
            name="last_name"
            label="Last name"
            withAsterisk
            placeholder="Doe"
          />
        </div>
        <FormTextField
          form={form}
          name="email"
          label="Email"
          withAsterisk
          placeholder="email@example.com"
          type="email"
        />
        <FormTextField
          form={form}
          name="calendar_link"
          label="Calendar link"
          placeholder="https://calendar.example.com"
        />
        <MainButton text="Submit" className="w-full" loading={mutation.isPending} />
      </form>
    </Form>
  );
}

interface CreateSalesRepDialogProps extends PropsWithChildren {
  isEdit?: boolean;
  data?: FormData;
  open: boolean;
  onOpenChange: (opened: boolean) => void;
}

export function CreateSalesRepDialog({
  children,
  isEdit = false,
  data,
  open,
  onOpenChange,
}: CreateSalesRepDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? 'Edit a Sales Representative' : 'Add a New Sales Representative'}
          </DialogTitle>
          {!isEdit && (
            <DialogDescription>
              Set up a new sales representative profile to manage leads, track performance, and
              streamline sales activities. Fill in the required details to get started.
            </DialogDescription>
          )}
        </DialogHeader>
        <CreateSalesRepForm onSuccess={() => onOpenChange(false)} isEdit={isEdit} data={data} />
      </DialogContent>
    </Dialog>
  );
}
