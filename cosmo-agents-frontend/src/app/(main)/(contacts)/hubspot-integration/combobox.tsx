'use client';

import * as React from 'react';

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
import { MainButton } from '@/components/buttons/main-button';
import { HubspotProperty } from '@/network/client/hubspot';

interface ComboboxProps {
  data: HubspotProperty[];
  onSelect: (value: string) => void;
}

export function ComboboxDemo({ data, onSelect }: ComboboxProps) {
  const [open, setOpen] = React.useState(false);

  const _data = React.useMemo(
    () =>
      data.reduce(
        (acc, item) => {
          const group = item.groupName;
          if (!acc[group]) {
            acc[group] = [];
          }
          acc[group].push(item);
          return acc;
        },
        {} as { [key: string]: HubspotProperty[] }
      ),
    [data]
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <MainButton
          variant="outline"
          text="+ Add custom field mapping"
          aria-expanded={open}
          className="w-full"
          role="combobox"
        />
      </PopoverTrigger>
      <PopoverContent className="p-0">
        <Command>
          <CommandInput placeholder="Search field..." className="h-9" />
          <CommandList>
            <CommandEmpty>No framework found.</CommandEmpty>
            {Object.entries(_data).map(([group, items]) => (
              <CommandGroup key={group} heading={group}>
                {items.map((item) => (
                  <CommandItem
                    key={item.name}
                    value={item.name}
                    onSelect={(currentValue) => {
                      onSelect(currentValue);
                      setOpen(false);
                    }}
                    keywords={[item.name, item.label]}
                  >
                    {item.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
