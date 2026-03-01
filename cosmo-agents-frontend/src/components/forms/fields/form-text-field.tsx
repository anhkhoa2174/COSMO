import * as React from 'react';
import { type FieldValues, type Path, type UseFormReturn } from 'react-hook-form';
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';

type FormTextFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TContext = any,
  TTransformedValues extends FieldValues | undefined = undefined,
> = {
  form: UseFormReturn<TFieldValues, TContext, TTransformedValues>;
  name: Path<TFieldValues>;
  label?: string;
  description?: string;
  withAsterisk?: boolean;
} & Omit<React.ComponentPropsWithoutRef<typeof Input>, 'form' | 'name'>;

export function FormTextField<
  TFieldValues extends FieldValues = FieldValues,
  TContext = any,
  TTransformedValues extends FieldValues | undefined = undefined,
>({
  form,
  name,
  label,
  description,
  withAsterisk = false,
  ...inputProps
}: FormTextFieldProps<TFieldValues, TContext, TTransformedValues>) {
  return (
    <FormField
      control={form.control as any}
      name={name}
      render={({ field }) => (
        <FormItem>
          {label && (
            <FormLabel>
              {label} {withAsterisk && <span className="text-destructive">*</span>}
            </FormLabel>
          )}
          <FormControl>
            <Input {...inputProps} {...field} />
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
