'use client';

import {
  ActionIcon,
  Box,
  Button,
  Checkbox,
  Combobox,
  ComboboxItem,
  Group,
  Input,
  Loader,
  Radio,
  rem,
  Stack,
  Text,
  useCombobox,
} from '@mantine/core';
import { Check, Search } from 'lucide-react';
import React, { useEffect, useState } from 'react';

import { IconPlusCircle } from '@/assets/icons';
import classes from './SelectMenu.module.css';

export type AddFn = (() => void) | (() => Promise<void>);

interface SelectMenuProps extends React.PropsWithChildren {
  value: string[];
  data: ComboboxItem[];
  onChange: (value: string[]) => void;
  onSubmit: (value: string[]) => void;
  fetchFn: () => Promise<any>;
  afterFetch: (data: any) => void;
  title: string;
  mode: 'checkbox' | 'radio';
  addFn: AddFn;
  isCustomSubmit?: boolean;
  textSubmit?: string;
  renderItem?: (item: ComboboxItem, idx: number) => React.ReactNode;
}

function RadioOption({ item, renderItem, index }: { item: ComboboxItem, renderItem?: (item: ComboboxItem, idx: number) => React.ReactNode, index: number }) {
  return (
    <Radio.Card value={item.value} withBorder={false} className={classes.radioCard}>
      <Group wrap="nowrap" justify="space-between">
        <Group wrap="nowrap" gap="sm">
          {renderItem ? renderItem(item, index) : <Text fw={500} lineClamp={1} title={item.label ?? item.value}>
            {item.label ?? item.value}
          </Text>}
        </Group>
        <Radio.Indicator variant="outline" color="#0085FF" classNames={classes} />
      </Group>
    </Radio.Card>
  );
}

function CheckboxOption({ item, renderItem, index }: { item: ComboboxItem, renderItem?: (item: ComboboxItem, idx: number) => React.ReactNode, index: number }) {
  return (
    <Checkbox.Card value={item.value} withBorder={false} className={classes.radioCard}>
      <Group wrap="nowrap" justify="space-between">
        <Group wrap="nowrap" gap="sm">
          {renderItem ? renderItem(item, index) : <Text fw={500} lineClamp={1} title={item.label ?? item.value}>
            {item.label ?? item.value}
          </Text>}
        </Group>
        <Checkbox.Indicator color="#0085FF" radius="sm" />
      </Group>
    </Checkbox.Card>
  );
}

function SelectMenu({
  value: _value = [],
  data: _data = [],
  onChange,
  onSubmit,
  fetchFn,
  isCustomSubmit = false,
  textSubmit = 'Submit',
  renderItem,
  afterFetch,
  title,
  mode = 'checkbox',
  addFn,
  children,
}: Partial<SelectMenuProps>) {
  const child = React.Children.only(children) as React.ReactElement<any>;

  const [search, setSearch] = useState('');
  const [value, setValue] = useState<string[]>(_value);
  const [data, setData] = useState<ComboboxItem[]>(_data);
  const [isLoading, setIsLoading] = useState(false);

  const combobox = useCombobox({
    onDropdownClose: () => combobox.resetSelectedOption(),
    onDropdownOpen: () => {
      if (data.length === 0 && !isLoading) {
        setIsLoading(true);
        if (!isCustomSubmit) {
          fetchFn?.().then((response: any) => {
            afterFetch?.(response);
            setIsLoading(false);
            combobox.resetSelectedOption();
          });
        } else {
          setIsLoading(false);
        }
      }
    },
  });

  const options = data
    .filter((item) =>
      (item.label ?? item.value).toLowerCase().includes(search.toLowerCase().trim())
    )
    .map((item, index) => (
      <Combobox.Option value={item.value} key={item.value}>
        {mode === 'checkbox' && <CheckboxOption item={item} renderItem={renderItem} index={index} />}
        {mode === 'radio' && <RadioOption item={item} renderItem={renderItem} index={index} />}
      </Combobox.Option>
    ));

  useEffect(() => {
    setData(_data);
  }, [_data]);

  useEffect(() => {
    setValue(_value);
  }, [_value]);

  useEffect(() => {
    if (isCustomSubmit) {
      const handler = setTimeout(() => {
        onChange?.(value);
      }, 300);
      return () => clearTimeout(handler);
    }
  }, [value, isCustomSubmit]);

  const handleSubmit = () => {
    combobox.closeDropdown();
    onSubmit?.(value);
  };

  return (
    <Combobox
      store={combobox}
      width={300}
      dropdownPadding={8}
      withinPortal={false}
      // onOptionSubmit={(val) => {}}
      classNames={classes}
    >
      <Combobox.Target>
        {React.cloneElement(child, { onClick: combobox.toggleDropdown })}
      </Combobox.Target>

      <Combobox.Dropdown>
        <Stack gap="xs">
          {title && (
            <Text fw={600} ta="center">
              {title}
            </Text>
          )}
          <Input
            variant="filled"
            placeholder="Search"
            leftSection={<Search style={{ width: rem(16), height: rem(16) }} />}
            value={search}
            onChange={(event) => setSearch(event.currentTarget.value)}
          />
          <Combobox.Options>
            {isLoading && (
              <Combobox.Empty>
                <Loader size={16} />
              </Combobox.Empty>
            )}
            {!isLoading && mode === 'checkbox' && (
              <Checkbox.Group
                value={value}
                onChange={(val) => {
                  setValue(val);
                  onChange?.(val);
                }}
              >
                <Stack gap="xs">
                  {options.length === 0 ? <Combobox.Empty>Nothing found</Combobox.Empty> : options}
                </Stack>
              </Checkbox.Group>
            )}
            {!isLoading && mode === 'radio' && (
              <Radio.Group
                value={value[0]}
                onChange={(val) => {
                  setValue([val]);
                  onChange?.([val]);
                }}
              >
                <Stack gap="xs">
                  {options.length === 0 ? <Combobox.Empty>Nothing found</Combobox.Empty> : options}
                </Stack>
              </Radio.Group>
            )}
          </Combobox.Options>
          <Group wrap="nowrap" gap="xs">
            {addFn && (
              <Button
                color="gray"
                leftSection={<IconPlusCircle />}
                onClick={addFn}
                variant="light"
                fullWidth
              >
                Add more
              </Button>
            )}
            {onSubmit && !isCustomSubmit && (
              <ActionIcon variant="light" onClick={handleSubmit}>
                <Check width={20} height={20} />
              </ActionIcon>
            )}
          </Group>
          {mode === 'checkbox' && value.length > 0 && !isCustomSubmit && (
            <Box ta="center">
              <Text
                span
                onClick={() => {
                  setValue([]);
                  onChange?.([]);
                }}
                className={classes.deselect}
              >
                Deselect ({value.length})
              </Text>
            </Box>
          )}
          {isCustomSubmit && (
            <Button
              color="gray"
              className='hover:bg-[#E6D6FF] transition-bg duration-300'
              leftSection={<IconPlusCircle />}
              onClick={handleSubmit}
              variant="light"
              fullWidth
            >
              {textSubmit}
            </Button>
          )}
        </Stack>
      </Combobox.Dropdown>
    </Combobox>
  );
}

export default SelectMenu;
