'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import _debounce from 'lodash/debounce';
import { Plus, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { DividerElement, FormElement, FormState, SpacerElement } from './types';

interface FormElementPropertiesProps {
  element: FormElement | null;
  propertyExcludedByElement?: string[];
  onChange: (property: keyof FormElement | string, value: any) => void;
  onStylePropertyChange?: (styleProperty: string, value: any) => void;
  setHeightToolbox?: (height: string) => void;
  formState: FormState;
  freezeFields?: string[];
}

export default function FormElementProperties({
  element,
  propertyExcludedByElement,
  onChange,
  onStylePropertyChange,
  setHeightToolbox,
  formState,
  freezeFields,
}: FormElementPropertiesProps) {
  const heightComponent = useRef<HTMLDivElement>(null);
  const [internalElement, setInternalElement] = useState<FormElement | null>(
    element
  );
  const [localValues, setLocalValues] = useState<Record<string, any>>({});

  const debouncedOnChange = useCallback(
    _debounce((property: keyof FormElement | string, value: any) => {
      onChange(property, value);
    }, 300),
    [onChange]
  );

  useEffect(() => {
    setInternalElement(element);
    // Clear local values when element changes
    setLocalValues({});
  }, [element]);

  if (!internalElement) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Properties</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            Select a form element to edit its properties
          </p>
        </CardContent>
      </Card>
    );
  }

  const handleChange = (field: string, value: any, name?: string) => {
    if (!internalElement) return;
    if (field === 'required' && name && freezeFields?.includes(name)) return;
    // Update local value immediately
    setLocalValues((prev) => ({ ...prev, [field]: value }));
    debouncedOnChange(field, value);
  };

  const handleStylePropertyChange = (field: string, value: string) => {
    if (!internalElement) return;
    const styleKey = 'styles.' + field;
    // Update local value immediately
    setLocalValues((prev) => ({ ...prev, [styleKey]: value }));
    debouncedOnChange(styleKey, value);
    if (onStylePropertyChange) {
      onStylePropertyChange(field, value);
    }
  };

  const handleOptionChange = (
    index: number,
    field: 'label' | 'value',
    value: string
  ) => {
    if (!element || !('options' in element)) return;

    // Update local state immediately for smooth UI
    setLocalValues((prev) => ({
      ...prev,
      [`options.${index}.${field}`]: value || null,
    }));

    // Create new options array from the latest element state
    const newOptions = [...element.options];
    newOptions[index] = { ...newOptions[index], [field]: value || null };

    // Use debounced onChange for actual state update
    debouncedOnChange('options', newOptions);
  };

  const handleAddOption = () => {
    if (!internalElement || !('options' in internalElement)) return;

    const newOptions = [
      ...(internalElement.options || []),
      {
        label: `Option ${internalElement.options.length + 1}`,
        value: `option${internalElement.options.length + 1}`,
      },
    ];

    onChange('options', newOptions);
  };

  const handleRemoveOption = (index: number) => {
    if (!internalElement || !('options' in internalElement)) return;

    const newOptions = internalElement.options.filter((_, i) => i !== index);
    onChange('options', newOptions);
  };
  useEffect(() => {
    if (heightComponent.current) {
      const height = heightComponent.current.offsetHeight;
      setHeightToolbox?.(`${height}px`);
    }
  }, []);

  return (
    <Card ref={heightComponent}>
      <CardHeader className="rounded-t-lg border-b bg-[#F1F5F9] p-0 px-4 pb-3 pt-4">
        <CardTitle>
          <span className="capitalize text-[#071F1D]">
            {internalElement?.type || 'N/A'}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 p-2">
        {propertyExcludedByElement?.includes(
          internalElement?.type || ''
        ) ? null : (
          <div className="space-y-2">
            <Label htmlFor="label">Label</Label>
            <Input
              id="label"
              value={localValues['label'] || internalElement.label || ''}
              onChange={(e) => handleChange('label', e.target.value)}
            />
          </div>
        )}

        {propertyExcludedByElement?.includes(
          internalElement?.type || ''
        ) ? null : (
          <>
            <div className="space-y-2">
              <Label htmlFor="description">Description (Optional)</Label>
              <Textarea
                id="description"
                value={
                  localValues['description'] || internalElement.description || ''
                }
                onChange={(e) => handleChange('description', e.target.value)}
                rows={2}
              />
            </div>
            <Separator className="my-4" />
          </>
        )}


        {/* Column span selection */}
        <div className="space-y-2">
          <Label htmlFor="colSpan">Column Span</Label>
          <div className="mt-1 grid grid-cols-2 gap-2">
            {Array.from({
              length: Number(formState?.settings?.gridColumns || 4),
            }).map((_, index) => (
              <Button
                key={index}
                type="button"
                variant={
                  localValues['colSpan'] === index + 1 ||
                    (!localValues['colSpan'] &&
                      internalElement.colSpan === index + 1)
                    ? 'default'
                    : 'outline'
                }
                onClick={() => handleChange('colSpan', index + 1)}
                className="w-full flex-1"
              >
                {index + 1} Column
              </Button>
            ))}
          </div>
        </div>

        {propertyExcludedByElement?.includes(
          internalElement?.type || ''
        ) ? null : (
          <div className="flex items-center justify-between">
            <Label htmlFor="required" className="flex flex-col space-y-1">
              <span>Required</span>
              <span className="font-normal leading-snug text-muted-foreground">
                Is this field mandatory?
              </span>
            </Label>
            <Switch
              id="required"
              checked={
                localValues['required'] || internalElement.required || false
              }
              onCheckedChange={(checked) => handleChange('required', checked, internalElement.name)}
            />
          </div>
        )}

        {(element?.type === 'text' ||
          element?.type === 'email' ||
          element?.type === 'password' ||
          element?.type === 'number' ||
          element?.type === 'textarea') && (
            <div className="space-y-2">
              <Label htmlFor="placeholder">Placeholder (Optional)</Label>
              <Input
                id="placeholder"
                value={
                  localValues['placeholder'] || internalElement.placeholder || ''
                }
                onChange={(e) => handleChange('placeholder', e.target.value)}
              />
            </div>
          )}

        {element?.type === 'textarea' && 'rows' in element && (
          <div className="space-y-2">
            <Label htmlFor="rows">Number of Rows</Label>
            <Input
              id="rows"
              type="number"
              value={localValues['rows'] || internalElement.rows || 3}
              onChange={(e) =>
                handleChange('rows', parseInt(e.target.value, 10) || 1)
              }
              min={1}
            />
          </div>
        )}

        {element?.type === 'checkbox' && 'defaultChecked' in element && (
          <div className="flex items-center justify-between">
            <Label htmlFor="defaultChecked" className="flex flex-col space-y-1">
              <span>Default Checked</span>
              <span className="font-normal leading-snug text-muted-foreground">
                Should this be checked by default?
              </span>
            </Label>
            <Switch
              id="defaultChecked"
              checked={
                localValues['defaultChecked'] ||
                internalElement.defaultChecked ||
                false
              }
              onCheckedChange={(checked) =>
                handleChange('defaultChecked', checked)
              }
            />
          </div>
        )}

        {element?.type === 'url' && 'defaultValue' in element && (
          <div className="space-y-2">
            <Label htmlFor="url">Url</Label>
            <Input
              id="url"
              type="string"
              value={
                localValues['defaultValue'] || internalElement.defaultValue || null
              }
              onChange={(e) => handleChange('defaultValue', e.target.value)}
            />
          </div>
        )}

        {(element?.type === 'select' || element?.type === 'radio') &&
          'options' in element && (
            <div className="space-y-3">
              <Separator />
              <div className="flex items-center justify-between">
                <Label>Options</Label>
                <Button variant="outline" size="sm" onClick={handleAddOption}>
                  <Plus className="mr-1 h-4 w-4" />
                  Option
                </Button>
              </div>

              <div className="space-y-3">
                {element.options.map((option, index) => (
                  <div key={index} className="flex items-center space-x-2">
                    <div className="flex-1">
                      <Input
                        placeholder="Label"
                        value={
                          localValues[`options.${index}.label`] ?? option.label
                        }
                        onChange={(e) =>
                          handleOptionChange(index, 'label', e.target.value)
                        }
                      />
                    </div>
                    <div className="flex-1">
                      <Input
                        placeholder="Value"
                        value={
                          localValues[`options.${index}.value`] ?? option.value
                        }
                        onChange={(e) =>
                          handleOptionChange(index, 'value', e.target.value)
                        }
                      />
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveOption(index)}
                      disabled={element.options.length <= 1}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

        {(element?.type === 'select' || element?.type === 'radio') &&
          'defaultValue' in element && (
            <div className="space-y-2">
              <Label htmlFor="defaultValue">Default Selected Value</Label>
              <Input
                id="defaultValue"
                value={
                  localValues['defaultValue'] ||
                  internalElement.defaultValue ||
                  null
                }
                onChange={(e) => handleChange('defaultValue', e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Enter the value (not label) of the option that should be
                selected by default
              </p>
            </div>
          )}

        {element?.type === 'spacer' && (
          <>
            <div className="space-y-2">
              <Label htmlFor="spacer-height">Height (e.g., 16px, 2rem)</Label>
              <Input
                id="spacer-height"
                value={
                  localValues['styles.height'] ||
                  (internalElement as SpacerElement).styles?.height ||
                  '16px'
                }
                onChange={(e) =>
                  handleStylePropertyChange('height', e.target.value)
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="spacer-bgColor">
                Background Color (e.g., #FF0000, transparent)
              </Label>
              <Input
                id="spacer-bgColor"
                value={
                  localValues['styles.backgroundColor'] ||
                  (internalElement as SpacerElement).styles?.backgroundColor ||
                  'transparent'
                }
                onChange={(e) =>
                  handleStylePropertyChange('backgroundColor', e.target.value)
                }
              />
            </div>
          </>
        )}

        {element?.type === 'divider' && (
          <>
            <div className="space-y-2">
              <Label htmlFor="divider-color">Color (e.g., #e0e0e0)</Label>
              <Input
                id="divider-color"
                value={
                  localValues['styles.color'] ||
                  (internalElement as DividerElement).styles?.color ||
                  '#e0e0e0'
                }
                onChange={(e) =>
                  handleStylePropertyChange('color', e.target.value)
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="divider-thickness">Thickness (e.g., 1px)</Label>
              <Input
                id="divider-thickness"
                value={
                  localValues['styles.thickness'] ||
                  (internalElement as DividerElement).styles?.thickness ||
                  '1px'
                }
                onChange={(e) =>
                  handleStylePropertyChange('thickness', e.target.value)
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="divider-width">Width (e.g., 100%, 50px)</Label>
              <Input
                id="divider-width"
                value={
                  localValues['styles.width'] ||
                  (internalElement as DividerElement).styles?.width ||
                  '100%'
                }
                onChange={(e) =>
                  handleStylePropertyChange('width', e.target.value)
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="divider-marginTop">Margin Top (e.g., 8px)</Label>
              <Input
                id="divider-marginTop"
                value={
                  localValues['styles.marginTop'] ||
                  (internalElement as DividerElement).styles?.marginTop ||
                  '8px'
                }
                onChange={(e) =>
                  handleStylePropertyChange('marginTop', e.target.value)
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="divider-marginBottom">
                Margin Bottom (e.g., 8px)
              </Label>
              <Input
                id="divider-marginBottom"
                value={
                  localValues['styles.marginBottom'] ||
                  (internalElement as DividerElement).styles?.marginBottom ||
                  '8px'
                }
                onChange={(e) =>
                  handleStylePropertyChange('marginBottom', e.target.value)
                }
              />
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
