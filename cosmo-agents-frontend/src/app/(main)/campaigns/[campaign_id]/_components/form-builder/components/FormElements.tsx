'use client';

import { DatetimePicker } from '@/components/datetime-picker';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';
import { format } from 'date-fns';
import { CalendarIcon } from 'lucide-react';
import Link from 'next/link';
import React from 'react';
import { Controller } from 'react-hook-form';
import { DividerElement, FormElement, SpacerElement } from './types';

interface FormElementRendererProps {
  element: FormElement;
  inBuilder?: boolean;
  onSelect?: () => void;
  isSelected?: boolean;
  error?: any;
  form?: any;
}

const SpacerElementRenderer: React.FC<{ element: SpacerElement }> = ({
  element,
}) => {
  const styles = element.styles || {}; // Fallback to empty object if styles is undefined
  return (
    <div
      style={{
        height: styles?.height || '16px',
        backgroundColor: styles?.backgroundColor || 'transparent',
      }}
      aria-label={`Spacer of height ${styles?.height || '16px'}`}
    />
  );
};

const DividerElementRenderer: React.FC<{ element: DividerElement }> = ({
  element,
}) => {
  const styles = element.styles || {}; // Fallback to empty object
  const dividerStyle: React.CSSProperties = {
    borderTopStyle: 'solid',
    borderTopColor: styles.color || '#e0e0e0',
    borderTopWidth: styles.thickness || '1px',
    width: styles.width || '100%',
    marginTop: styles.marginTop || '8px',
    marginBottom: styles.marginBottom || '8px',
  };
  return (
    <div className="flex w-full justify-center">
      {' '}
      {/* Outer div to respect width style for centering if width is not 100% */}
      <div style={dividerStyle} aria-label="Divider" />{' '}
      {/* Changed hr to div for more style control */}
    </div>
  );
};

export function FormElementRenderer({
  element,
  inBuilder = false,
  onSelect,
  isSelected = false,
  error,
  form,
}: FormElementRendererProps) {
  const boxShadow = isSelected
    ? 'shadow-[0_25px_20px_-20px_rgba(0,0,0,0.30)] bg-white'
    : '';
  const commonClasses = cn(
    'w-full transition-shadow duration-300',
    element.className,
    inBuilder && 'cursor-move',
    boxShadow
  );

  const handleClick = (e: React.MouseEvent) => {
    if (inBuilder && onSelect) {
      e.stopPropagation();
      onSelect();
    }
  };

  const renderDescription = () => {
    if (error)
      return (
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm text-destructive">{error}</p>{' '}
          <p className="text-sm text-muted-foreground">{element.description}</p>
        </div>
      );
    if (!element.description) return null;
    return (
      <p className="text-sm text-muted-foreground">{element.description}</p>
    );
  };

  const renderLabel = () => {
    return (
      <Label>
        {element.label}
        {element.required && <span className="ml-1 text-destructive">*</span>}
      </Label>
    );
  };

  switch (element.type) {
    case 'text':
    case 'email':
    case 'password':
    case 'number':
      return (
        <div className={cn('space-y-2', commonClasses)} onClick={handleClick}>
          {renderLabel()}
          <Input
            className={cn(error ? 'border-red-500' : '')}
            type={element.type}
            placeholder={element.placeholder}
            defaultValue={element.defaultValue}
            disabled={inBuilder}
            {...form?.register(element.name)}
          />
          {renderDescription()}
        </div>
      );
    case 'url':
      return (
        <div className={cn('flex flex-col space-y-2', commonClasses)}>
          {renderLabel()}
          <Link
            href={element.defaultValue}
            target="_"
            {...form?.register(element.name)}
          >
            <span
              className={cn(
                error ? 'border-red-500 underline' : 'text-[#4F46E5] underline'
              )}
            >
              {element.defaultValue}
            </span>
          </Link>
          {renderDescription()}
        </div>
      );

    case 'textarea':
      return (
        <div className={cn('space-y-2', commonClasses)} onClick={handleClick}>
          {renderLabel()}
          <Textarea
            className={cn('max-h-[200px]', error ? 'border-red-500' : '')}
            placeholder={element.placeholder}
            defaultValue={element.defaultValue}
            rows={element.rows || 3}
            disabled={inBuilder}
            {...form?.register(element.name)}
          />
          {renderDescription()}
        </div>
      );

    case 'select':
      return (
        <div className={cn('space-y-2', commonClasses)} onClick={handleClick}>
          {renderLabel()}
          <Controller
            name={element.name}
            control={form.control}
            defaultValue={element.defaultValue}
            render={({ field }) => (
              <Select
                {...field}
                disabled={inBuilder}
                defaultValue={element.defaultValue}
                onValueChange={(selected) => field.onChange(selected)}
              >
                <SelectTrigger className={cn(error ? 'border-red-500' : '')}>
                  <SelectValue
                    placeholder={element.placeholder || 'Select an option'}
                  />
                </SelectTrigger>
                <SelectContent>
                  {element.options.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
          {renderDescription()}
        </div>
      );

    case 'checkbox':
      return (
        <Controller
          name={element.name}
          control={form.control}
          defaultValue={!!element.defaultValue}
          render={({ field }) => (
            <div
              className={cn(
                'flex items-start space-x-3 space-y-0',
                commonClasses
              )}
              onClick={handleClick}
            >
              <Checkbox
                {...field}
                className={cn(error ? 'border-red-500' : '')}
                defaultChecked={!!element.defaultValue}
                checked={field.value}
                onCheckedChange={(checked) => field.onChange(checked)}
                disabled={inBuilder}
              />
              <div className="space-y-1 leading-none">
                <Label>
                  {element.label}
                  {element.required && (
                    <span className="ml-1 text-destructive">*</span>
                  )}
                </Label>
                {renderDescription()}
              </div>
            </div>
          )}
        />
      );

    case 'radio':
      return (
        <div className={cn('space-y-2', commonClasses)} onClick={handleClick}>
          {renderLabel()}
          <Controller
            name={element.name}
            control={form.control}
            defaultValue={element.defaultValue}
            render={({ field }) => (
              <RadioGroup
                {...field}
                disabled={inBuilder}
                defaultValue={element.defaultValue}
                value={field.value}
                onValueChange={(selected) => {
                  field.onChange(selected);
                }}
              >
                {element.options.map((option) => (
                  <div
                    key={option.value}
                    className="flex items-center space-x-2"
                  >
                    <RadioGroupItem
                      value={option.value}
                      id={`${element.id}-${option.value}`}
                    />
                    <Label htmlFor={`${element.id}-${option.value}`}>
                      {option.label}
                    </Label>
                  </div>
                ))}
              </RadioGroup>
            )}
          />
          {renderDescription()}
        </div>
      );

    case 'date':
      return (
        <div className={cn('space-y-2', commonClasses)} onClick={handleClick}>
          {renderLabel()}
          <Controller
            name={element.name}
            control={form.control}
            defaultValue={
              element.defaultValue ? new Date(element.defaultValue) : null
            }
            render={({ field }) => (
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      'w-full pl-3 text-left font-normal',
                      !element.defaultValue && 'text-muted-foreground',
                      error ? 'border-red-500' : ''
                    )}
                    disabled={inBuilder}
                  >
                    {field.value ? (
                      format(new Date(field.value), 'PPP')
                    ) : (
                      <span>{element.placeholder || 'Pick a date'}</span>
                    )}
                    <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <DatetimePicker
                    {...field}
                    value={field.value}
                    onChange={(selected) => {
                      form.setValue(element.name, selected);
                    }}
                  />
                </PopoverContent>
              </Popover>
            )}
          />
          {renderDescription()}
        </div>
      );

    case 'spacer':
      return <SpacerElementRenderer element={element as SpacerElement} />;

    case 'divider':
      return <DividerElementRenderer element={element as DividerElement} />;

    default:
      return <div>Unsupported element type</div>;
  }
}
