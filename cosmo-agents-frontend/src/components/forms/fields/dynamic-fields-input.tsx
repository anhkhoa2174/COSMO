'use client';

import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { X, Plus } from 'lucide-react';

interface DynamicField {
  key: string;
  value: string;
}

interface DynamicFieldsInputProps {
  value?: Record<string, any>;
  onChange: (value: Record<string, any>) => void;
  label?: string;
  description?: string;
}

export function DynamicFieldsInput({
  value = {},
  onChange,
  label = 'Additional Fields',
  description,
}: DynamicFieldsInputProps) {
  const toDisplayLabel = (rawKey: string) => {
    const withSpaces = rawKey
      .replace(/_/g, ' ')
      .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
      .replace(/\s+/g, ' ')
      .trim();
    return withSpaces.replace(/\b\w/g, (c) => c.toUpperCase());
  };

  const toSlugKey = (label: string) => {
    return label
      .toLowerCase()
      .replace(/[^a-z0-9\s]/g, '')
      .trim()
      .replace(/\s+/g, '_');
  };

  const [fields, setFields] = useState<DynamicField[]>(() => {
    // Convert object to array of fields
    return Object.entries(value).map(([key, val]) => ({
      key: toDisplayLabel(key),
      value: String(val || ''),
    }));
  });

  // Sync state when value prop changes (e.g., when editing a contact)
  useEffect(() => {
    const newFields = Object.entries(value).map(([key, val]) => ({
      key: toDisplayLabel(key),
      value: String(val || ''),
    }));
    setFields(newFields);
  }, [value]);

  const [newFieldKey, setNewFieldKey] = useState('');
  const [newFieldValue, setNewFieldValue] = useState('');

  const handleAddField = () => {
    if (!newFieldKey.trim()) return;

    const updatedFields = [
      ...fields,
      { key: newFieldKey, value: newFieldValue },
    ];
    setFields(updatedFields);

    // Update parent form
    const newValue = updatedFields.reduce(
      (acc, field) => {
        if (field.key && field.value) {
          acc[toSlugKey(field.key)] = field.value;
        }
        return acc;
      },
      {} as Record<string, any>
    );

    onChange(newValue);

    // Reset inputs
    setNewFieldKey('');
    setNewFieldValue('');
  };

  const handleRemoveField = (index: number) => {
    const updatedFields = fields.filter((_, i) => i !== index);
    setFields(updatedFields);

    // Update parent form
    const newValue = updatedFields.reduce(
      (acc, field) => {
        if (field.key && field.value) {
          acc[toSlugKey(field.key)] = field.value;
        }
        return acc;
      },
      {} as Record<string, any>
    );
    onChange(newValue);
  };

  const handleUpdateField = (index: number, key: string, value: string) => {
    const updatedFields = [...fields];
    updatedFields[index] = { key, value };
    setFields(updatedFields);

    // Update parent form
    const newValue = updatedFields.reduce(
      (acc, field) => {
        if (field.key && field.value) {
          acc[toSlugKey(field.key)] = field.value;
        }
        return acc;
      },
      {} as Record<string, any>
    );
    onChange(newValue);
  };

  return (
    <div className="space-y-4">
      <div>
        <Label>{label}</Label>
        {description && (
          <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        )}
      </div>

      {/* Existing fields - styled like AI custom fields */}
      <div className="space-y-3">
        {fields.map((field, index) => (
          <div
            key={index}
            className="rounded-lg border border-gray-200 bg-white p-3"
          >
            <div className="mb-2 flex items-start justify-between gap-2">
              <Input
                placeholder="Field name"
                value={field.key}
                onChange={(e) =>
                  handleUpdateField(index, e.target.value, field.value)
                }
                className="h-auto border-0 p-0 font-medium text-gray-700 focus-visible:ring-0"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => handleRemoveField(index)}
                className="h-6 w-6 shrink-0"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <Input
              placeholder="Field value"
              value={field.value}
              onChange={(e) =>
                handleUpdateField(index, field.key, e.target.value)
              }
              className="h-auto border-0 p-0 text-sm text-gray-900 focus-visible:ring-0"
            />
          </div>
        ))}
      </div>

      {/* Add new field */}
      <div className="flex items-end gap-2">
        <div className="flex-1">
          <Input
            placeholder="New field name (e.g., LinkedIn URL)"
            value={newFieldKey}
            onChange={(e) => setNewFieldKey(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                handleAddField();
              }
            }}
          />
        </div>
        <div className="flex-1">
          <Input
            placeholder="Field value"
            value={newFieldValue}
            onChange={(e) => setNewFieldValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                handleAddField();
              }
            }}
          />
        </div>
        <Button
          type="button"
          variant="outline"
          size="icon"
          onClick={handleAddField}
          disabled={!newFieldKey.trim()}
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {fields.length === 0 && (
        <p className="rounded-md border border-dashed py-4 text-center text-sm text-muted-foreground">
          No additional fields. Add custom fields specific to this contact.
        </p>
      )}
    </div>
  );
}
