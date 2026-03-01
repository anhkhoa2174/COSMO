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

type FormNumberFieldProps<
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

export function FormNumberField<
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
}: FormNumberFieldProps<TFieldValues, TContext, TTransformedValues>) {
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
            <Input {...inputProps} type="number" min={0} {...field} onChange={event => field.onChange(+event.target.value)} />
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
