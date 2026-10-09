import { type UseFormReturn } from 'react-hook-form';
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

type SelectOption = {
  value: string;
  label: string;
  textOptional?: string;
  icon?: JSX.Element;
};

interface FormSelectFieldProps {
  form: UseFormReturn<any>;
  name: string;
  label?: string;
  description?: string | JSX.Element;
  withAsterisk?: boolean;
  placeholder?: string;
  disabled?: boolean;
  options: SelectOption[];
}

export function FormSelectField({
  form,
  name,
  label,
  description,
  withAsterisk = false,
  placeholder,
  disabled = false,
  options = [],
}: FormSelectFieldProps) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => {
        return (
          <FormItem>
            {label && (
              <FormLabel>
                {label}{' '}
                {withAsterisk && <span className="text-destructive">*</span>}
              </FormLabel>
            )}
            <Select
              onValueChange={field.onChange}
              defaultValue={field.value}
              disabled={disabled}
            >
              <FormControl>
                <SelectTrigger>
                  <SelectValue placeholder={placeholder} />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                {options.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    <div className="flex items-center gap-2">
                      {option.icon && option.icon}{' '}
                      {option.textOptional ? (
                        <>
                          {option.label}{' '}
                          <span className="text-muted-foreground">
                            {option.textOptional}
                          </span>
                        </>
                      ) : (
                        option.label
                      )}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {description && <FormDescription>{description}</FormDescription>}
            <FormMessage />
          </FormItem>
        );
      }}
    />
  );
}
