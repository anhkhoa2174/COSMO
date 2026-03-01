'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { DialogProps } from '@radix-ui/react-dialog';
import { useMutation } from '@tanstack/react-query';
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
  // DialogTrigger,
} from '@/components/ui/dialog';
import { Form } from '@/components/ui/form';
import { getValuable } from '@/lib/utils';
import ContactApi from '@/network/client/contact';
import { FormTextField } from '@/components/forms/fields/form-text-field';
import { CustomField } from '@/models/custom-field';
import { FormSelectField } from '@/components/forms/fields/form-select-field';
import { FormNumberField } from '@/components/forms/fields/form-number-field';
import { FormDatePickerField } from '@/components/forms/fields/form-date-picker-field';
import { DynamicFieldsInput } from '@/components/forms/fields/dynamic-fields-input';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Image } from 'lucide-react';
import { ExtractFromScreenshotDialog } from '@/components/forms/extract-from-screenshot-dialog';

interface Field {
  normalized_name: string;
  name: string;
  data_type: string;
  is_required: boolean;
  options?: string[];
}

interface CreateContactFormProps {
  data?: any;
  isEdit?: boolean;
  onSuccess: () => void;
  customFields: CustomField[];
}

const systemFields = [
  {
    normalized_name: 'name',
    name: 'Name',
    data_type: 'text',
    is_required: true,
  },
  {
    normalized_name: 'company',
    name: 'Company',
    data_type: 'text',
    is_required: true,
  },
  {
    normalized_name: 'job_title',
    name: 'Job Title',
    data_type: 'text',
    is_required: true,
  },
  {
    normalized_name: 'source',
    name: 'Source',
    data_type: 'select',
    is_required: true,
    options: ['LinkedIn', 'Email', 'Referral', 'Website', 'Event', 'Other'],
  },
  {
    normalized_name: 'contact_information',
    name: 'Contact Info (LinkedIn URL or Email)',
    data_type: 'text',
    is_required: true,
  },
  {
    normalized_name: 'industry',
    name: 'Industry',
    data_type: 'text',
    is_required: false,
  },
] as Field[];

export function CreateContactForm({
  data,
  isEdit = false,
  onSuccess,
  customFields,
}: CreateContactFormProps) {
  // Extract additional fields from profile.custom_fields (AI-like format)
  const getAdditionalFieldsFromData = (contactData: any) => {
    if (!contactData) {
      return {};
    }

    const profile = contactData.profile || {};
    const customFields = profile.custom_fields || {};

    // Convert from AI format {field: {value, source, updated_at}} to simple {field: value}
    const additionalFields: Record<string, any> = {};
    Object.entries(customFields).forEach(([key, fieldData]: [string, any]) => {
      // Skip system fields like 'id'
      if (key === 'id') {
        return;
      }

      // Extract value from nested structure
      if (fieldData && typeof fieldData === 'object' && 'value' in fieldData) {
        additionalFields[key] = fieldData.value;
      } else {
        // Fallback for simple values
        additionalFields[key] = fieldData;
      }
    });

    return additionalFields;
  };

  const _customFields = customFields.map(
    ({ normalized_name, name, data_type, is_required, options }) => ({
      normalized_name,
      name,
      data_type,
      is_required,
      options,
    })
  ) as Field[];

  const [additionalFields, setAdditionalFields] = useState<Record<string, any>>(
    getAdditionalFieldsFromData(data)
  );

  const normalizeAdditionalValue = (value: any) => {
    if (value === null || value === undefined) return '';
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      return String(value);
    }
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  };

  const toDisplayLabel = (rawKey: string) => {
    const withSpaces = rawKey
      .replace(/_/g, ' ')
      .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
      .replace(/\s+/g, ' ')
      .trim();
    return withSpaces.replace(/\b\w/g, (c) => c.toUpperCase());
  };

  const handleScreenshotExtract = async (extractedData: Record<string, any>) => {
    const knownFields = new Set(
      [...systemFields, ..._customFields].map((field) => field.normalized_name)
    );

    const nextAdditionalFields: Record<string, any> = { ...additionalFields };
    let filledCount = 0;
    let extraCount = 0;

    Object.entries(extractedData).forEach(([key, value]) => {
      if (value === null || value === undefined || value === '') {
        return;
      }

      const normalizedKey = key.toLowerCase();
      if (knownFields.has(normalizedKey)) {
        form.setValue(normalizedKey as any, value);
        filledCount += 1;
        return;
      }

      const normalized = normalizeAdditionalValue(value);
      if (normalized !== '') {
        const labelKey = toDisplayLabel(key);
        nextAdditionalFields[labelKey] = normalized;
        extraCount += 1;
      }
    });

    if (extraCount > 0) {
      setAdditionalFields(nextAdditionalFields);
    }

    toast.success(
      `Auto-filled ${filledCount} fields and added ${extraCount} additional field(s)`
    );
  };

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
    ..._customFields.reduce((acc, field) => {
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
        if (field.data_type === 'select') {
          return field.is_required
            ? z.enum(field.options as [string, ...string[]])
            : z.enum(field.options as [string, ...string[]]).optional();
        }
        if (field.data_type === 'number') {
          return field.is_required ? z.number() : z.number().optional();
        }
        if (field.data_type === 'date') {
          return field.is_required ? z.date() : z.date().optional();
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
    mutationFn: isEdit
      ? (_data: FormData) => ContactApi.update(data?.id as string, _data)
      : ContactApi.create,
    onSuccess: () => {
      toast.success(
        isEdit ? 'Contact has been updated' : 'Contact has been created'
      );
      form.reset();
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Contact has not been created'
      );
    },
  });

  const onSubmit = (data: FormData) => {
    // Merge form data with additional fields at root level
    // Backend will capture them as ExtraFields and store in Profile JSONB
    const formData = getValuable(data);

    // Remove custom_fields if it exists (not a valid backend field)
    const { custom_fields, ...cleanFormData } = formData as any;

    // Filter out system/special fields from additionalFields
    const { id, ...cleanAdditionalFields } = additionalFields;

    const payload = {
      ...cleanFormData,
      ...cleanAdditionalFields, // Spread additional fields at root level
    };

    mutation.mutate(payload);
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

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        {!isEdit && (
          <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium text-blue-900 dark:text-blue-100">
                  Quick Fill from Screenshot
                </p>
                <p className="text-sm text-blue-700 dark:text-blue-300">
                  Upload a screenshot of a profile to auto-fill contact information
                </p>
              </div>
              <ExtractFromScreenshotDialog
                onExtract={handleScreenshotExtract}
                trigger={
                  <Button variant="outline" size="sm" type="button">
                    <Image className="mr-2 h-4 w-4" />
                    Upload Screenshot
                  </Button>
                }
              />
            </div>
          </div>
        )}
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <p className="text-base font-semibold uppercase">System fields</p>
          </div>
          <div className="grid grid-cols-2 gap-4">
            {renderFields(systemFields)}
          </div>
        </div>
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <p className="text-base font-semibold uppercase">Custom fields</p>
          </div>
          <div className="grid grid-cols-2 gap-4">
            {renderFields(_customFields)}
          </div>
          {_customFields.length === 0 && (
            <p className="text-center text-sm text-muted-foreground">
              No custom fields found
            </p>
          )}
        </div>
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <p className="text-base font-semibold uppercase">Additional fields</p>
            <p className="text-xs text-muted-foreground">(specific to this contact)</p>
          </div>
          <DynamicFieldsInput
            value={additionalFields}
            onChange={setAdditionalFields}
            description="Add custom fields that are unique to this contact only"
          />
        </div>
        <MainButton
          text="Submit"
          className="w-full"
          loading={mutation.isPending}
        />
      </form>
    </Form>
  );
}

interface CreateContactDialogProps
  extends CreateContactFormProps,
    DialogProps {}

export function CreateContactDialog({
  data,
  isEdit = false,
  onSuccess,
  customFields,
  ...props
}: CreateContactDialogProps) {
  return (
    <Dialog {...props}>
      {/* <DialogTrigger asChild>{children}</DialogTrigger> */}
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-screen-lg">
        <DialogHeader>
          <DialogTitle>Add a New Contact</DialogTitle>
          <DialogDescription>
            Manually add a new contact to your list. Provide the essential
            details to keep your records up-to-date and stay connected
            effortlessly.
          </DialogDescription>
        </DialogHeader>
        <CreateContactForm
          onSuccess={onSuccess}
          data={data}
          isEdit={isEdit}
          customFields={customFields}
        />
      </DialogContent>
    </Dialog>
  );
}
