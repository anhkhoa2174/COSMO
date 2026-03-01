'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { toast } from 'sonner';
import { z } from 'zod';
import { MainButton } from '@/components/buttons/main-button';
import { FormTagsInputField } from '@/components/forms/fields/form-tags-input-field';
import { FormTextField } from '@/components/forms/fields/form-text-field';
import { FormTextareaField } from '@/components/forms/fields/form-textarea-field';
import { ContentLayout } from '@/components/nav/content-layout';
import { Form } from '@/components/ui/form';
import { getValuable, validURL } from '@/lib/utils';
import OrganizationApi from '@/network/client/organization';
import { Skeleton } from '@/components/ui/skeleton';
import { useEffect } from 'react';

const FormSchema = z.object({
  name: z.string().nonempty(),
  company_url: z.string().refine((value) => !value || validURL(value), {
    message: 'Invalid url',
  }),
  company_description: z.string().nonempty(),
  company_targeting_persona: z.array(z.string()).nonempty(),
  value_offering: z.string().nonempty(),
});

type FormData = z.infer<typeof FormSchema>;

export default function CompanyInfoPage() {
  const { data, isLoading, isSuccess, refetch } = useQuery({
    queryKey: ['organizations'],
    queryFn: () => OrganizationApi.list(),
  });

  const form = useForm<FormData>({
    resolver: zodResolver(FormSchema),
  });

  useEffect(() => {
    if (isSuccess) {
      form.reset(data.data.list[0]);
    }
  }, [isSuccess, data]);

  const { mutate, isPending } = useMutation({
    mutationFn: (_data: FormData) =>
      OrganizationApi.update(data.data.list[0].id, _data),
    onSuccess: () => {
      refetch();
      toast.success('Company information updated successfully');
    },
    onError: (err: any) => {
      toast.error(err.message);
    },
  });

  const onSubmit = (data: FormData) => {
    mutate(getValuable(data));
  };

  const rightSection = form.formState.isDirty && (
    <MainButton
      text="Save changes"
      loading={isPending}
      onClick={() => form.handleSubmit(onSubmit)()}
    />
  );

  return (
    <ContentLayout title="Company Information" rightSection={rightSection}>
      {isLoading ? (
        <div className="grid grid-cols-4 gap-4">
          <div className="col-span-1">
            <Skeleton className="h-6" />
          </div>
          <div className="col-span-3 space-y-6">
            <Skeleton className="h-10" />
            <Skeleton className="h-10" />
            <Skeleton className="h-32" />
          </div>
        </div>
      ) : data ? (
        <Form {...form}>
          <form className="space-y-6">
            <div className="grid grid-cols-4 gap-4">
              <div className="col-span-1">
                <p className="text-lg font-semibold">Company Profile</p>
                <p className="text-muted-foreground">
                  Set your company details
                </p>
              </div>
              <div className="col-span-3 space-y-6">
                <FormTextField
                  form={form}
                  name="name"
                  label="Company name"
                  withAsterisk
                />
                <FormTextField
                  form={form}
                  name="company_url"
                  label="Company url"
                  withAsterisk
                />
              </div>
            </div>
            <div className="grid grid-cols-4 gap-4">
              <div className="col-span-1">
                <p className="text-lg font-semibold">Company Information</p>
                <p className="text-muted-foreground">
                  Set your company details
                </p>
              </div>
              <div className="col-span-3 space-y-6">
                <FormTextareaField
                  form={form}
                  name="company_description"
                  label="Description"
                  withAsterisk
                  rows={6}
                />
                <FormTagsInputField
                  form={form}
                  name="company_targeting_persona"
                  label="Target persona"
                  data={[
                    'Digital Transformation Team at Enterprises',
                    'Marketing Manager at Series A Tech Companies',
                    'Marketing Manager at Educational Companies',
                  ]}
                  required
                  description="Press Enter to add a target"
                />
                <FormTextareaField
                  form={form}
                  name="value_offering"
                  label="Value offering"
                  withAsterisk
                  rows={6}
                />
              </div>
            </div>
          </form>
        </Form>
      ) : null}
    </ContentLayout>
  );
}
