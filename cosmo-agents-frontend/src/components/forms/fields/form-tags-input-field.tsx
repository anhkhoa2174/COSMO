import { type UseFormReturn } from 'react-hook-form';
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { TagsInput } from '@/components/ui/tags-input';

interface FormTagsInputProps {
  form: UseFormReturn<any>;
  name: string;
  label?: string;
  required?: boolean;
  placeholder?: string;
  description?: string | JSX.Element;
  readOnly?: boolean;
  data?: string[];
}

export function FormTagsInputField({
  form,
  name,
  label,
  required = false,
  placeholder,
  description,
  readOnly = false,
  data,
}: FormTagsInputProps) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          {label && (
            <FormLabel>
              {label} {required && <span className="text-destructive">*</span>}
            </FormLabel>
          )}
          <FormControl>
            <TagsInput
              onValueChange={field.onChange}
              value={field.value}
              placeholder={placeholder}
              readOnly={readOnly}
              data={data}
            />
          </FormControl>
          {description && <FormDescription>{description}</FormDescription>}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}
