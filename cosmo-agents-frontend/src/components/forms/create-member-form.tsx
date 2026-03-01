'use client';

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
import { inviteRedirectUri } from '@/helpers/env';
import { getValuable } from '@/lib/utils';
import OrganizationApi from '@/network/client/organization';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { PropsWithChildren, useState } from 'react';
import { useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { z } from 'zod';
import { FormSelectField } from './fields/form-select-field';
import { FormTextField } from './fields/form-text-field';

const FormSchema = z.object({
  email: z.string().email({
    message: 'Invalid email address.',
  }),
  name: z.string().nonempty(),
  job_title: z.string().nonempty(),
  role: z.enum(['member', 'admin']),
  redirect_uri: z.string(),
});

type FormData = z.infer<typeof FormSchema>;

export function CreateMemberForm({ onSuccess }: { onSuccess: () => void }) {
  const queryClient = useQueryClient();
  const form = useForm<FormData>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      role: 'member',
      redirect_uri: inviteRedirectUri,
    },
  });
  const mutation = useMutation({
    mutationFn: (data: FormData) => OrganizationApi.createMember('me', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members'] });
      toast.success('Member has been invited');
      form.reset();
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Member has not been invited'
      );
    },
  });

  const onSubmit = (data: FormData) => {
    mutation.mutate(getValuable(data));
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <FormTextField
          form={form}
          name="email"
          label="Email"
          required
          placeholder="email@example.com"
          type="email"
        />
        <FormTextField
          form={form}
          name="name"
          label="Name"
          withAsterisk
          placeholder="John Doe"
        />
        <FormTextField
          form={form}
          name="job_title"
          label="Job title"
          placeholder="Sales Engineer"
          withAsterisk
        />
        <FormSelectField
          form={form}
          name="role"
          label="Role"
          placeholder="Select role"
          withAsterisk
          options={[
            { value: 'member', label: 'Member' },
            { value: 'admin', label: 'Admin' },
          ]}
        />
        <MainButton
          text="Invite"
          className="w-full"
          loading={mutation.isPending}
        />
      </form>
    </Form>
  );
}

export function CreateMemberDialog({ children }: PropsWithChildren) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>Invite a New Member</DialogTitle>
          <DialogDescription>
            Add a new member to collaborate with your team. Enter their details
            and assign the appropriate role to ensure seamless participation.
          </DialogDescription>
        </DialogHeader>
        <CreateMemberForm onSuccess={() => setIsDialogOpen(false)} />
      </DialogContent>
    </Dialog>
  );
}
