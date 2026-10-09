'use client';

import * as React from 'react';
import { Check, ChevronDown } from 'lucide-react';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';

type ComboboxItem = {
  value: string;
  label: string;
  group?: string;
};

interface ComboboxProps {
  value: string;
  onChange: (value: string) => void;
  data?: ComboboxItem[];
  placeholder?: string;
  className?: string;
  withInput?: boolean;
}

export function Combobox({
  value,
  onChange,
  data = [],
  placeholder,
  className = '',
  withInput = true,
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn(
            'w-full justify-between gap-1 bg-transparent px-2 font-normal',
            className
          )}
        >
          <span className="truncate">
            {value
              ? data.find((dt) => dt.value === value)?.label
              : placeholder || 'Select option...'}
          </span>
          <ChevronDown className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0">
        <Command>
          {withInput && (
            <CommandInput placeholder="Search ..." className="h-9" />
          )}
          <CommandList>
            <CommandEmpty>No found.</CommandEmpty>
            <CommandGroup>
              {data.map((dt) => (
                <CommandItem
                  key={dt.value}
                  value={dt.value}
                  onSelect={(currentValue) => {
                    onChange(currentValue === value ? '' : currentValue);
                    setOpen(false);
                  }}
                  keywords={[dt.label, dt.value]}
                >
                  {dt.label}
                  <Check
                    className={cn(
                      'ml-auto',
                      value === dt.value ? 'opacity-100' : 'opacity-0'
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
