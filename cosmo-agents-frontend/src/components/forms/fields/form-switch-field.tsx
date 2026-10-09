import { UseFormReturn } from 'react-hook-form';
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Switch } from '@/components/ui/switch';

interface FormSwitchFieldProps {
  form: UseFormReturn<any>;
  name: string;
  label?: string;
  description?: string;
  withAsterisk?: boolean;
}

function FormSwitchField({
  form,
  name,
  label,
  description,
  withAsterisk = false,
}: FormSwitchFieldProps) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <div>
            {label && (
              <FormLabel>
                {label}{' '}
                {withAsterisk && <span className="text-destructive">*</span>}
              </FormLabel>
            )}
            {description && <FormDescription>{description}</FormDescription>}
          </div>
          <FormControl>
            <Switch checked={field.value} onCheckedChange={field.onChange} />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

export { FormSwitchField };
